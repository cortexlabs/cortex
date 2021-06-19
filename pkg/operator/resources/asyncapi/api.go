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

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	autoscalerlib "github.com/cortexlabs/cortex/pkg/operator/lib/autoscaler"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kautoscaling "k8s.io/api/autoscaling/v2beta2"
	kcore "k8s.io/api/core/v1"
)

const (
	_stalledPodTimeout = 15 * time.Minute
	_tickPeriodMetrics = 10 * time.Second
	_asyncDashboardUID = "asyncapi"
)

var (
	_autoscalerCrons = make(map[string]cron.Cron)
	_metricsCrons    = make(map[string]cron.Cron)
)

type resources struct {
	apiDeployment         *kapps.Deployment
	apiConfigMap          *kcore.ConfigMap
	gatewayDeployment     *kapps.Deployment
	gatewayService        *kcore.Service
	gatewayHPA            *kautoscaling.HorizontalPodAutoscaler
	gatewayVirtualService *istioclientnetworking.VirtualService
}

func getGatewayK8sName(apiName string) string {
	return "gateway-" + apiName
}

func generateDeploymentID() string {
	return k8s.RandomName()[:10]
}

func UpdateAPI(apiConfig userconfig.API, force bool) (*spec.API, string, error) {
	prevK8sResources, err := getK8sResources(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	createdTime := time.Now().UnixNano()
	deploymentID := generateDeploymentID()
	if prevK8sResources.gatewayVirtualService != nil && prevK8sResources.gatewayVirtualService.Labels["createdTime"] != "" {
		var err error
		createdTime, err = k8s.ParseInt64Label(prevK8sResources.gatewayVirtualService, "createdTime")
		if err != nil {
			return nil, "", err
		}
		deploymentID = prevK8sResources.gatewayVirtualService.Labels["deploymentID"]
	}

	api := spec.GetAPISpec(&apiConfig, createdTime, deploymentID, config.ClusterConfig.ClusterUID)

	// resource creation
	if prevK8sResources.gatewayVirtualService == nil {
		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		tags := map[string]string{
			"apiName": apiConfig.Name,
		}

		queueURL, err := createFIFOQueue(apiConfig.Name, createdTime, tags)
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

		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		createdTime, err := k8s.ParseInt64Label(prevK8sResources.gatewayVirtualService, "createdTime")
		if err != nil {
			return nil, "", err
		}

		queueURL, err := getQueueURL(api.Name, createdTime)
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

func RefreshAPI(apiName string, force bool) (string, error) {
	prevK8sResources, err := getK8sResources(apiName)
	if err != nil {
		return "", err
	} else if prevK8sResources.gatewayVirtualService == nil {
		return "", errors.ErrorUnexpected("unable to find deployment", apiName)
	}

	isUpdating, err := isAPIUpdating(prevK8sResources.apiDeployment)
	if err != nil {
		return "", err
	}

	if isUpdating && !force {
		return "", ErrorAPIUpdating(apiName)
	}

	apiID, err := k8s.GetLabel(prevK8sResources.gatewayVirtualService, "apiID")
	if err != nil {
		return "", err
	}

	api, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return "", err
	}

	createdTime, err := k8s.ParseInt64Label(prevK8sResources.gatewayVirtualService, "createdTime")
	if err != nil {
		return "", err
	}

	api = spec.GetAPISpec(api.API, createdTime, generateDeploymentID(), config.ClusterConfig.ClusterUID)

	if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
		return "", errors.Wrap(err, "upload api spec")
	}

	queueURL, err := getQueueURL(api.Name, createdTime)
	if err != nil {
		return "", err
	}

	if err = applyK8sResources(*api, prevK8sResources, queueURL); err != nil {
		return "", err
	}

	return fmt.Sprintf("updating %s", api.Resource.UserString()), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	err := parallel.RunFirstErr(
		func() error {
			vs, err := config.K8s.GetVirtualService(workloads.K8sName(apiName))
			if err != nil {
				return err
			}
			if vs != nil {
				createdTime, err := k8s.ParseInt64Label(vs, "createdTime")
				if err != nil {
					return err
				}
				queueURL, err := getQueueURL(apiName, createdTime)
				if err != nil {
					return err
				}
				// best effort deletion
				_ = deleteQueueByURL(queueURL)
			}
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

	metrics, err := GetMetrics(*api)
	if err != nil {
		return nil, err
	}

	dashboardURL := pointer.String(getDashboardURL(api.Name))

	return []schema.APIResponse{
		{
			Spec:         *api,
			Status:       status,
			Endpoint:     apiEndpoint,
			DashboardURL: dashboardURL,
			Metrics:      metrics,
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

	allMetrics, err := GetMultipleMetrics(apis)
	if err != nil {
		return nil, err
	}

	asyncAPIs := make([]schema.APIResponse, len(apis))

	for i := range apis {
		api := apis[i]
		endpoint, err := operator.APIEndpoint(&api)
		if err != nil {
			return nil, err
		}

		asyncAPIs[i] = schema.APIResponse{
			Spec:     api,
			Status:   &statuses[i],
			Endpoint: endpoint,
			Metrics:  &allMetrics[i],
		}
	}

	return asyncAPIs, nil
}

func UpdateMetricsCron(deployment *kapps.Deployment) error {
	// skip gateway deployments
	if deployment.Labels["cortex.dev/async"] != "api" {
		return nil
	}

	apiName := deployment.Labels["apiName"]

	if prevMetricsCron, ok := _metricsCrons[apiName]; ok {
		prevMetricsCron.Cancel()
	}

	createdTime, err := k8s.ParseInt64Label(deployment, "createdTime")
	if err != nil {
		return err
	}

	queueURL, err := getQueueURL(apiName, createdTime)
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

func getK8sResources(apiName string) (resources, error) {
	var deployment *kapps.Deployment
	var apiConfigMap *kcore.ConfigMap
	var gatewayDeployment *kapps.Deployment
	var gatewayService *kcore.Service
	var gatewayHPA *kautoscaling.HorizontalPodAutoscaler
	var gatewayVirtualService *istioclientnetworking.VirtualService

	gatewayK8sName := getGatewayK8sName(apiName)
	apiK8sName := workloads.K8sName(apiName)

	err := parallel.RunFirstErr(
		func() error {
			var err error
			deployment, err = config.K8s.GetDeployment(apiK8sName)
			return err
		},
		func() error {
			var err error
			apiConfigMap, err = config.K8s.GetConfigMap(apiK8sName)
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
			gatewayHPA, err = config.K8s.GetHPA(gatewayK8sName)
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
		apiConfigMap:          apiConfigMap,
		gatewayDeployment:     gatewayDeployment,
		gatewayService:        gatewayService,
		gatewayHPA:            gatewayHPA,
		gatewayVirtualService: gatewayVirtualService,
	}, err
}

func applyK8sResources(api spec.API, prevK8sResources resources, queueURL string) error {
	apiDeployment := deploymentSpec(api, prevK8sResources.apiDeployment, queueURL)
	apiConfigMap, err := configMapSpec(api)
	if err != nil {
		return err
	}
	gatewayDeployment := gatewayDeploymentSpec(api, queueURL)
	gatewayHPA, err := gatewayHPASpec(api)
	if err != nil {
		return err
	}
	gatewayService := gatewayServiceSpec(api)
	gatewayVirtualService := gatewayVirtualServiceSpec(api)

	return parallel.RunFirstErr(
		func() error {
			if err := applyK8sConfigMap(prevK8sResources.apiConfigMap, &apiConfigMap); err != nil {
				return err
			}

			if err := applyK8sDeployment(prevK8sResources.apiDeployment, &apiDeployment); err != nil {
				return err
			}

			if err := UpdateMetricsCron(&apiDeployment); err != nil {
				return err
			}

			if err := UpdateAutoscalerCron(&apiDeployment, api); err != nil {
				return err
			}

			return nil
		},
		func() error {
			return applyK8sDeployment(prevK8sResources.gatewayDeployment, &gatewayDeployment)
		},
		func() error {
			return applyK8sHPA(prevK8sResources.gatewayHPA, &gatewayHPA)
		},
		func() error {
			return applyK8sService(prevK8sResources.gatewayService, &gatewayService)
		},
		func() error {
			return applyK8sVirtualService(prevK8sResources.gatewayVirtualService, &gatewayVirtualService)
		},
	)
}

func applyK8sConfigMap(prevConfigMap *kcore.ConfigMap, newConfigMap *kcore.ConfigMap) error {
	if prevConfigMap == nil {
		_, err := config.K8s.CreateConfigMap(newConfigMap)
		if err != nil {
			return err
		}
	} else {
		_, err := config.K8s.UpdateConfigMap(newConfigMap)
		if err != nil {
			return err
		}
	}
	return nil
}

func applyK8sDeployment(prevDeployment *kapps.Deployment, newDeployment *kapps.Deployment) error {
	if prevDeployment == nil {
		_, err := config.K8s.CreateDeployment(newDeployment)
		if err != nil {
			return err
		}
	} else if prevDeployment.Status.ReadyReplicas == 0 {
		// Delete deployment if it never became ready
		_, _ = config.K8s.DeleteDeployment(prevDeployment.Name)
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
	return nil
}

func applyK8sHPA(prevHPA *kautoscaling.HorizontalPodAutoscaler, newHPA *kautoscaling.HorizontalPodAutoscaler) error {
	var err error
	if prevHPA == nil {
		_, err = config.K8s.CreateHPA(newHPA)
	} else {
		_, err = config.K8s.UpdateHPA(newHPA)
	}
	if err != nil {
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
	prefix := filepath.Join(config.ClusterConfig.ClusterUID, "apis", apiName)
	return config.AWS.DeleteS3Dir(config.ClusterConfig.Bucket, prefix, true)
}

func deleteK8sResources(apiName string) error {
	apiK8sName := workloads.K8sName(apiName)
	gatewayK8sName := getGatewayK8sName(apiName)

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
			_, err := config.K8s.DeleteConfigMap(apiK8sName)
			return err
		},
		func() error {
			_, err := config.K8s.DeleteDeployment(gatewayK8sName)
			return err
		},
		func() error {
			_, err := config.K8s.DeleteHPA(gatewayK8sName)
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

func getDashboardURL(apiName string) string {
	loadBalancerURL, err := operator.LoadBalancerURL()
	if err != nil {
		return ""
	}

	dashboardURL := fmt.Sprintf(
		"%s/dashboard/d/%s/asyncapi?orgId=1&refresh=30s&var-api_name=%s",
		loadBalancerURL, _asyncDashboardUID, apiName,
	)

	return dashboardURL
}
