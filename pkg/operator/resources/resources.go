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

package resources

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/archive"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/realtimeapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/trafficsplitter"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
)

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
	virtualService, err := config.K8s.GetVirtualService(operator.K8sName(resourceName))
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

func Deploy(projectBytes []byte, configFileName string, configBytes []byte, force bool) ([]schema.DeployResult, error) {
	projectID := hash.Bytes(projectBytes)
	projectFileMap, err := archive.UnzipMemToMem(projectBytes)
	if err != nil {
		return nil, err
	}

	projectFiles := ProjectFiles{
		ProjectByteMap: projectFileMap,
	}

	var apiConfigs []userconfig.API
	if config.Provider == types.AWSProviderType {
		apiConfigs, err = spec.ExtractAPIConfigs(configBytes, config.Provider, configFileName, &config.Cluster.Config, nil)
		if err != nil {
			return nil, err
		}
	} else {
		apiConfigs, err = spec.ExtractAPIConfigs(configBytes, config.Provider, configFileName, nil, &config.GCPCluster.GCPConfig)
		if err != nil {
			return nil, err
		}
	}

	err = ValidateClusterAPIs(apiConfigs, projectFiles)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found here:\n  → Realtime API: https://docs.cortex.dev/v/%s/deployments/realtime-api/api-configuration\n  → Batch API: https://docs.cortex.dev/v/%s/deployments/batch-api/api-configuration\n  → Traffic Splitter: https://docs.cortex.dev/v/%s/deployments/realtime-api/traffic-splitter", consts.CortexVersionMinor, consts.CortexVersionMinor, consts.CortexVersionMinor))
		return nil, err
	}

	projectKey := spec.ProjectKey(projectID, config.ClusterName())
	isProjectUploaded, err := config.IsBucketFile(projectKey)
	if err != nil {
		return nil, err
	}
	if !isProjectUploaded {
		if err = config.UploadBytesToBucket(projectBytes, projectKey); err != nil {
			return nil, err
		}
	}

	// This is done if user specifies RealtimeAPIs in same file as TrafficSplitter
	apiConfigs = append(ExclusiveFilterAPIsByKind(apiConfigs, userconfig.TrafficSplitterKind), InclusiveFilterAPIsByKind(apiConfigs, userconfig.TrafficSplitterKind)...)

	results := make([]schema.DeployResult, 0, len(apiConfigs))
	for i := range apiConfigs {
		apiConfig := apiConfigs[i]
		api, msg, err := UpdateAPI(&apiConfig, projectID, force)

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

func UpdateAPI(apiConfig *userconfig.API, projectID string, force bool) (*schema.APIResponse, string, error) {
	deployedResource, err := GetDeployedResourceByNameOrNil(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	if deployedResource != nil && deployedResource.Kind != apiConfig.Kind {
		return nil, "", ErrorCannotChangeKindOfDeployedAPI(apiConfig.Name, apiConfig.Kind, deployedResource.Kind)
	}

	telemetry.Event("operator.deploy", apiConfig.TelemetryEvent(config.Provider))

	var api *spec.API
	var msg string
	switch apiConfig.Kind {
	case userconfig.RealtimeAPIKind:
		api, msg, err = realtimeapi.UpdateAPI(apiConfig, projectID, force)
	case userconfig.BatchAPIKind:
		api, msg, err = batchapi.UpdateAPI(apiConfig, projectID)
	case userconfig.TrafficSplitterKind:
		api, msg, err = trafficsplitter.UpdateAPI(apiConfig, force)
	default:
		return nil, "", ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind, userconfig.BatchAPIKind, userconfig.TrafficSplitterKind) // unexpected
	}

	if err == nil && api != nil {
		apiEndpoint, _ := operator.APIEndpoint(api)

		return &schema.APIResponse{
			Spec:     *api,
			Endpoint: apiEndpoint,
		}, msg, nil
	}

	return nil, msg, err
}

func Patch(configBytes []byte, configFileName string, force bool) ([]schema.DeployResult, error) {
	var apiConfigs []userconfig.API
	var err error

	if config.Provider == types.AWSProviderType {
		apiConfigs, err = spec.ExtractAPIConfigs(configBytes, config.Provider, configFileName, &config.Cluster.Config, nil)
		if err != nil {
			return nil, err
		}
	} else {
		apiConfigs, err = spec.ExtractAPIConfigs(configBytes, config.Provider, configFileName, nil, &config.GCPCluster.GCPConfig)
		if err != nil {
			return nil, err
		}
	}

	results := make([]schema.DeployResult, 0, len(apiConfigs))
	for i := range apiConfigs {
		apiConfig := &apiConfigs[i]
		result := schema.DeployResult{}

		apiSpec, msg, err := patchAPI(apiConfig, configFileName, force)
		if err == nil && apiSpec != nil {
			apiEndpoint, _ := operator.APIEndpoint(apiSpec)

			result.API = &schema.APIResponse{
				Spec:     *apiSpec,
				Endpoint: apiEndpoint,
			}
		}

		result.Message = msg
		if err != nil {
			result.Error = errors.ErrorStr(err)
		}

		results = append(results, result)
	}
	return results, nil
}

func patchAPI(apiConfig *userconfig.API, configFileName string, force bool) (*spec.API, string, error) {
	deployedResource, err := GetDeployedResourceByName(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	if deployedResource.Kind == userconfig.UnknownKind {
		return nil, "", ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind, userconfig.BatchAPIKind, userconfig.TrafficSplitterKind) // unexpected
	}

	var projectFiles ProjectFiles

	prevAPISpec, err := operator.DownloadAPISpec(deployedResource.Name, deployedResource.ID())
	if err != nil {
		return nil, "", err
	}

	if deployedResource.Kind != userconfig.TrafficSplitterKind {
		bytes, err := config.AWS.ReadBytesFromS3(config.Cluster.Bucket, prevAPISpec.ProjectKey)
		if err != nil {
			return nil, "", err
		}

		projectFileMap, err := archive.UnzipMemToMem(bytes)
		if err != nil {
			return nil, "", err
		}

		projectFiles = ProjectFiles{
			ProjectByteMap: projectFileMap,
		}
	}

	err = ValidateClusterAPIs([]userconfig.API{*apiConfig}, projectFiles)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found here:\n  → Realtime API: https://docs.cortex.dev/v/%s/deployments/realtime-api/api-configuration\n  → Batch API: https://docs.cortex.dev/v/%s/deployments/batch-api/api-configuration\n  → Traffic Splitter: https://docs.cortex.dev/v/%s/deployments/realtime-api/traffic-splitter", consts.CortexVersionMinor, consts.CortexVersionMinor, consts.CortexVersionMinor))
		return nil, "", err
	}

	switch deployedResource.Kind {
	case userconfig.RealtimeAPIKind:
		return realtimeapi.UpdateAPI(apiConfig, prevAPISpec.ProjectID, force)
	case userconfig.BatchAPIKind:
		return batchapi.UpdateAPI(apiConfig, prevAPISpec.ProjectID)
	default:
		return trafficsplitter.UpdateAPI(apiConfig, force)
	}
}

