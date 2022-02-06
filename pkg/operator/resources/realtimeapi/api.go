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

package realtimeapi

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
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

func generateDeploymentID() string {
	return k8s.RandomName()[:10]
}

func UpdateAPI(apiConfig *userconfig.API, force bool) (*spec.API, string, error) {
	prevDeployment, prevService, prevVirtualService, err := getK8sResources(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	initialDeploymentTime := time.Now().UnixNano()
	deploymentID := generateDeploymentID()
	if prevVirtualService != nil && prevVirtualService.Labels["initialDeploymentTime"] != "" {
		var err error
		initialDeploymentTime, err = k8s.ParseInt64Label(prevVirtualService, "initialDeploymentTime")
		if err != nil {
			return nil, "", err
		}
		deploymentID = prevVirtualService.Labels["deploymentID"]
	}

	api := spec.GetAPISpec(apiConfig, initialDeploymentTime, deploymentID, config.ClusterConfig.ClusterUID)

	if prevDeployment == nil {
		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		if err := applyK8sResources(api, prevDeployment, prevService, prevVirtualService); err != nil {
			routines.RunWithPanicHandler(func() {
				_ = deleteK8sResources(api.Name)
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
	prevDeployment, prevService, prevVirtualService, err := getK8sResources(apiName)
	if err != nil {
		return "", err
	} else if prevDeployment == nil || prevVirtualService == nil {
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

	initialDeploymentTime, err := k8s.ParseInt64Label(prevVirtualService, "initialDeploymentTime")
	if err != nil {
		return "", err
	}

	api = spec.GetAPISpec(api.API, initialDeploymentTime, generateDeploymentID(), config.ClusterConfig.ClusterUID)

	if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
		return "", errors.Wrap(err, "upload api spec")
	}

	if err := applyK8sResources(api, prevDeployment, prevService, prevVirtualService); err != nil {
		return "", err
	}

	return fmt.Sprintf("updating %s", api.Resource.UserString()), nil
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
			_ = deleteBucketResources(apiName)
			return nil
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func GetAllAPIs(deployments []kapps.Deployment) ([]schema.APIResponse, error) {
	realtimeAPIs := make([]schema.APIResponse, len(deployments))
	mappedRealtimeAPIs := make(map[string]schema.APIResponse, len(deployments))
	apiNames := make([]string, len(deployments))

	for i := range deployments {
		apiName := deployments[i].Labels["apiName"]
		apiNames[i] = apiName

		metadata, err := spec.MetadataFromDeployment(&deployments[i])
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("api %s", apiName))
		}
		mappedRealtimeAPIs[apiName] = schema.APIResponse{
			Status:   status.FromDeployment(&deployments[i]),
			Metadata: metadata,
		}
	}

	sort.Strings(apiNames)
	for i := range apiNames {
		realtimeAPIs[i] = mappedRealtimeAPIs[apiNames[i]]
	}

	return realtimeAPIs, nil
}

func GetAPIByName(deployedResource *operator.DeployedResource) ([]schema.APIResponse, error) {
	deployment, err := config.K8s.GetDeployment(workloads.K8sName(deployedResource.Name))
	if err != nil {
		return nil, err
	}

	if deployment == nil {
		return nil, errors.ErrorUnexpected("unable to find deployment", deployedResource.Name)
	}

	apiStatus := status.FromDeployment(deployment)
	apiMetadata, err := spec.MetadataFromDeployment(deployment)
	if err != nil {
		return nil, errors.ErrorUnexpected("unable to obtain metadata", deployedResource.Name)
	}

	api, err := operator.DownloadAPISpec(apiMetadata.Name, apiMetadata.APIID)
	if err != nil {
		return nil, err
	}

	apiEndpoint, err := operator.APIEndpoint(api)
	if err != nil {
		return nil, err
	}

	dashboardURL := pointer.String(getDashboardURL(api.Name))

	return []schema.APIResponse{
		{
			Spec:         api,
			Metadata:     apiMetadata,
			Status:       apiStatus,
			Endpoint:     &apiEndpoint,
			DashboardURL: dashboardURL,
		},
	}, nil
}

func DescribeAPIByName(deployedResource *operator.DeployedResource) ([]schema.APIResponse, error) {
	deployment, err := config.K8s.GetDeployment(workloads.K8sName(deployedResource.Name))
	if err != nil {
		return nil, err
	}

	if deployment == nil {
		return nil, errors.ErrorUnexpected("unable to find deployment", deployedResource.Name)
	}

	apiStatus := status.FromDeployment(deployment)
	apiMetadata, err := spec.MetadataFromDeployment(deployment)
	if err != nil {
		return nil, errors.ErrorUnexpected("unable to obtain metadata", deployedResource.Name)
	}

	pods, err := config.K8s.ListPodsByLabel("apiName", deployment.Labels["apiName"])
	if err != nil {
		return nil, err
	}
	apiStatus.ReplicaCounts = GetReplicaCounts(deployment, pods)

	apiEndpoint, err := operator.APIEndpointFromResource(deployedResource)
	if err != nil {
		return nil, err
	}

	dashboardURL := pointer.String(getDashboardURL(deployedResource.Name))

	return []schema.APIResponse{
		{
			Metadata:     apiMetadata,
			Status:       apiStatus,
			Endpoint:     &apiEndpoint,
			DashboardURL: dashboardURL,
		},
	}, nil
}

func getK8sResources(apiName string) (*kapps.Deployment, *kcore.Service, *istioclientnetworking.VirtualService, error) {
	var deployment *kapps.Deployment
	var service *kcore.Service
	var virtualService *istioclientnetworking.VirtualService

	err := parallel.RunFirstErr(
		func() error {
			var err error
			deployment, err = config.K8s.GetDeployment(workloads.K8sName(apiName))
			return err
		},
		func() error {
			var err error
			service, err = config.K8s.GetService(workloads.K8sName(apiName))
			return err
		},
		func() error {
			var err error
			virtualService, err = config.K8s.GetVirtualService(workloads.K8sName(apiName))
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
		_, _ = config.K8s.DeleteDeployment(workloads.K8sName(api.Name))
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

	replicaCounts := GetReplicaCounts(deployment, pods)

	autoscalingSpec, err := userconfig.AutoscalingFromAnnotations(deployment)
	if err != nil {
		return false, err
	}

	if replicaCounts.Ready < autoscalingSpec.MinReplicas && replicaCounts.TotalFailed() == 0 {
		return true, nil
	}

	return false, nil
}

func isPodSpecLatest(deployment *kapps.Deployment, pod *kcore.Pod) bool {
	return deployment.Spec.Template.Labels["podID"] == pod.Labels["podID"] &&
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
