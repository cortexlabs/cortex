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

package asyncapi

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	autoscalerlib "github.com/cortexlabs/cortex/pkg/operator/lib/autoscaler"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
)

const (
	_stalledPodTimeout = 10 * time.Minute
	_tickPeriodMetrics = 10 * time.Second
)

var (
	_autoscalerCrons = make(map[string]cron.Cron)
	_metricsCrons    = make(map[string]cron.Cron)
)

type resources struct {
	apiDeployment         *kapps.Deployment
	gatewayDeployment     *kapps.Deployment
	gatewayService        *kcore.Service
	gatewayVirtualService *istioclientnetworking.VirtualService
}

func getGatewayK8sName(apiName string) string {
	return "gateway-" + apiName
}

func deploymentID() string {
	return k8s.RandomName()[:10]
}

func UpdateAPI(apiConfig userconfig.API, projectID string, force bool) (*spec.API, string, error) {
	prevK8sResources, err := getK8sResources(apiConfig)
	if err != nil {
		return nil, "", err
	}

	deployID := deploymentID()
	if prevK8sResources.apiDeployment != nil && prevK8sResources.apiDeployment.Labels["deploymentID"] != "" {
		deployID = prevK8sResources.apiDeployment.Labels["deploymentID"]
	}

	api := spec.GetAPISpec(&apiConfig, projectID, deployID, config.ClusterName())

	// resource creation
	if prevK8sResources.apiDeployment == nil {
		if err = uploadAPItoS3(*api); err != nil {
			return nil, "", err
		}

		tags := map[string]string{
			"apiName": apiConfig.Name,
		}

		queueURL, err := createFIFOQueue(apiConfig.Name, deployID, tags)
		if err != nil {
			return nil, "", err
		}

		if err = applyK8sResources(*api, prevK8sResources, queueURL); err != nil {
			routines.RunWithPanicHandler(func() {
				_ = parallel.RunFirstErr(
					func() error {
						return deleteQueueByURL(queueURL)
					},
					func() error {
						return deleteK8sResources(api.Name)
					},
				)
			})
			return nil, "", err
		}

		return api, fmt.Sprintf("creating %s", api.Resource.UserString()), nil
	}

	// resource update
	if prevK8sResources.gatewayVirtualService.Labels["specID"] != api.SpecID {
		isUpdating, err := isAPIUpdating(prevK8sResources.apiDeployment)
		if err != nil {
			return nil, "", err
		}
		if isUpdating && !force {
			return nil, "", ErrorAPIUpdating(api.Name)
		}

		if err = uploadAPItoS3(*api); err != nil {
			return nil, "", err
		}

		queueURL, err := getQueueURL(api.Name, prevK8sResources.gatewayVirtualService.Labels["deploymentID"])
		if err != nil {
			return nil, "", err
		}

		if err = applyK8sResources(*api, prevK8sResources, queueURL); err != nil {
			return nil, "", err
		}

		return api, fmt.Sprintf("updating %s", api.Resource.UserString()), nil
	}

	// nothing changed
	isUpdating, err := isAPIUpdating(prevK8sResources.apiDeployment)
	if err != nil {
		return nil, "", err
	}
	if isUpdating {
		return api, fmt.Sprintf("%s is already updating", api.Resource.UserString()), nil
	}
	return api, fmt.Sprintf("%s is up to date", api.Resource.UserString()), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	err := parallel.RunFirstErr(
		func() error {
			deployment, err := config.K8s.GetVirtualService(operator.K8sName(apiName))
			if err != nil {
				return err
			}
			queueURL, err := getQueueURL(apiName, deployment.Labels["deploymentID"])
			if err != nil {
				return err
			}
			// best effort deletion
			_ = deleteQueueByURL(queueURL)
			return nil
		},
		func() error {
			return deleteK8sResources(apiName)
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

	if err != nil {
		return err
	}

	return nil
}

func GetAPIByName(deployedResource *operator.DeployedResource) ([]schema.APIResponse, error) {
	status, err := GetStatus(deployedResource.Name)
	if err != nil {
		return nil, err
	}

	api, err := operator.DownloadAPISpec(status.APIName, status.APIID)
	if err != nil {
		return nil, err
	}

	apiEndpoint, err := operator.APIEndpoint(api)
	if err != nil {
		return nil, err
	}

	return []schema.APIResponse{
		{
			Spec:     *api,
			Status:   status,
			Endpoint: apiEndpoint,
		},
	}, nil
}

func GetAllAPIs(pods []kcore.Pod, deployments []kapps.Deployment) ([]schema.APIResponse, error) {
	statuses, err := GetAllStatuses(deployments, pods)
	if err != nil {
		return nil, err
	}

	apiNames, apiIDs := namesAndIDsFromStatuses(statuses)
	apis, err := operator.DownloadAPISpecs(apiNames, apiIDs)
	if err != nil {
		return nil, err
	}

	//allMetrics, err := GetMultipleMetrics(apis)
	//if err != nil {
	//	return nil, err
	//}

	realtimeAPIs := make([]schema.APIResponse, len(apis))

	for i := range apis {
		api := apis[i]
		endpoint, err := operator.APIEndpoint(&api)
		if err != nil {
			return nil, err
		}

		realtimeAPIs[i] = schema.APIResponse{
			Spec:   api,
			Status: &statuses[i],
			//Metrics:  &allMetrics[i],
			Endpoint: endpoint,
		}
	}

	return realtimeAPIs, nil
}

func UpdateMetricsCron(deployment *kapps.Deployment) error {
	// skip gateway deployments
	if deployment.Labels["cortex.dev/async"] != "api" {
		return nil
	}

	apiName := deployment.Labels["apiName"]
	deployID := deployment.Labels["deploymentID"]

	if prevMetricsCron, ok := _metricsCrons[apiName]; ok {
		prevMetricsCron.Cancel()
	}

	queueURL, err := getQueueURL(apiName, deployID)
	if err != nil {
		return err
	}

	metricsCron := updateQueueLengthMetricsFn(apiName, queueURL)

	_metricsCrons[apiName] = cron.Run(metricsCron, operator.ErrorHandler(apiName+" metrics"), _tickPeriodMetrics)

	return nil
}

func UpdateAutoscalerCron(deployment *kapps.Deployment, apiSpec spec.API) error {
	// skip gateway deployments
	if deployment.Labels["cortex.dev/async"] != "api" {
		return nil
	}

	apiName := deployment.Labels["apiName"]
	if prevAutoscalerCron, ok := _autoscalerCrons[apiName]; ok {
		prevAutoscalerCron.Cancel()
	}

	autoscaler, err := autoscalerlib.AutoscaleFn(deployment, &apiSpec, getMessagesInQueue)
	if err != nil {
		return err
	}

	_autoscalerCrons[apiName] = cron.Run(autoscaler, operator.ErrorHandler(apiName+" autoscaler"), spec.AutoscalingTickInterval)

	return nil
}

func getK8sResources(apiConfig userconfig.API) (resources, error) {
	var deployment *kapps.Deployment
	var gatewayDeployment *kapps.Deployment
	var gatewayService *kcore.Service
	var gatewayVirtualService *istioclientnetworking.VirtualService

	gatewayK8sName := getGatewayK8sName(apiConfig.Name)
	apiK8sName := operator.K8sName(apiConfig.Name)

	err := parallel.RunFirstErr(
		func() error {
			var err error
			deployment, err = config.K8s.GetDeployment(apiK8sName)
			return err
		},
		func() error {
			var err error
			gatewayDeployment, err = config.K8s.GetDeployment(gatewayK8sName)
			return err
		},
		func() error {
			var err error
			gatewayService, err = config.K8s.GetService(apiK8sName)
			return err
		},
		func() error {
			var err error
			gatewayVirtualService, err = config.K8s.GetVirtualService(apiK8sName)
			return err
		},
	)

	return resources{
		apiDeployment:         deployment,
		gatewayDeployment:     gatewayDeployment,
		gatewayService:        gatewayService,
		gatewayVirtualService: gatewayVirtualService,
	}, err
}

func applyK8sResources(api spec.API, prevK8sResources resources, queueURL string) error {
	gatewayDeployment := apiDeploymentSpec(api, prevK8sResources.apiDeployment, queueURL)
	apiDeployment := gatewayDeploymentSpec(api, prevK8sResources.gatewayDeployment, queueURL)
	gatewayService := gatewayServiceSpec(api)
	gatewayVirtualService := gatewayVirtualServiceSpec(api)

	return parallel.RunFirstErr(
		func() error {
			return applyK8sDeployment(api, prevK8sResources.apiDeployment, &apiDeployment)
		},
		func() error {
			return applyK8sDeployment(api, prevK8sResources.gatewayDeployment, &gatewayDeployment)
		},
		func() error {
			return applyK8sService(prevK8sResources.gatewayService, &gatewayService)
		},
		func() error {
			return applyK8sVirtualService(prevK8sResources.gatewayVirtualService, &gatewayVirtualService)
		},
	)
}

func applyK8sDeployment(api spec.API, prevDeployment *kapps.Deployment, newDeployment *kapps.Deployment) error {
	if prevDeployment == nil {
		_, err := config.K8s.CreateDeployment(newDeployment)
		if err != nil {
			return err
		}
	} else if prevDeployment.Status.ReadyReplicas == 0 {
		// Delete deployment if it never became ready
		_, _ = config.K8s.DeleteDeployment(operator.K8sName(api.Name))
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

	if err := UpdateMetricsCron(newDeployment); err != nil {
		return err
	}

	if err := UpdateAutoscalerCron(newDeployment, api); err != nil {
		return err
	}

	return nil
}

func applyK8sService(prevService *kcore.Service, newService *kcore.Service) error {
	if prevService == nil {
		_, err := config.K8s.CreateService(newService)
		return err
	}

	_, err := config.K8s.UpdateService(prevService, newService)
	return err
}

func applyK8sVirtualService(prevVirtualService *istioclientnetworking.VirtualService, newVirtualService *istioclientnetworking.VirtualService) error {
	if prevVirtualService == nil {
		_, err := config.K8s.CreateVirtualService(newVirtualService)
		return err
	}

	_, err := config.K8s.UpdateVirtualService(prevVirtualService, newVirtualService)
	return err
}

func deleteBucketResources(apiName string) error {
	prefix := filepath.Join(config.ClusterName(), "apis", apiName)
	return config.DeleteBucketDir(prefix, true)
}

func deleteK8sResources(apiName string) error {
	apiK8sName := operator.K8sName(apiName)

	err := parallel.RunFirstErr(
		func() error {
			if metricsCron, ok := _metricsCrons[apiName]; ok {
				metricsCron.Cancel()
				delete(_metricsCrons, apiName)
			}

			if autoscalerCron, ok := _autoscalerCrons[apiName]; ok {
				autoscalerCron.Cancel()
				delete(_autoscalerCrons, apiName)
			}
			_, err := config.K8s.DeleteDeployment(apiK8sName)
			return err
		},
		func() error {
			gatewayK8sName := getGatewayK8sName(apiName)
			_, err := config.K8s.DeleteDeployment(gatewayK8sName)
			return err
		},
		func() error {
			_, err := config.K8s.DeleteService(apiK8sName)
			return err
		},
		func() error {
			_, err := config.K8s.DeleteVirtualService(apiK8sName)
			return err
		},
	)

	return err
}

func uploadAPItoS3(api spec.API) error {
	return parallel.RunFirstErr(
		func() error {
			var err error
			err = config.UploadJSONToBucket(api, api.Key)
			if err != nil {
				err = errors.Wrap(err, "upload api spec")
			}
			return err
		},
		func() error {
			var err error
			// Use api spec indexed by PredictorID for replicas to prevent rolling updates when SpecID changes without PredictorID changing
			err = config.UploadJSONToBucket(api, api.PredictorKey)
			if err != nil {
				err = errors.Wrap(err, "upload predictor spec")
			}
			return err
		},
	)
}