func RefreshAPI(apiName string, force bool) (string, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return "", err
	}

	switch deployedResource.Kind {
	case userconfig.RealtimeAPIKind:
		return realtimeapi.RefreshAPI(apiName, force)
	default:
		return "", ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind)
	}
}

func DeleteAPI(apiName string, keepCache bool) (*schema.DeleteResponse, error) {
	deployedResource, err := GetDeployedResourceByNameOrNil(apiName)
	if err != nil {
		return nil, err
	}
	if deployedResource == nil {
		// Delete anyways just to be sure everything is deleted
		go func() {
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
			)
			if err != nil {
				telemetry.Error(err)
			}
		}()
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
	default:
		return nil, ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind, userconfig.BatchAPIKind, userconfig.TrafficSplitterKind) // unexpected
	}

	return &schema.DeleteResponse{
		Message: fmt.Sprintf("deleting %s", apiName),
	}, nil
}

func GetAPIs() ([]schema.APIResponse, error) {
	var deployments []kapps.Deployment
	var k8sJobs []kbatch.Job
	var pods []kcore.Pod
	var virtualServices []istioclientnetworking.VirtualService

	err := parallel.RunFirstErr(
		func() error {
			var err error
			deployments, err = config.K8s.ListDeploymentsWithLabelKeys("apiName")
			return err
		},
		func() error {
			var err error
			pods, err = config.K8s.ListPodsWithLabelKeys("apiName")
			return err
		},
		func() error {
			var err error
			k8sJobs, err = config.K8s.ListJobsWithLabelKeys("apiName")
			return err
		},
		func() error {
			var err error
			virtualServices, err = config.K8s.ListVirtualServicesWithLabelKeys("apiName")
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	realtimeAPIPods := []kcore.Pod{}
	batchAPIPods := []kcore.Pod{}
	for _, pod := range pods {
		switch pod.Labels["apiKind"] {
		case userconfig.RealtimeAPIKind.String():
			realtimeAPIPods = append(realtimeAPIPods, pod)
		case userconfig.BatchAPIKind.String():
			batchAPIPods = append(batchAPIPods, pod)
		}
	}

	var batchAPIVirtualServices []istioclientnetworking.VirtualService
	var trafficSplitterVirtualServices []istioclientnetworking.VirtualService

	for _, vs := range virtualServices {
		switch vs.Labels["apiKind"] {
		case userconfig.BatchAPIKind.String():
			batchAPIVirtualServices = append(batchAPIVirtualServices, vs)
		case userconfig.TrafficSplitterKind.String():
			trafficSplitterVirtualServices = append(trafficSplitterVirtualServices, vs)
		}
	}

	realtimeAPIList, err := realtimeapi.GetAllAPIs(realtimeAPIPods, deployments)
	if err != nil {
		return nil, err
	}

	var batchAPIList []schema.APIResponse
	if config.Provider == types.AWSProviderType {
		batchAPIList, err = batchapi.GetAllAPIs(batchAPIVirtualServices, k8sJobs, batchAPIPods)
		if err != nil {
			return nil, err
		}
	}

	trafficSplitterList, err := trafficsplitter.GetAllAPIs(trafficSplitterVirtualServices)
	if err != nil {
		return nil, err
	}

	response := make([]schema.APIResponse, 0, len(realtimeAPIList)+len(batchAPIList)+len(trafficSplitterList))

	response = append(response, realtimeAPIList...)
	response = append(response, batchAPIList...)
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
	case userconfig.TrafficSplitterKind:
		apiResponse, err = trafficsplitter.GetAPIByName(deployedResource)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind, userconfig.BatchAPIKind) // unexpected
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
	spec, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		if aws.IsGenericNotFoundErr(err) {
			return nil, ErrorAPIIDNotFound(apiName, apiID)
		}
		return nil, err
	}

	return []schema.APIResponse{
		{
			Spec: *spec,
		},
	}, nil
}

func getPastAPIDeploys(apiName string) ([]schema.APIVersion, error) {
	var apiVersions []schema.APIVersion

	apiIDs, err := config.ListBucketDirOneLevel(spec.KeysPrefix(apiName, config.ClusterName()), pointer.Int64(10))
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

//checkIfUsedByTrafficSplitter checks if api is used by a deployed TrafficSplitter
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
