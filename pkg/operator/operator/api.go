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

package operator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _autoscalerCrons = make(map[string]cron.Cron) // apiName -> cron

func UpdateAPI(apiConfig *userconfig.API, projectID string, force bool) (*spec.API, string, error) {
	prevDeployment, prevService, prevVirtualService, err := getK8sResources(apiConfig)
	if err != nil {
		return nil, "", err
	}

	deploymentID := k8s.RandomName()
	if prevDeployment != nil && prevDeployment.Labels["deploymentID"] != "" {
		deploymentID = prevDeployment.Labels["deploymentID"]
	}

	api := spec.GetAPISpec(apiConfig, projectID, deploymentID)

	if prevDeployment == nil {
		if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err := applyK8sResources(api, prevDeployment, prevService, prevVirtualService); err != nil {
			go deleteK8sResources(api.Name)
			return nil, "", err
		}
		err = addAPIToAPIGateway(*api.Endpoint, api.Networking.APIGateway)
		if err != nil {
			go deleteK8sResources(api.Name)
			return nil, "", err
		}
		err = addAPIToDashboard(config.Cluster.ClusterName, api.Name)
		if err != nil {
			errors.PrintError(err)
		}
		return api, fmt.Sprintf("creating %s", api.Name), nil
	}

	if !areAPIsEqual(prevDeployment, deploymentSpec(api, prevDeployment)) {
		isUpdating, err := isAPIUpdating(prevDeployment)
		if err != nil {
			return nil, "", err
		}
		if isUpdating && !force {
			return nil, "", ErrorAPIUpdating(api.Name)
		}
		if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}
		if err := applyK8sResources(api, prevDeployment, prevService, prevVirtualService); err != nil {
			return nil, "", err
		}
		if err := updateAPIGatewayK8s(prevVirtualService, api); err != nil {
			return nil, "", err
		}
		return api, fmt.Sprintf("updating %s", api.Name), nil
	}

	// deployment didn't change
	isUpdating, err := isAPIUpdating(prevDeployment)
	if err != nil {
		return nil, "", err
	}
	if isUpdating {
		return api, fmt.Sprintf("%s is already updating", api.Name), nil
	}
	return api, fmt.Sprintf("%s is up to date", api.Name), nil
}

func RefreshAPI(apiName string, force bool) (string, error) {
	prevDeployment, err := config.K8s.GetDeployment(k8sName(apiName))
	if err != nil {
		return "", err
	} else if prevDeployment == nil {
		return "", ErrorAPINotDeployed(apiName)
	}

	isUpdating, err := isAPIUpdating(prevDeployment)
	if err != nil {
		return "", err
	}

	if isUpdating && !force {
		return "", ErrorAPIUpdating(apiName)
	}

	apiID, err := k8s.GetLabel(prevDeployment, "apiID")
	if err != nil {
		return "", err
	}

	api, err := DownloadAPISpec(apiName, apiID)
	if err != nil {
		return "", err
	}

	api = spec.GetAPISpec(api.API, api.ProjectID, k8s.RandomName())

	if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
		return "", errors.Wrap(err, "upload api spec")
	}

	if err := applyK8sDeployment(api, prevDeployment); err != nil {
		return "", err
	}

	return fmt.Sprintf("updating %s", api.Name), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	// best effort deletion, so don't handle error yet
	virtualService, vsErr := config.K8s.GetVirtualService(k8sName(apiName))

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
			err := removeAPIFromAPIGatewayK8s(virtualService)
			if err != nil {
				return err
			}
			return nil
		},
		// delete api from cloudwatch
		func() error {
			statuses, err := GetAllStatuses()
			if err != nil {
				return errors.Wrap(err, "failed to get API Statuses")
			}
			//extract all api names from statuses
			allAPINames := make([]string, len(statuses))
			for i, stat := range statuses {
				allAPINames[i] = stat.APIName
			}
			err = removeAPIFromDashboard(allAPINames, config.Cluster.ClusterName, apiName)
			if err != nil {
				return errors.Wrap(err, "failed to delete API from dashboard")
			}
			return nil
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func getK8sResources(apiConfig *userconfig.API) (*kapps.Deployment, *kcore.Service, *kunstructured.Unstructured, error) {
	var deployment *kapps.Deployment
	var service *kcore.Service
	var virtualService *kunstructured.Unstructured

	err := parallel.RunFirstErr(
		func() error {
			var err error
			deployment, err = config.K8s.GetDeployment(k8sName(apiConfig.Name))
			return err
		},
		func() error {
			var err error
			service, err = config.K8s.GetService(k8sName(apiConfig.Name))
			return err
		},
		func() error {
			var err error
			virtualService, err = config.K8s.GetVirtualService(k8sName(apiConfig.Name))
			return err
		},
	)

	return deployment, service, virtualService, err
}

func applyK8sResources(api *spec.API, prevDeployment *kapps.Deployment, prevService *kcore.Service, prevVirtualService *kunstructured.Unstructured) error {
	return parallel.RunFirstErr(
		func() error {
			return applyK8sDeployment(api, prevDeployment)
		},
		func() error {
			return applyK8sService(api, prevService)
		},
		func() error {
			return applyK8sVirtualService(api, prevVirtualService)
		},
	)
}

func applyK8sDeployment(api *spec.API, prevDeployment *kapps.Deployment) error {
	newDeployment := deploymentSpec(api, prevDeployment)

	if prevDeployment == nil {
		_, err := config.K8s.CreateDeployment(newDeployment)
		if err != nil {
			return err
		}
	} else if prevDeployment.Status.ReadyReplicas == 0 {
		// Delete deployment if it never became ready
		config.K8s.DeleteDeployment(k8sName(api.Name))
		_, err := config.K8s.CreateDeployment(newDeployment)
		if err != nil {
			return err
		}
	} else {
		_, err := config.K8s.UpdateDeployment(newDeployment)
		if err != nil {
			return err
		}
	}

	if err := updateAutoscalerCron(newDeployment); err != nil {
		return err
	}

	return nil
}

func updateAutoscalerCron(deployment *kapps.Deployment) error {
	apiName := deployment.Labels["apiName"]

	if prevAutoscalerCron, ok := _autoscalerCrons[apiName]; ok {
		prevAutoscalerCron.Cancel()
	}

	autoscaler, err := autoscaleFn(deployment)
	if err != nil {
		return err
	}

	_autoscalerCrons[apiName] = cron.Run(autoscaler, cronErrHandler(apiName+" autoscaler"), spec.AutoscalingTickInterval)

	return nil
}

func applyK8sService(api *spec.API, prevService *kcore.Service) error {
	newService := serviceSpec(api)

	if prevService == nil {
		_, err := config.K8s.CreateService(newService)
		return err
	}

	_, err := config.K8s.UpdateService(prevService, newService)
	return err
}

func applyK8sVirtualService(api *spec.API, prevVirtualService *kunstructured.Unstructured) error {
	newVirtualService := virtualServiceSpec(api)

	if prevVirtualService == nil {
		_, err := config.K8s.CreateVirtualService(newVirtualService)
		return err
	}

	_, err := config.K8s.UpdateVirtualService(prevVirtualService, newVirtualService)
	return err
}

func deleteK8sResources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			if autoscalerCron, ok := _autoscalerCrons[apiName]; ok {
				autoscalerCron.Cancel()
				delete(_autoscalerCrons, apiName)
			}

			_, err := config.K8s.DeleteDeployment(k8sName(apiName))
			return err
		},
		func() error {
			_, err := config.K8s.DeleteService(k8sName(apiName))
			return err
		},
		func() error {
			_, err := config.K8s.DeleteVirtualService(k8sName(apiName))
			return err
		},
	)
}

