/*
Copyright 2020 Cortex Labs, Inc.

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

package apisplitter

import (
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func UpdateAPI(apiConfig *userconfig.API, projectID string, force bool) (*spec.API, string, error) {
	prevVirtualService, err := getK8sResources(apiConfig)
	if err != nil {
		return nil, "", err
	}

	api := spec.GetAPISpec(apiConfig, projectID, "")
	if prevVirtualService == nil {
		if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err := applyK8sResources(api, prevVirtualService); err != nil {
			go deleteK8sResources(api.Name)
			return nil, "", err
		}
		err = operator.AddAPIToAPIGateway(*api.Networking.Endpoint, api.Networking.APIGateway)
		if err != nil {
			go deleteK8sResources(api.Name)
			return nil, "", err
		}
		return api, fmt.Sprintf("created %s", api.Name), nil
	}

	if !areVirtualServiceEqual(prevVirtualService, virtualServiceSpec(api)) {
		if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err := applyK8sResources(api, prevVirtualService); err != nil {
			return nil, "", err
		}
		if err := operator.UpdateAPIGatewayK8s(prevVirtualService, api); err != nil {
			return nil, "", err
		}
		return api, fmt.Sprintf("updated %s", api.Name), nil
	}
	return api, fmt.Sprintf("%s is up to date", api.Name), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	// best effort deletion, so don't handle error yet
	virtualService, vsErr := config.K8s.GetVirtualService(operator.K8sName(apiName))
	err := parallel.RunFirstErr(
		func() error {
			return vsErr
		},
		func() error {
			return deleteK8sResources(apiName)
		},
		func() error {
			if keepCache {
				return nil
			}
			// best effort deletion
			deleteS3Resources(apiName)
			return nil
		},
		// delete API from API Gateway
		func() error {
			err := operator.RemoveAPIFromAPIGatewayK8s(virtualService)
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func getK8sResources(apiConfig *userconfig.API) (*istioclientnetworking.VirtualService, error) {
	var virtualService *istioclientnetworking.VirtualService

	virtualService, err := config.K8s.GetVirtualService(operator.K8sName(apiConfig.Name))
	if err != nil {
		return nil, err
	}

	return virtualService, err
}

func applyK8sResources(api *spec.API, prevVirtualService *istioclientnetworking.VirtualService) error {
	return applyK8sVirtualService(api, prevVirtualService)
}

func applyK8sVirtualService(apiSplitter *spec.API, prevVirtualService *istioclientnetworking.VirtualService) error {
	newVirtualService := virtualServiceSpec(apiSplitter)

	if prevVirtualService == nil {
		_, err := config.K8s.CreateVirtualService(newVirtualService)
		return err
	}

	_, err := config.K8s.UpdateVirtualService(prevVirtualService, newVirtualService)
	return err
}

func getAPISplitterDestinations(apiSplitter *spec.API) []k8s.Destination {
	destinations := make([]k8s.Destination, len(apiSplitter.APIs))
	for i, api := range apiSplitter.APIs {
		destinations[i] = k8s.Destination{
			ServiceName: operator.K8sName(api.Name),
			Weight:      int32(api.Weight),
			Port:        uint32(_defaultPortInt32),
		}
	}
	return destinations
}

func deleteK8sResources(apiName string) error {
	_, err := config.K8s.DeleteVirtualService(operator.K8sName(apiName))
	return err
}

func deleteS3Resources(apiName string) error {
	prefix := filepath.Join("apis", apiName)
	return config.AWS.DeleteS3Dir(config.Cluster.Bucket, prefix, true)
}

func areVirtualServiceEqual(vs1, vs2 *istioclientnetworking.VirtualService) bool {
	return vs1.ObjectMeta.Name == vs2.ObjectMeta.Name &&
		reflect.DeepEqual(vs1.ObjectMeta.Labels, vs2.ObjectMeta.Labels) &&
		reflect.DeepEqual(vs1.ObjectMeta.Annotations, vs2.ObjectMeta.Annotations) &&
		reflect.DeepEqual(vs1.Spec.Http, vs2.Spec.Http) &&
		reflect.DeepEqual(vs1.Spec.Gateways, vs2.Spec.Gateways) &&
		reflect.DeepEqual(vs1.Spec.Hosts, vs2.Spec.Hosts)
}

// APIBaseURL returns BaseURL of the API without resource endpoint
func APIBaseURL(api *spec.API) (string, error) {
	if api.Networking.APIGateway == userconfig.PublicAPIGatewayType {
		return *config.Cluster.APIGateway.ApiEndpoint, nil
	}
	return operator.APILoadBalancerURL()
}
