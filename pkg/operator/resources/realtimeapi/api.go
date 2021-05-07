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
	"net/http"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/requests"
	autoscalerlib "github.com/cortexlabs/cortex/pkg/operator/lib/autoscaler"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
)

const _realtimeDashboardUID = "realtimeapi"

var _autoscalerCrons = make(map[string]cron.Cron) // apiName -> cron

func deploymentID() string {
	return k8s.RandomName()[:10]
}

func UpdateAPI(apiConfig *userconfig.API, projectID string, force bool) (*spec.API, string, error) {
	prevDeployment, prevService, prevVirtualService, err := getK8sResources(apiConfig)
	if err != nil {
		return nil, "", err
	}

	deploymentID := deploymentID()
	if prevDeployment != nil && prevDeployment.Labels["deploymentID"] != "" {
		deploymentID = prevDeployment.Labels["deploymentID"]
	}

	api := spec.GetAPISpec(apiConfig, projectID, deploymentID, config.ClusterConfig.ClusterUID)

	if prevDeployment == nil {
		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		// Use api spec indexed by HandlerID for replicas to prevent rolling updates when SpecID changes without HandlerID changing
		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.HandlerKey); err != nil {
			return nil, "", errors.Wrap(err, "upload handler spec")
		}

		if err := applyK8sResources(api, prevDeployment, prevService, prevVirtualService); err != nil {
			routines.RunWithPanicHandler(func() {
				deleteK8sResources(api.Name)
			})
			return nil, "", err
		}

		return api, fmt.Sprintf("creating %s", api.Resource.UserString()), nil
	}

	if prevVirtualService.Labels["specID"] != api.SpecID || prevVirtualService.Labels["deploymentID"] != api.DeploymentID {
		isUpdating, err := isAPIUpdating(prevDeployment)
		if err != nil {
			return nil, "", err
		}
		if isUpdating && !force {
			return nil, "", ErrorAPIUpdating(api.Name)
		}

		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		// Use api spec indexed by HandlerID for replicas to prevent rolling updates when SpecID changes without HandlerID changing
		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.HandlerKey); err != nil {
			return nil, "", errors.Wrap(err, "upload handler spec")
		}

		if err := applyK8sResources(api, prevDeployment, prevService, prevVirtualService); err != nil {
			return nil, "", err
		}
		return api, fmt.Sprintf("updating %s", api.Resource.UserString()), nil
	}

	// deployment didn't change
	isUpdating, err := isAPIUpdating(prevDeployment)
	if err != nil {
		return nil, "", err
	}
	if isUpdating {
		return api, fmt.Sprintf("%s is already updating", api.Resource.UserString()), nil
	}
	return api, fmt.Sprintf("%s is up to date", api.Resource.UserString()), nil
}

func RefreshAPI(apiName string, force bool) (string, error) {
	prevDeployment, err := config.K8s.GetDeployment(workloads.K8sName(apiName))
	if err != nil {
		return "", err
	} else if prevDeployment == nil {
		return "", errors.ErrorUnexpected("unable to find deployment", apiName)
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

	api, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return "", err
	}

	api = spec.GetAPISpec(api.API, api.ProjectID, deploymentID(), config.ClusterConfig.ClusterUID)

	if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
		return "", errors.Wrap(err, "upload api spec")
	}

	// Reupload api spec to the same HandlerID but with the new DeploymentID
	if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.HandlerKey); err != nil {
		return "", errors.Wrap(err, "upload handler spec")
	}

	if err := applyK8sDeployment(api, prevDeployment); err != nil {
		return "", err
	}

	return fmt.Sprintf("updating %s", api.Name), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	err := parallel.RunFirstErr(
		func() error {
			return deleteK8sResources(apiName)
		},
		func() error {
			if keepCache {
				return nil
			}
			// best effort deletion, swallow errors because there could be weird error messages
			deleteBucketResources(apiName)
			return nil
		},
	)

	if err != nil {
		return err
	}

	return nil
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

	realtimeAPIs := make([]schema.APIResponse, len(apis))

	for i := range apis {
		api := apis[i]
		endpoint, err := operator.APIEndpoint(&api)
		if err != nil {
			return nil, err
		}

		realtimeAPIs[i] = schema.APIResponse{
			Spec:     api,
			Status:   &statuses[i],
			Metrics:  &allMetrics[i],
			Endpoint: endpoint,
		}
	}

	return realtimeAPIs, nil
}

func namesAndIDsFromStatuses(statuses []status.Status) ([]string, []string) {
	apiNames := make([]string, len(statuses))
	apiIDs := make([]string, len(statuses))

	for i, status := range statuses {
		apiNames[i] = status.APIName
		apiIDs[i] = status.APIID
	}

	return apiNames, apiIDs
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

	metrics, err := GetMetrics(api)
	if err != nil {
		return nil, err
	}

	apiEndpoint, err := operator.APIEndpoint(api)
	if err != nil {
		return nil, err
	}

	dashboardURL := pointer.String(getDashboardURL(api.Name))

	grpcPorts := map[string]int64{}
	if api.Handler != nil && api.Handler.IsGRPC() {
		grpcPorts["insecure"] = 80
		grpcPorts["secure"] = 443
	}

	return []schema.APIResponse{
		{
			Spec:         *api,
			Status:       status,
			Metrics:      metrics,
			Endpoint:     apiEndpoint,
			GRPCPorts:    grpcPorts,
			DashboardURL: dashboardURL,
		},
	}, nil
}

