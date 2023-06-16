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

package resources

import (
	"context"
	"fmt"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/asyncapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job/taskapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/realtimeapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/trafficsplitter"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var operatorLogger = logging.GetLogger()

// Returns an error if resource doesn't exist
func GetDeployedResourceByName(resourceName string) (*operator.DeployedResource, error) {
	resource, err := GetDeployedResourceByNameOrNil(resourceName)
	if err != nil {
		return nil, err
	}

	if resource == nil {
		return nil, ErrorAPINotDeployed(resourceName)
	}

	return resource, nil
}

func GetDeployedResourceByNameOrNil(resourceName string) (*operator.DeployedResource, error) {
	virtualService, err := config.K8s.GetVirtualService(workloads.K8sName(resourceName))
	if err != nil {
		return nil, err
	}

	if virtualService == nil {
		return nil, nil
	}

	return &operator.DeployedResource{
		Resource: userconfig.Resource{
			Name: virtualService.Labels["apiName"],
			Kind: userconfig.KindFromString(virtualService.Labels["apiKind"]),
		},
		VirtualService: virtualService,
	}, nil
}

func Deploy(configFileName string, configBytes []byte, force bool) ([]schema.DeployResult, error) {
	apiConfigs, err := spec.ExtractAPIConfigs(configBytes, configFileName)
	if err != nil {
		return nil, err
	}

	err = ValidateClusterAPIs(apiConfigs)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found at https://docs.cortexlabs.com/v/%s/", consts.CortexVersionMinor))
		return nil, err
	}

	// This is done if user specifies RealtimeAPIs in same file as TrafficSplitter
	apiConfigs = append(ExclusiveFilterAPIsByKind(apiConfigs, userconfig.TrafficSplitterKind), InclusiveFilterAPIsByKind(apiConfigs, userconfig.TrafficSplitterKind)...)

	results := make([]schema.DeployResult, 0, len(apiConfigs))
	for i := range apiConfigs {
		apiConfig := apiConfigs[i]

		api, msg, err := UpdateAPI(&apiConfig, force)

		result := schema.DeployResult{
			Message: msg,
			API:     api,
		}

		if err != nil {
			result.Error = errors.ErrorStr(err)
		}

		results = append(results, result)
	}

	return results, nil
}

func UpdateAPI(apiConfig *userconfig.API, force bool) (*schema.APIResponse, string, error) {
	deployedResource, err := GetDeployedResourceByNameOrNil(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	if deployedResource != nil && deployedResource.Kind != apiConfig.Kind {
		return nil, "", ErrorCannotChangeKindOfDeployedAPI(apiConfig.Name, apiConfig.Kind, deployedResource.Kind)
	}

	telemetry.Event("operator.deploy", apiConfig.TelemetryEvent())

	var api *spec.API
	var msg string
	switch apiConfig.Kind {
	case userconfig.RealtimeAPIKind:
		api, msg, err = realtimeapi.UpdateAPI(apiConfig, force)
	case userconfig.BatchAPIKind:
		api, msg, err = batchapi.UpdateAPI(apiConfig)
	case userconfig.TaskAPIKind:
		api, msg, err = taskapi.UpdateAPI(apiConfig)
	case userconfig.AsyncAPIKind:
		api, msg, err = asyncapi.UpdateAPI(*apiConfig, force)
	case userconfig.TrafficSplitterKind:
		api, msg, err = trafficsplitter.UpdateAPI(apiConfig)
	default:
		return nil, "", ErrorOperationIsOnlySupportedForKind(
			*deployedResource, userconfig.RealtimeAPIKind,
			userconfig.AsyncAPIKind,
			userconfig.BatchAPIKind,
			userconfig.TrafficSplitterKind,
			userconfig.TaskAPIKind,
		) // unexpected
	}

	if err == nil && api != nil {
		apiEndpoint, _ := operator.APIEndpoint(api)

		return &schema.APIResponse{
			Spec:     api,
			Endpoint: &apiEndpoint,
		}, msg, nil
	}

	return nil, msg, err
}

func RefreshAPI(apiName string, force bool) (string, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return "", err
	}

	switch deployedResource.Kind {
	case userconfig.RealtimeAPIKind:
		return realtimeapi.RefreshAPI(apiName, force)
	case userconfig.AsyncAPIKind:
		return asyncapi.RefreshAPI(apiName, force)
	default:
		return "", ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind, userconfig.AsyncAPIKind)
	}
}

