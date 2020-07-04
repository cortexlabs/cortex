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
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _autoscalerCrons = make(map[string]cron.Cron) // apiName -> cron

func UpdateAPI(apiConfig *userconfig.API, projectID string, force bool) (*spec.API, string, error) {
	prevVirtualService, err := getK8sResources(apiConfig)
	if err != nil {
		return nil, "", err
	}
	fmt.Println("LOOOADBALANNNNCER")

	api := spec.GetAPISpec(apiConfig, projectID, "")
	fmt.Println(APIBaseURL(api))
	if prevVirtualService == nil {
		if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		err = operator.AddAPIToAPIGateway(*api.Networking.Endpoint, api.Networking.APIGateway)
		if err != nil {
			go deleteK8sResources(api.Name)
			return nil, "", err
		}
		if err := applyK8sResources(api, prevVirtualService); err != nil {
			go deleteK8sResources(api.Name)
			return nil, "", err
		}

		return api, fmt.Sprintf("creating %s", api.Name), nil
	}
	services, weight := getServicesWeightsTrafficSplitter(api)
	if err != nil {
		return nil, "", err
	}
	fmt.Println(prevVirtualService)
	fmt.Println(virtualServiceSpec(api, services, weight))
	if !areVirtualServiceEqual(prevVirtualService, virtualServiceSpec(api, services, weight)) {
		if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err := applyK8sResources(api, prevVirtualService); err != nil {
			return nil, "", err
		}
		if err := operator.UpdateAPIGatewayK8s(prevVirtualService, api); err != nil {
			return nil, "", err
		}
		return api, fmt.Sprintf("updating %s", api.Name), nil
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

	err := parallel.RunFirstErr(
		// func() error {
		// 	var err error
		// 	deployment, err = config.K8s.GetDeployment(operator.K8sName(apiConfig.Name))
		// 	return err
		// },
		// func() error {
		// 	var err error
		// 	service, err = config.K8s.GetService(operator.K8sName(apiConfig.Name))
		// 	return err
		// },
		func() error {
			var err error
			virtualService, err = config.K8s.GetVirtualService(operator.K8sName(apiConfig.Name))
			return err
		},
	)

	return virtualService, err
}

func applyK8sResources(api *spec.API, prevVirtualService *istioclientnetworking.VirtualService) error {
	return parallel.RunFirstErr(
		func() error {
			return applyK8sVirtualService(api, prevVirtualService)
		},
	)
}

func applyK8sVirtualService(trafficsplitter *spec.API, prevVirtualService *istioclientnetworking.VirtualService) error {

	services, weights := getServicesWeightsTrafficSplitter(trafficsplitter)

	newVirtualService := virtualServiceSpec(trafficsplitter, services, weights)

	if prevVirtualService == nil {
		_, err := config.K8s.CreateVirtualService(newVirtualService)
		return err
	}

	_, err := config.K8s.UpdateVirtualService(prevVirtualService, newVirtualService)
	return err
}

func getServicesWeightsTrafficSplitter(trafficsplitter *spec.API) ([]string, []int32) {
	services := []string{}
	weights := []int32{}
	for _, api := range trafficsplitter.APIs {
		services = append(services, "api-"+api.Name)
		weights = append(weights, int32(api.Weight))
	}
	return services, weights

}

func deleteK8sResources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			_, err := config.K8s.DeleteVirtualService(apiName)
			return err
		},
	)
}

func deleteS3Resources(apiName string) error {
	prefix := filepath.Join("apis", apiName)
	return config.AWS.DeleteS3Dir(config.Cluster.Bucket, prefix, true)
}

func IsAPIUpdating(apiName string) (bool, error) {
	deployment, err := config.K8s.GetDeployment(operator.K8sName(apiName))
	if err != nil {
		return false, err
	}

	return isAPIUpdating(deployment)
}

// returns true if min_replicas are not ready and no updated replicas have errored
func isAPIUpdating(deployment *kapps.Deployment) (bool, error) {
	pods, err := config.K8s.ListPodsByLabel("apiName", deployment.Labels["apiName"])
	if err != nil {
		return false, err
	}

	replicaCounts := getReplicaCounts(deployment, pods)

	autoscalingSpec, err := userconfig.AutoscalingFromAnnotations(deployment)
	if err != nil {
		return false, err
	}

	if replicaCounts.Updated.Ready < autoscalingSpec.MinReplicas && replicaCounts.Updated.TotalFailed() == 0 {
		return true, nil
	}

	return false, nil
}

func isPodSpecLatest(deployment *kapps.Deployment, pod *kcore.Pod) bool {
	return k8s.PodComputesEqual(&deployment.Spec.Template.Spec, &pod.Spec) &&
		deployment.Spec.Template.Labels["apiName"] == pod.Labels["apiName"] &&
		deployment.Spec.Template.Labels["apiID"] == pod.Labels["apiID"] &&
		deployment.Spec.Template.Labels["deploymentID"] == pod.Labels["deploymentID"]
}

func areVirtualServiceEqual(vs1, vs2 *istioclientnetworking.VirtualService) bool {
	return reflect.DeepEqual(vs1, vs2)
}

func doCortexAnnotationsMatch(obj1, obj2 kmeta.Object) bool {
	cortexAnnotations1 := extractCortexAnnotations(obj1)
	cortexAnnotations2 := extractCortexAnnotations(obj2)
	return maps.StrMapsEqual(cortexAnnotations1, cortexAnnotations2)
}

func extractCortexAnnotations(obj kmeta.Object) map[string]string {
	cortexAnnotations := make(map[string]string)
	for key, value := range obj.GetAnnotations() {
		if strings.Contains(key, "cortex.dev/") {
			cortexAnnotations[key] = value
		}
	}
	return cortexAnnotations
}

func IsAPIDeployed(apiName string) (bool, error) {
	deployment, err := config.K8s.GetDeployment(operator.K8sName(apiName))
	if err != nil {
		return false, err
	}
	return deployment != nil, nil
}

// APIBaseURL returns BaseURL of the API without resource endpoint
func APIBaseURL(api *spec.API) (string, error) {
	if api.Networking.APIGateway == userconfig.PublicAPIGatewayType {
		return *config.Cluster.APIGateway.ApiEndpoint, nil
	}
	return operator.APILoadBalancerURL()
}
