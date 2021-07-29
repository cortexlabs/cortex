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
	"sort"
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	serverless "github.com/cortexlabs/cortex/pkg/crds/apis/serverless/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/google/go-cmp/cmp"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateAPI(apiConfig *userconfig.API, force bool) (*spec.API, string, error) {
	ctx := context.Background()
	var api serverless.RealtimeAPI
	key := client.ObjectKey{Namespace: consts.DefaultNamespace, Name: apiConfig.Name}

	err := config.K8s.Get(ctx, key, &api)
	if err != nil {
		if kerrors.IsNotFound(err) {
			if kerrors.IsNotFound(err) {
				api = k8sResourceFromAPIConfig(*apiConfig, nil)
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

	desiredAPI := k8sResourceFromAPIConfig(*apiConfig, &api)

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

	if !cmp.Equal(api.Spec, desiredAPI.Spec) || force {
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
	for i, api := range apis.Items {
		apiNames[i] = api.Name
	}

	realtimeAPIs := make([]schema.APIResponse, len(apis.Items))
	mappedRealtimeAPIs := map[string]schema.APIResponse{}
	for i := range apis.Items {
		api := apis.Items[i]

		metadata, err := metadataFromRealtimeAPI(&api)
		if err != nil {
			return nil, err
		}

		mappedRealtimeAPIs[api.Name] = schema.APIResponse{
			Metadata: metadata,
			Status: &status.Status{
				Ready:     api.Status.Ready,
				Requested: api.Status.Requested,
				UpToDate:  api.Status.UpToDate,
			},
		}
	}

	sort.Strings(apiNames)
	for i, apiName := range apiNames {
		realtimeAPIs[i] = mappedRealtimeAPIs[apiName]
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

	metadata, err := metadataFromRealtimeAPI(&api)
	if err != nil {
		return nil, err
	}

	apiSpec, err := operator.DownloadAPISpec(api.Name, api.Annotations["cortex.dev/api-id"])
	if err != nil {
		return nil, err
	}

	dashboardURL := pointer.String(getDashboardURL(api.Name))

	return []schema.APIResponse{
		{
			Spec:     apiSpec,
			Metadata: metadata,
			Status: &status.Status{
				Ready:     api.Status.Ready,
				Requested: api.Status.Requested,
				UpToDate:  api.Status.UpToDate,
			},
			Endpoint:     &api.Status.Endpoint,
			DashboardURL: dashboardURL,
		},
	}, nil
}

func DescribeAPIByName(apiName string) ([]schema.APIResponse, error) {
	ctx := context.Background()

	api := serverless.RealtimeAPI{}
	key := client.ObjectKey{Namespace: consts.DefaultNamespace, Name: apiName}
	if err := config.K8s.Get(ctx, key, &api); err != nil {
		return nil, errors.Wrap(err, "failed to get realtime api resource")
	}

	metadata, err := metadataFromRealtimeAPI(&api)
	if err != nil {
		return nil, err
	}

	var podList kcore.PodList
	if err = config.K8s.List(ctx, &podList, client.MatchingLabels{
		"apiName": metadata.Name,
		"apiKind": userconfig.RealtimeAPIKind.String(),
	}); err != nil {
		return nil, err
	}

	replicaCounts := getReplicaCounts(podList.Items, metadata)
	replicaCounts.Requested = api.Status.Requested
	replicaCounts.UpToDate = api.Status.UpToDate

	dashboardURL := pointer.String(getDashboardURL(api.Name))

	return []schema.APIResponse{
		{
			Metadata:      metadata,
			ReplicaCounts: &replicaCounts,
			Endpoint:      &api.Status.Endpoint,
			DashboardURL:  dashboardURL,
		},
	}, nil
}