func DeleteAPI(apiName string, keepCache bool) (*schema.DeleteResponse, error) {
	deployedResource, err := GetDeployedResourceByNameOrNil(apiName)
	if err != nil {
		return nil, err
	}
	if deployedResource == nil {
		// Delete anyways just to be sure everything is deleted
		routines.RunWithPanicHandler(func() {
			err := parallel.RunFirstErr(
				func() error {
					return realtimeapi.DeleteAPI(apiName, keepCache)
				},
				func() error {
					return batchapi.DeleteAPI(apiName, keepCache)
				},
				func() error {
					return trafficsplitter.DeleteAPI(apiName, keepCache)
				},
				func() error {
					return taskapi.DeleteAPI(apiName, keepCache)
				},
				func() error {
					return asyncapi.DeleteAPI(apiName, keepCache)
				},
			)
			if err != nil {
				telemetry.Error(err)
			}
		})
		return nil, ErrorAPINotDeployed(apiName)
	}

	switch deployedResource.Kind {
	case userconfig.RealtimeAPIKind:
		err := checkIfUsedByTrafficSplitter(apiName)
		if err != nil {
			return nil, err
		}
		err = realtimeapi.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	case userconfig.TrafficSplitterKind:
		err := trafficsplitter.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	case userconfig.BatchAPIKind:
		err := batchapi.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	case userconfig.TaskAPIKind:
		err := taskapi.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	case userconfig.AsyncAPIKind:
		err = asyncapi.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind, userconfig.AsyncAPIKind, userconfig.BatchAPIKind, userconfig.TrafficSplitterKind) // unexpected
	}

	return &schema.DeleteResponse{
		Message: fmt.Sprintf("deleting %s", apiName),
	}, nil
}

func GetAPIs() ([]schema.APIResponse, error) {
	var deployments []kapps.Deployment
	var k8sTaskJobs []kbatch.Job
	var taskAPIPods []kcore.Pod
	var virtualServices []*istioclientnetworking.VirtualService
	var batchJobList batch.BatchJobList

	err := parallel.RunFirstErr(
		func() error {
			var err error
			deployments, err = config.K8s.ListDeploymentsWithLabelKeys("apiName")
			return err
		},
		func() error {
			var err error
			taskAPIPods, err = config.K8s.ListPodsByLabel("apiKind", userconfig.TaskAPIKind.String())
			return err
		},
		func() error {
			var err error
			k8sTaskJobs, err = config.K8s.ListJobs(
				&kmeta.ListOptions{
					LabelSelector: klabels.SelectorFromSet(
						map[string]string{
							"apiKind": userconfig.TaskAPIKind.String(),
						},
					).String(),
				},
			)
			return err
		},
		func() error {
			var err error
			virtualServices, err = config.K8s.ListVirtualServicesWithLabelKeys("apiName")
			return err
		},
		func() error {
			return config.K8s.List(context.Background(), &batchJobList)
		},
	)
	if err != nil {
		return nil, err
	}

	var realtimeAPIDeployments []kapps.Deployment
	var asyncAPIDeployments []kapps.Deployment
	for _, deployment := range deployments {
		switch deployment.Labels["apiKind"] {
		case userconfig.RealtimeAPIKind.String():
			realtimeAPIDeployments = append(realtimeAPIDeployments, deployment)
		case userconfig.AsyncAPIKind.String():
			asyncAPIDeployments = append(asyncAPIDeployments, deployment)
		}
	}

	var batchAPIVirtualServices []*istioclientnetworking.VirtualService
	var taskAPIVirtualServices []*istioclientnetworking.VirtualService
	var trafficSplitterVirtualServices []*istioclientnetworking.VirtualService

	for _, vs := range virtualServices {
		switch vs.Labels["apiKind"] {
		case userconfig.BatchAPIKind.String():
			batchAPIVirtualServices = append(batchAPIVirtualServices, vs)
		case userconfig.TrafficSplitterKind.String():
			trafficSplitterVirtualServices = append(trafficSplitterVirtualServices, vs)
		case userconfig.TaskAPIKind.String():
			taskAPIVirtualServices = append(taskAPIVirtualServices, vs)
		}
	}

	realtimeAPIList, err := realtimeapi.GetAllAPIs(realtimeAPIDeployments)
	if err != nil {
		return nil, err
	}

	var taskAPIList []schema.APIResponse
	taskAPIList, err = taskapi.GetAllAPIs(taskAPIVirtualServices, k8sTaskJobs, taskAPIPods)
	if err != nil {
		return nil, err
	}

	batchAPIList, err := batchapi.GetAllAPIs(batchAPIVirtualServices, batchJobList.Items)
	if err != nil {
		return nil, err
	}

	asyncAPIList, err := asyncapi.GetAllAPIs(asyncAPIDeployments)
	if err != nil {
		return nil, err
	}

	trafficSplitterList, err := trafficsplitter.GetAllAPIs(trafficSplitterVirtualServices)
	if err != nil {
		return nil, err
	}

	response := make([]schema.APIResponse, 0, len(realtimeAPIList)+len(batchAPIList)+len(trafficSplitterList))

	response = append(response, realtimeAPIList...)
	response = append(response, batchAPIList...)
	response = append(response, taskAPIList...)
	response = append(response, asyncAPIList...)
	response = append(response, trafficSplitterList...)

	return response, nil
}