func getK8sResources(apiConfig *userconfig.API) (*kapps.Deployment, *kcore.Service, *istioclientnetworking.VirtualService, error) {
	var deployment *kapps.Deployment
	var service *kcore.Service
	var virtualService *istioclientnetworking.VirtualService

	err := parallel.RunFirstErr(
		func() error {
			var err error
			deployment, err = config.K8s.GetDeployment(workloads.K8sName(apiConfig.Name))
			return err
		},
		func() error {
			var err error
			service, err = config.K8s.GetService(workloads.K8sName(apiConfig.Name))
			return err
		},
		func() error {
			var err error
			virtualService, err = config.K8s.GetVirtualService(workloads.K8sName(apiConfig.Name))
			return err
		},
	)

	return deployment, service, virtualService, err
}

func applyK8sResources(api *spec.API, prevDeployment *kapps.Deployment, prevService *kcore.Service, prevVirtualService *istioclientnetworking.VirtualService) error {
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
		config.K8s.DeleteDeployment(workloads.K8sName(api.Name))
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

	if err := UpdateAutoscalerCron(newDeployment, api); err != nil {
		return err
	}

	return nil
}

func UpdateAutoscalerCron(deployment *kapps.Deployment, apiSpec *spec.API) error {
	apiName := deployment.Labels["apiName"]

	if prevAutoscalerCron, ok := _autoscalerCrons[apiName]; ok {
		prevAutoscalerCron.Cancel()
	}

	autoscaler, err := autoscalerlib.AutoscaleFn(deployment, apiSpec, getInflightRequests)
	if err != nil {
		return err
	}

	_autoscalerCrons[apiName] = cron.Run(autoscaler, operator.ErrorHandler(apiName+" autoscaler"), spec.AutoscalingTickInterval)

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

func applyK8sVirtualService(api *spec.API, prevVirtualService *istioclientnetworking.VirtualService) error {
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

			_, err := config.K8s.DeleteDeployment(workloads.K8sName(apiName))
			return err
		},
		func() error {
			_, err := config.K8s.DeleteService(workloads.K8sName(apiName))
			return err
		},
		func() error {
			_, err := config.K8s.DeleteVirtualService(workloads.K8sName(apiName))
			return err
		},
	)
}

func deleteBucketResources(apiName string) error {
	prefix := filepath.Join(config.ClusterConfig.ClusterUID, "apis", apiName)
	return config.AWS.DeleteS3Dir(config.ClusterConfig.Bucket, prefix, true)
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
	return deployment.Spec.Template.Labels["handlerID"] == pod.Labels["handlerID"] &&
		deployment.Spec.Template.Labels["deploymentID"] == pod.Labels["deploymentID"]
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

func GetModelsMetadata(status *status.Status, handler *userconfig.Handler, apiEndpoint string) (*schema.TFLiveReloadingSummary, *schema.PythonModelSummary, error) {
	if status.Updated.Ready+status.Stale.Ready == 0 {
		return nil, nil, nil
	}

	cachingEnabled := handler.Models != nil && handler.Models.CacheSize != nil && handler.Models.DiskCacheSize != nil
	if handler.Type == userconfig.TensorFlowHandlerType && !cachingEnabled {
		tfLiveReloadingSummary, err := getTFLiveReloadingSummary(apiEndpoint)
		if err != nil {
			return nil, nil, err
		}
		return tfLiveReloadingSummary, nil, nil
	}

	if handler.Type == userconfig.PythonHandlerType && handler.MultiModelReloading != nil {
		pythonModelSummary, err := getPythonModelSummary(apiEndpoint)
		if err != nil {
			return nil, nil, err
		}
		return nil, pythonModelSummary, nil
	}

	return nil, nil, nil
}

func getPythonModelSummary(apiEndpoint string) (*schema.PythonModelSummary, error) {
	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request api summary")
	}
	req.Header.Set("Content-Type", "application/json")
	_, response, err := requests.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	var pythonModelSummary schema.PythonModelSummary
	err = json.DecodeWithNumber(response, &pythonModelSummary)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse api summary response")
	}
	return &pythonModelSummary, nil
}

func getTFLiveReloadingSummary(apiEndpoint string) (*schema.TFLiveReloadingSummary, error) {
	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request api summary")
	}
	req.Header.Set("Content-Type", "application/json")
	_, response, err := requests.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	var tfLiveReloadingSummary schema.TFLiveReloadingSummary
	err = json.DecodeWithNumber(response, &tfLiveReloadingSummary)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse api summary response")
	}
	return &tfLiveReloadingSummary, nil
}
