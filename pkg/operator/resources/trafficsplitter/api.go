/*
Copyright 2022 Cortex Labs, Inc.

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

package trafficsplitter

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
)

// UpdateAPI creates or updates a traffic splitter API kind
func UpdateAPI(apiConfig *userconfig.API) (*spec.API, string, error) {
	prevVirtualService, err := config.K8s.GetVirtualService(workloads.K8sName(apiConfig.Name))
	if err != nil {
		return nil, "", err
	}

	initialDeploymentTime := time.Now().UnixNano()
	if prevVirtualService != nil && prevVirtualService.Labels["initialDeploymentTime"] != "" {
		var err error
		initialDeploymentTime, err = k8s.ParseInt64Label(prevVirtualService, "initialDeploymentTime")
		if err != nil {
			return nil, "", err
		}
	}

	api := spec.GetAPISpec(apiConfig, initialDeploymentTime, "", config.ClusterConfig.ClusterUID)
	if prevVirtualService == nil {
		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "failed to upload api spec")
		}

		if err := applyK8sVirtualService(api, prevVirtualService); err != nil {
			routines.RunWithPanicHandler(func() {
				_ = deleteK8sResources(api.Name)
			})
			return nil, "", err
		}

		return api, fmt.Sprintf("created %s", api.Resource.UserString()), nil
	}

	if prevVirtualService.Labels["specID"] != api.SpecID {
		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "failed to upload api spec")
		}

		if err := applyK8sVirtualService(api, prevVirtualService); err != nil {
			return nil, "", err
		}

		return api, fmt.Sprintf("updated %s", api.Resource.UserString()), nil
	}

	return api, fmt.Sprintf("%s is up to date", api.Resource.UserString()), nil
}

// DeleteAPI deletes all the resources related to a given traffic splitter API
func DeleteAPI(apiName string, keepCache bool) error {
	err := parallel.RunFirstErr(
		func() error {
			return deleteK8sResources(apiName)
		},
		func() error {
			if keepCache {
				return nil
			}
			// best effort deletion
			_ = deleteS3Resources(apiName)
			return nil
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func applyK8sVirtualService(trafficSplitter *spec.API, prevVirtualService *istioclientnetworking.VirtualService) error {
	newVirtualService := virtualServiceSpec(trafficSplitter)

	if prevVirtualService == nil {
		_, err := config.K8s.CreateVirtualService(newVirtualService)
		return err
	}

	_, err := config.K8s.UpdateVirtualService(prevVirtualService, newVirtualService)
	return err
}

func getTrafficSplitterDestinations(trafficSplitter *spec.API) []k8s.Destination {
	destinations := make([]k8s.Destination, len(trafficSplitter.APIs))
	for i, api := range trafficSplitter.APIs {
		destinations[i] = k8s.Destination{
			ServiceName: workloads.K8sName(api.Name),
			Weight:      api.Weight,
			Port:        uint32(consts.ProxyPortInt32),
			Shadow:      api.Shadow,
		}
	}
	return destinations
}

// GetAllAPIs returns a list of metadata, in the form of schema.APIResponse, about all the created traffic splitter APIs
func GetAllAPIs(virtualServices []*istioclientnetworking.VirtualService) ([]schema.APIResponse, error) {
	var trafficSplitters []schema.APIResponse
	for i := range virtualServices {
		apiName := virtualServices[i].Labels["apiName"]

		metadata, err := spec.MetadataFromVirtualService(virtualServices[i])
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("api %s", apiName))
		}

		if metadata.Kind != userconfig.TrafficSplitterKind {
			continue
		}

		targets, err := userconfig.TrafficSplitterTargetsFromAnnotations(virtualServices[i])
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("api %s", apiName))
		}

		trafficSplitters = append(trafficSplitters, schema.APIResponse{
			Metadata:                  metadata,
			NumTrafficSplitterTargets: pointer.Int32(targets),
		})
	}

	return trafficSplitters, nil
}

// GetAPIByName retrieves the metadata, in the form of schema.APIResponse, of a single traffic splitter API
func GetAPIByName(deployedResource *operator.DeployedResource) ([]schema.APIResponse, error) {
	metadata, err := spec.MetadataFromVirtualService(deployedResource.VirtualService)
	if err != nil {
		return nil, err
	}

	api, err := operator.DownloadAPISpec(deployedResource.Name, metadata.APIID)
	if err != nil {
		return nil, err
	}

	endpoint, err := operator.APIEndpoint(api)
	if err != nil {
		return nil, err
	}

	return []schema.APIResponse{
		{
			Spec:     api,
			Metadata: metadata,
			Endpoint: &endpoint,
		},
	}, nil
}

func deleteK8sResources(apiName string) error {
	_, err := config.K8s.DeleteVirtualService(workloads.K8sName(apiName))
	return err
}

func deleteS3Resources(apiName string) error {
	prefix := filepath.Join(config.ClusterConfig.ClusterUID, "apis", apiName)
	return config.AWS.DeleteS3Dir(config.ClusterConfig.Bucket, prefix, true)
}