func GetAPI(apiName string) ([]schema.APIResponse, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return nil, err
	}

	var apiResponse []schema.APIResponse

	switch deployedResource.Kind {
	case userconfig.RealtimeAPIKind:
		apiResponse, err = realtimeapi.GetAPIByName(deployedResource)
		if err != nil {
			return nil, err
		}
	case userconfig.BatchAPIKind:
		apiResponse, err = batchapi.GetAPIByName(deployedResource)
		if err != nil {
			return nil, err
		}
	case userconfig.TaskAPIKind:
		apiResponse, err = taskapi.GetAPIByName(deployedResource)
		if err != nil {
			return nil, err
		}
	case userconfig.AsyncAPIKind:
		apiResponse, err = asyncapi.GetAPIByName(deployedResource)
		if err != nil {
			return nil, err
		}
	case userconfig.TrafficSplitterKind:
		apiResponse, err = trafficsplitter.GetAPIByName(deployedResource)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrorOperationIsOnlySupportedForKind(
			*deployedResource,
			userconfig.RealtimeAPIKind, userconfig.BatchAPIKind,
			userconfig.TaskAPIKind, userconfig.TrafficSplitterKind,
			userconfig.AsyncAPIKind,
		) // unexpected
	}

	// Get past API deploy times
	if len(apiResponse) > 0 {
		apiResponse[0].APIVersions, err = getPastAPIDeploys(deployedResource.Name)
		if err != nil {
			return nil, err
		}
	}

	return apiResponse, nil
}

func GetAPIByID(apiName string, apiID string) ([]schema.APIResponse, error) {
	// check if the API is currently running, so that additional information can be returned
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err == nil && deployedResource != nil && deployedResource.ID() == apiID {
		return GetAPI(apiName)
	}

	// search for the API spec with the old ID
	apiSpec, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		if aws.IsGenericNotFoundErr(err) {
			return nil, ErrorAPIIDNotFound(apiName, apiID)
		}
		return nil, err
	}

	return []schema.APIResponse{
		{
			Spec: apiSpec,
		},
	}, nil
}

func getPastAPIDeploys(apiName string) ([]schema.APIVersion, error) {
	var apiVersions []schema.APIVersion

	apiIDs, err := config.AWS.ListS3DirOneLevel(config.ClusterConfig.Bucket, spec.KeysPrefix(apiName, config.ClusterConfig.ClusterUID), pointer.Int64(10), nil)
	if err != nil {
		return nil, err
	}

	for _, apiID := range apiIDs {
		lastUpdated, err := spec.TimeFromAPIID(apiID)
		if err != nil {
			return nil, err
		}
		apiVersions = append(apiVersions, schema.APIVersion{
			APIID:       apiID,
			LastUpdated: lastUpdated.Unix(),
		})
	}

	return apiVersions, nil
}

// checkIfUsedByTrafficSplitter checks if api is used by a deployed TrafficSplitter
func checkIfUsedByTrafficSplitter(apiName string) error {
	virtualServices, err := config.K8s.ListVirtualServicesByLabel("apiKind", userconfig.TrafficSplitterKind.String())
	if err != nil {
		return err
	}

	var usedByTrafficSplitters []string
	for _, vs := range virtualServices {
		trafficSplitterSpec, err := operator.DownloadAPISpec(vs.Labels["apiName"], vs.Labels["apiID"])
		if err != nil {
			return err
		}
		for _, api := range trafficSplitterSpec.APIs {
			if apiName == api.Name {
				usedByTrafficSplitters = append(usedByTrafficSplitters, trafficSplitterSpec.Name)
			}
		}
	}
	if len(usedByTrafficSplitters) > 0 {
		return ErrorAPIUsedByTrafficSplitter(usedByTrafficSplitters)
	}
	return nil
}

func DescribeAPI(apiName string) ([]schema.APIResponse, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return nil, err
	}

	var apiResponse []schema.APIResponse

	switch deployedResource.Kind {
	case userconfig.RealtimeAPIKind:
		apiResponse, err = realtimeapi.DescribeAPIByName(deployedResource)
		if err != nil {
			return nil, err
		}
	case userconfig.AsyncAPIKind:
		apiResponse, err = asyncapi.DescribeAPIByName(deployedResource)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrorOperationIsOnlySupportedForKind(
			*deployedResource,
			userconfig.RealtimeAPIKind,
			userconfig.AsyncAPIKind,
		) // unexpected
	}

	return apiResponse, nil
}
