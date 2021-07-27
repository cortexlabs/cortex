/*
Copyright 2021 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package realtimeapi

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	serverless "github.com/cortexlabs/cortex/pkg/crds/apis/serverless/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const _realtimeDashboardUID = "realtimeapi"

func generateDeploymentID() string {
	return k8s.RandomName()[:10]
}

func UpdateAPI(apiConfig *userconfig.API, force bool) (*spec.API, string, error) {
	ctx := context.Background()
	var api serverless.RealtimeAPI
	key := client.ObjectKey{Namespace: consts.DefaultNamespace, Name: apiConfig.Name}

	err := config.K8s.Get(ctx, key, &api)
	if err != nil {
		if kerrors.IsNotFound(err) {
			if kerrors.IsNotFound(err) {
				api = K8sResourceFromAPIConfig(*apiConfig)
				if err = config.K8s.Create(ctx, &api); err != nil {
					return nil, "", errors.Wrap(err, "failed to create realtime api resource")
				}

				apiSpec := &spec.API{
					API:                   apiConfig,
					ID:                    api.Annotations["cortex.dev/api-id"],
					SpecID:                api.Annotations["cortex.dev/spec-id"],
					PodID:                 api.Annotations["cortex.dev/pod-id"],
					DeploymentID:          api.Annotations["cortex.dev/deployment-id"],
					Key:                   spec.Key(apiConfig.Name, api.Annotations["cortex.dev/api-id"], config.ClusterConfig.ClusterUID),
					InitialDeploymentTime: api.CreationTimestamp.Unix(),
					LastUpdated:           api.CreationTimestamp.Unix(),
					MetadataRoot:          spec.MetadataRoot(apiConfig.Name, config.ClusterConfig.ClusterUID),
				}

				if err := config.AWS.UploadJSONToS3(apiSpec, config.ClusterConfig.Bucket, apiSpec.Key); err != nil {
					return nil, "", errors.Wrap(err, "failed to upload api spec")
				}

				return apiSpec, fmt.Sprintf("creating %s", apiConfig.Resource.UserString()), nil
			}
		}
		return nil, "", errors.Wrap(err, "failed to get realtime api resource")
	}

	desiredAPI := K8sResourceFromAPIConfig(*apiConfig)

	apiSpec := &spec.API{
		API:                   apiConfig,
		ID:                    desiredAPI.Annotations["cortex.dev/api-id"],
		SpecID:                desiredAPI.Annotations["cortex.dev/spec-id"],
		PodID:                 desiredAPI.Annotations["cortex.dev/pod-id"],
		DeploymentID:          desiredAPI.Annotations["cortex.dev/deployment-id"],
		Key:                   spec.Key(apiConfig.Name, desiredAPI.Annotations["cortex.dev/api-id"], config.ClusterConfig.ClusterUID),
		InitialDeploymentTime: api.CreationTimestamp.Unix(),
		MetadataRoot:          spec.MetadataRoot(apiConfig.Name, config.ClusterConfig.ClusterUID),
	}

	if !reflect.DeepEqual(api.Spec, desiredAPI.Spec) || force {
		api.Spec = desiredAPI.Spec
		api.Annotations = maps.MergeStrMapsString(api.Annotations, desiredAPI.Annotations)

		lastUpdated := time.Now().Unix()
		api.Annotations["cortex.dev/last-updated"] = s.Int64(lastUpdated)
		apiSpec.LastUpdated = lastUpdated

		if err = config.K8s.Update(ctx, &api); err != nil {
			return nil, "", errors.Wrap(err, "failed to update realtime api resource")
		}

		if err := config.AWS.UploadJSONToS3(apiSpec, config.ClusterConfig.Bucket, apiSpec.Key); err != nil {
			return nil, "", errors.Wrap(err, "failed to upload api spec")
		}

		return apiSpec, fmt.Sprintf("updating %s", apiConfig.Resource.UserString()), nil
	}

	return apiSpec, fmt.Sprintf("%s is up to date", apiConfig.Resource.UserString()), nil
}

func RefreshAPI(apiName string) (string, error) {
	ctx := context.Background()
	api := serverless.RealtimeAPI{
		ObjectMeta: kmeta.ObjectMeta{
			Namespace: consts.DefaultNamespace,
			Name:      apiName,
		},
	}

	// slashes are encoded as ~1 in the json patch
	patch := []byte(fmt.Sprintf(
		"[{\"op\": \"replace\", \"path\": \"/metadata/annotations/cortex.dev~1deployment-id\", \"value\": \"%s\" }]",
		generateDeploymentID()))
	if err := config.K8s.Patch(ctx, &api, client.RawPatch(ktypes.JSONPatchType, patch)); err != nil {
		return "", errors.Wrap(err, "failed to get realtime api resource")
	}

	apiResource := userconfig.Resource{
		Name: apiName,
		Kind: userconfig.RealtimeAPIKind,
	}

	return fmt.Sprintf("updating %s", apiResource.UserString()), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	return parallel.RunFirstErr(
		func() error {
			ctx := context.Background()
			api := serverless.RealtimeAPI{
				ObjectMeta: kmeta.ObjectMeta{
					Name:      apiName,
					Namespace: consts.DefaultNamespace,
				},
			}
			if err := config.K8s.Delete(ctx, &api); err != nil {
				return errors.Wrap(err, "failed to delete realtime api resource")
			}
			return nil
		},
		func() error {
			if keepCache {
				return nil
			}
			// best effort deletion, swallow errors because there could be weird error messages
			_ = deleteBucketResources(apiName)
			return nil
		},
	)
}

func GetAllAPIs() ([]schema.APIResponse, error) {
	ctx := context.Background()
	apis := serverless.RealtimeAPIList{}
	if err := config.K8s.List(ctx, &apis); err != nil {
		return nil, errors.Wrap(err, "failed to list realtime api resources")
	}

	apiNames := make([]string, len(apis.Items))
	apiIDs := make([]string, len(apis.Items))
	for i, api := range apis.Items {
		apiNames[i] = api.Name
		apiIDs[i] = api.Annotations["cortex.dev/api-id"]
	}

	apiSpecs, err := operator.DownloadAPISpecs(apiNames, apiIDs)
	if err != nil {
		return nil, err
	}

	realtimeAPIs := make([]schema.APIResponse, len(apis.Items))
	for i := range apis.Items {
		api := apis.Items[i]
		api.Status.ReplicaCounts.Requested = api.Spec.Pod.Replicas

		realtimeAPIs[i] = schema.APIResponse{
			Spec: apiSpecs[i],
			Status: &status.Status{
				APIName:       api.Name,
				APIID:         api.Annotations["cortex.dev/api-id"],
				Code:          api.Status.Status,
				ReplicaCounts: api.Status.ReplicaCounts,
			},
			Endpoint: api.Status.Endpoint,
		}
	}

	return realtimeAPIs, nil
}

func GetAPIByName(apiName string) ([]schema.APIResponse, error) {
	ctx := context.Background()

	api := serverless.RealtimeAPI{}
	key := client.ObjectKey{Namespace: consts.DefaultNamespace, Name: apiName}
	if err := config.K8s.Get(ctx, key, &api); err != nil {
		return nil, errors.Wrap(err, "failed to get realtime api resource")
	}

	apiSpec, err := operator.DownloadAPISpec(api.Name, api.Annotations["cortex.dev/api-id"])
	if err != nil {
		return nil, err
	}

	dashboardURL := pointer.String(getDashboardURL(api.Name))
	api.Status.ReplicaCounts.Requested = api.Spec.Pod.Replicas

	return []schema.APIResponse{
		{
			Spec: *apiSpec,
			Status: &status.Status{
				APIName:       api.Name,
				APIID:         api.Annotations["cortex.dev/api-id"],
				Code:          api.Status.Status,
				ReplicaCounts: api.Status.ReplicaCounts,
			},
			Endpoint:     api.Status.Endpoint,
			DashboardURL: dashboardURL,
		},
	}, nil
}

func getDashboardURL(apiName string) string {
	loadBalancerURL, err := operator.LoadBalancerURL()
	if err != nil {
		return ""
	}

	dashboardURL := fmt.Sprintf(
		"%s/dashboard/d/%s/realtimeapi?orgId=1&refresh=30s&var-api_name=%s",
		loadBalancerURL, _realtimeDashboardUID, apiName,
	)

	return dashboardURL
}

// K8sResourceFromAPIConfig converts a cortex API config into a realtime API CRD resource
func K8sResourceFromAPIConfig(apiConfig userconfig.API) serverless.RealtimeAPI {
	containers := make([]serverless.ContainerSpec, len(apiConfig.Pod.Containers))
	for i := range apiConfig.Pod.Containers {
		container := apiConfig.Pod.Containers[i]
		var env []kcore.EnvVar
		for k, v := range container.Env {
			env = append(env, kcore.EnvVar{
				Name:  k,
				Value: v,
			})
		}

		var compute *serverless.ComputeSpec
		if container.Compute != nil {
			var cpu *kresource.Quantity
			if container.Compute.CPU != nil {
				cpu = &container.Compute.CPU.Quantity
			}
			var mem *kresource.Quantity
			if container.Compute.Mem != nil {
				mem = &container.Compute.Mem.Quantity
			}
			var shm *kresource.Quantity
			if container.Compute.Shm != nil {
				shm = &container.Compute.Shm.Quantity
			}

			compute = &serverless.ComputeSpec{
				CPU: cpu,
				GPU: container.Compute.GPU,
				Inf: container.Compute.Inf,
				Mem: mem,
				Shm: shm,
			}
		}

		containers[i] = serverless.ContainerSpec{
			Name:           container.Name,
			Image:          container.Image,
			Command:        container.Command,
			Args:           container.Args,
			Env:            env,
			Compute:        compute,
			ReadinessProbe: workloads.GetProbeSpec(container.ReadinessProbe),
			LivenessProbe:  workloads.GetProbeSpec(container.LivenessProbe),
		}
	}

	api := serverless.RealtimeAPI{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      apiConfig.Name,
			Namespace: consts.DefaultNamespace,
		},
		Spec: serverless.RealtimeAPISpec{
			Pod: serverless.PodSpec{
				Port:           *apiConfig.Pod.Port,
				MaxConcurrency: int32(apiConfig.Pod.MaxConcurrency),
				MaxQueueLength: int32(apiConfig.Pod.MaxQueueLength),
				Replicas:       apiConfig.Autoscaling.InitReplicas,
				Containers:     containers,
			},
			Autoscaling: serverless.AutoscalingSpec{
				MinReplicas:                  apiConfig.Autoscaling.MinReplicas,
				MaxReplicas:                  apiConfig.Autoscaling.MaxReplicas,
				TargetInFlight:               fmt.Sprintf("%f", *apiConfig.Autoscaling.TargetInFlight),
				Window:                       kmeta.Duration{Duration: apiConfig.Autoscaling.Window},
				DownscaleStabilizationPeriod: kmeta.Duration{Duration: apiConfig.Autoscaling.DownscaleStabilizationPeriod},
				UpscaleStabilizationPeriod:   kmeta.Duration{Duration: apiConfig.Autoscaling.UpscaleStabilizationPeriod},
				MaxDownscaleFactor:           fmt.Sprintf("%f", apiConfig.Autoscaling.MaxDownscaleFactor),
				MaxUpscaleFactor:             fmt.Sprintf("%f", apiConfig.Autoscaling.MaxUpscaleFactor),
				DownscaleTolerance:           fmt.Sprintf("%f", apiConfig.Autoscaling.DownscaleTolerance),
				UpscaleTolerance:             fmt.Sprintf("%f", apiConfig.Autoscaling.UpscaleTolerance),
			},
			NodeGroups: apiConfig.NodeGroups,
			UpdateStrategy: serverless.UpdateStrategySpec{
				MaxSurge:       intstr.FromString(apiConfig.UpdateStrategy.MaxSurge),
				MaxUnavailable: intstr.FromString(apiConfig.UpdateStrategy.MaxUnavailable),
			},
			Networking: serverless.NetworkingSpec{
				Endpoint: *apiConfig.Networking.Endpoint,
			},
		},
	}

	deploymentID, podID, specID, apiID := api.GetOrCreateAPIIDs()
	api.Annotations = map[string]string{
		"cortex.dev/deployment-id": deploymentID,
		"cortex.dev/spec-id":       specID,
		"cortex.dev/pod-id":        podID,
		"cortex.dev/api-id":        apiID,
	}

	return api
}

func deleteBucketResources(apiName string) error {
	prefix := filepath.Join(config.ClusterConfig.ClusterUID, "apis", apiName)
	return config.AWS.DeleteS3Dir(config.ClusterConfig.Bucket, prefix, true)
}
