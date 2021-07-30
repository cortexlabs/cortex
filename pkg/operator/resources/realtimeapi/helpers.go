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
	"fmt"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/crds/apis/serverless/v1alpha1"
	serverless "github.com/cortexlabs/cortex/pkg/crds/apis/serverless/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const _realtimeDashboardUID = "realtimeapi"

func generateDeploymentID() string {
	return k8s.RandomName()[:10]
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

// k8sResourceFromAPIConfig converts a cortex API config into a realtime API CRD resource
func k8sResourceFromAPIConfig(apiConfig userconfig.API, prevAPI *serverless.RealtimeAPI) v1alpha1.RealtimeAPI {
	containers := make([]v1alpha1.ContainerSpec, len(apiConfig.Pod.Containers))
	for i := range apiConfig.Pod.Containers {
		container := apiConfig.Pod.Containers[i]
		var env []v1.EnvVar
		for k, v := range container.Env {
			env = append(env, v1.EnvVar{
				Name:  k,
				Value: v,
			})
		}

		var compute *v1alpha1.ComputeSpec
		if container.Compute != nil {
			var cpu *resource.Quantity
			if container.Compute.CPU != nil {
				cpu = &container.Compute.CPU.Quantity
			}
			var mem *resource.Quantity
			if container.Compute.Mem != nil {
				mem = &container.Compute.Mem.Quantity
			}
			var shm *resource.Quantity
			if container.Compute.Shm != nil {
				shm = &container.Compute.Shm.Quantity
			}

			compute = &v1alpha1.ComputeSpec{
				CPU: cpu,
				GPU: container.Compute.GPU,
				Inf: container.Compute.Inf,
				Mem: mem,
				Shm: shm,
			}
		}

		containers[i] = v1alpha1.ContainerSpec{
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

	api := v1alpha1.RealtimeAPI{
		ObjectMeta: v12.ObjectMeta{
			Name:      apiConfig.Name,
			Namespace: consts.DefaultNamespace,
		},
		Spec: v1alpha1.RealtimeAPISpec{
			Replicas: apiConfig.Autoscaling.InitReplicas,
			Pod: v1alpha1.PodSpec{
				Port:           *apiConfig.Pod.Port,
				MaxConcurrency: int32(apiConfig.Pod.MaxConcurrency),
				MaxQueueLength: int32(apiConfig.Pod.MaxQueueLength),
				Containers:     containers,
			},
			Autoscaling: v1alpha1.AutoscalingSpec{
				InitReplicas:                 apiConfig.Autoscaling.InitReplicas,
				MinReplicas:                  apiConfig.Autoscaling.MinReplicas,
				MaxReplicas:                  apiConfig.Autoscaling.MaxReplicas,
				TargetInFlight:               fmt.Sprintf("%f", *apiConfig.Autoscaling.TargetInFlight),
				Window:                       v12.Duration{Duration: apiConfig.Autoscaling.Window},
				DownscaleStabilizationPeriod: v12.Duration{Duration: apiConfig.Autoscaling.DownscaleStabilizationPeriod},
				UpscaleStabilizationPeriod:   v12.Duration{Duration: apiConfig.Autoscaling.UpscaleStabilizationPeriod},
				MaxDownscaleFactor:           fmt.Sprintf("%f", apiConfig.Autoscaling.MaxDownscaleFactor),
				MaxUpscaleFactor:             fmt.Sprintf("%f", apiConfig.Autoscaling.MaxUpscaleFactor),
				DownscaleTolerance:           fmt.Sprintf("%f", apiConfig.Autoscaling.DownscaleTolerance),
				UpscaleTolerance:             fmt.Sprintf("%f", apiConfig.Autoscaling.UpscaleTolerance),
			},
			NodeGroups: apiConfig.NodeGroups,
			UpdateStrategy: v1alpha1.UpdateStrategySpec{
				MaxSurge:       intstr.FromString(apiConfig.UpdateStrategy.MaxSurge),
				MaxUnavailable: intstr.FromString(apiConfig.UpdateStrategy.MaxUnavailable),
			},
			Networking: v1alpha1.NetworkingSpec{
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

	if prevAPI != nil {
		// we should keep the existing number of replicas instead of init_replicas
		api.Spec.Replicas = prevAPI.Spec.Replicas
		if prevDeployID := prevAPI.Annotations["cortex.dev/deployment-id"]; prevDeployID != "" {
			api.Annotations["cortex.dev/deployment-id"] = prevDeployID
		}
	}

	return api
}

func deleteBucketResources(apiName string) error {
	prefix := filepath.Join(config.ClusterConfig.ClusterUID, "apis", apiName)
	return config.AWS.DeleteS3Dir(config.ClusterConfig.Bucket, prefix, true)
}

func metadataFromRealtimeAPI(sv *v1alpha1.RealtimeAPI) (*spec.Metadata, error) {
	lastUpdated, err := spec.TimeFromAPIID(sv.Annotations["cortex.dev/api-id"])
	if err != nil {
		return nil, err
	}
	return &spec.Metadata{
		Resource: &userconfig.Resource{
			Name: sv.Name,
			Kind: userconfig.RealtimeAPIKind,
		},
		APIID:        sv.Annotations["cortex.dev/api-id"],
		DeploymentID: sv.Annotations["cortex.dev/deployment-id"],
		LastUpdated:  lastUpdated.Unix(),
	}, nil
}

func getReplicaCounts(pods []v1.Pod, metadata *spec.Metadata) status.ReplicaCounts {
	counts := status.ReplicaCounts{}

	for i := range pods {
		pod := pods[i]
		if pod.Labels["apiName"] != metadata.Name {
			continue
		}
		addPodToReplicaCounts(&pods[i], metadata, &counts)
	}

	return counts
}

func addPodToReplicaCounts(pod *v1.Pod, metadata *spec.Metadata, counts *status.ReplicaCounts) {
	latest := false
	if isPodSpecLatest(pod, metadata) {
		latest = true
	}

	isPodReady := k8s.IsPodReady(pod)
	if latest && isPodReady {
		counts.Ready++
		return
	} else if !latest && isPodReady {
		counts.ReadyOutOfDate++
		return
	}

	podStatus := k8s.GetPodStatus(pod)

	if podStatus == k8s.PodStatusTerminating {
		counts.Terminating++
		return
	}

	if !latest {
		return
	}

	switch podStatus {
	case k8s.PodStatusPending:
		counts.Pending++
	case k8s.PodStatusStalled:
		counts.Stalled++
	case k8s.PodStatusCreating:
		counts.Creating++
	case k8s.PodStatusReady:
		counts.Ready++
	case k8s.PodStatusNotReady:
		counts.NotReady++
	case k8s.PodStatusErrImagePull:
		counts.ErrImagePull++
	case k8s.PodStatusFailed:
		counts.Failed++
	case k8s.PodStatusKilled:
		counts.Killed++
	case k8s.PodStatusKilledOOM:
		counts.KilledOOM++
	case k8s.PodStatusUnknown:
		counts.Unknown++
	}
}

func isPodSpecLatest(pod *v1.Pod, metadata *spec.Metadata) bool {
	return metadata.APIID == pod.Labels["apiID"]
}
