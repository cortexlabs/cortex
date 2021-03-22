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

package trafficsplitter

import (
	"fmt"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
)

// UpdateAPI creates or updates a traffic splitter API kind
func UpdateAPI(apiConfig *userconfig.API) (*spec.API, string, error) {
	prevVirtualService, err := config.K8s.GetVirtualService(operator.K8sName(apiConfig.Name))
	if err != nil {
		return nil, "", err
	}

	api := spec.GetAPISpec(apiConfig, "", "", config.ClusterName())
	if prevVirtualService == nil {
		if err := config.UploadJSONToBucket(api, api.Key); err != nil {
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
		if err := config.UploadJSONToBucket(api, api.Key); err != nil {
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
			ServiceName: operator.K8sName(api.Name),
			Weight:      api.Weight,
			Port:        uint32(_defaultPortInt32),
			Shadow:      api.Shadow,
		}
	}
	return destinations
}

// GetAllAPIs returns a list of metadata, in the form of schema.APIResponse, about all the created traffic splitter APIs
func GetAllAPIs(virtualServices []istioclientnetworking.VirtualService) ([]schema.APIResponse, error) {
	var (
		apiNames         []string
		apiIDs           []string
		trafficSplitters []schema.APIResponse
	)

	for _, virtualService := range virtualServices {
		if virtualService.Labels["apiKind"] == userconfig.TrafficSplitterKind.String() {
			apiNames = append(apiNames, virtualService.Labels["apiName"])
			apiIDs = append(apiIDs, virtualService.Labels["apiID"])
		}
	}

	apis, err := operator.DownloadAPISpecs(apiNames, apiIDs)
	if err != nil {
		return nil, err
	}

	for i := range apis {
		trafficSplitter := apis[i]
		endpoint, err := operator.APIEndpoint(&trafficSplitter)
		if err != nil {
			return nil, err
		}

		trafficSplitters = append(trafficSplitters, schema.APIResponse{
			Spec:     trafficSplitter,
			Endpoint: endpoint,
		})
	}

	return trafficSplitters, nil
}

// GetAPIByName retrieves the metadata, in the form of schema.APIResponse, of a single traffic splitter API
func GetAPIByName(deployedResource *operator.DeployedResource) ([]schema.APIResponse, error) {
	api, err := operator.DownloadAPISpec(deployedResource.Name, deployedResource.VirtualService.Labels["apiID"])
	if err != nil {
		return nil, err
	}

	endpoint, err := operator.APIEndpoint(api)
	if err != nil {
		return nil, err
	}

	return []schema.APIResponse{
		{
			Spec:     *api,
			Endpoint: endpoint,
		},
	}, nil
}

func deleteK8sResources(apiName string) error {
	_, err := config.K8s.DeleteVirtualService(operator.K8sName(apiName))
	return err
}

func deleteS3Resources(apiName string) error {
	prefix := filepath.Join(config.ClusterName(), "apis", apiName)
	return config.DeleteBucketDir(prefix, true)
}