func deleteS3Resources(apiName string) error {
	prefix := filepath.Join("apis", apiName)
	return config.AWS.DeleteS3Dir(config.Cluster.Bucket, prefix, true)
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

func areAPIsEqual(d1, d2 *kapps.Deployment) bool {
	return k8s.PodComputesEqual(&d1.Spec.Template.Spec, &d2.Spec.Template.Spec) &&
		k8s.DeploymentStrategiesMatch(d1.Spec.Strategy, d2.Spec.Strategy) &&
		d1.Labels["apiName"] == d2.Labels["apiName"] &&
		d1.Labels["apiID"] == d2.Labels["apiID"] &&
		d1.Labels["deploymentID"] == d2.Labels["deploymentID"] &&
		doCortexAnnotationsMatch(d1, d2)
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
	deployment, err := config.K8s.GetDeployment(k8sName(apiName))
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
	return APILoadBalancerURL()
}

// APILoadBalancerURL returns http endpoint of cluster ingress elb
func APILoadBalancerURL() (string, error) {
	service, err := config.K8sIstio.GetService("ingressgateway-apis")
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", ErrorCortexInstallationBroken()
	}
	if len(service.Status.LoadBalancer.Ingress) == 0 {
		return "", ErrorLoadBalancerInitializing()
	}
	return "http://" + service.Status.LoadBalancer.Ingress[0].Hostname, nil
}

func DownloadAPISpec(apiName string, apiID string) (*spec.API, error) {
	s3Key := spec.Key(apiName, apiID)
	var api spec.API

	if err := config.AWS.ReadMsgpackFromS3(&api, config.Cluster.Bucket, s3Key); err != nil {
		return nil, err
	}

	return &api, nil
}

func DownloadAPISpecs(apiNames []string, apiIDs []string) ([]spec.API, error) {
	apis := make([]spec.API, len(apiNames))
	fns := make([]func() error, len(apiNames))

	for i := range apiNames {
		localIdx := i
		fns[i] = func() error {
			api, err := DownloadAPISpec(apiNames[localIdx], apiIDs[localIdx])
			if err != nil {
				return err
			}
			apis[localIdx] = *api
			return nil
		}
	}

	if len(fns) > 0 {
		err := parallel.RunFirstErr(fns[0], fns[1:]...)
		if err != nil {
			return nil, err
		}
	}

	return apis, nil
}

func GetEndpointFromVirtualService(virtualService *kunstructured.Unstructured) (string, error) {
	endpoints, err := k8s.ExtractVirtualServiceEndpoints(virtualService)
	if err != nil {
		return "", err
	}

	if len(endpoints) != 1 {
		return "", errors.ErrorUnexpected("expected 1 endpoint, but got", endpoints)
	}

	return endpoints.GetOne(), nil
}
