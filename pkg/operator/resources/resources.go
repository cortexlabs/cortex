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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/realtimeapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/trafficsplitter"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
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

func Deploy(projectBytes []byte, configFileName string, configBytes []byte, force bool) (*schema.DeployResponse, error) {
	projectID := hash.Bytes(projectBytes)
	projectKey := spec.ProjectKey(projectID)
	projectFileMap, err := zip.UnzipMemToMem(projectBytes)
	if err != nil {
		return nil, err
	}

	projectFiles := ProjectFiles{
		ProjectByteMap: projectFileMap,
		ConfigFileName: configFileName,
	}

	apiConfigs, err := spec.ExtractAPIConfigs(configBytes, types.AWSProviderType, configFileName)
	if err != nil {
		return nil, err
	}

	err = ValidateClusterAPIs(apiConfigs, projectFiles)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found here:\n  → Realtime API: https://docs.cortex.dev/v/%s/deployments/realtime-api/api-configuration\n  → Batch API: https://docs.cortex.dev/v/%s/deployments/batch-api/api-configuration\n  → Traffic Splitter: https://docs.cortex.dev/v/%s/deployments/realtime-api/traffic-splitter", consts.CortexVersionMinor, consts.CortexVersionMinor, consts.CortexVersionMinor))
		return nil, err
	}

	isProjectUploaded, err := config.AWS.IsS3File(config.Cluster.Bucket, projectKey)
	if err != nil {
		return nil, err
	}
	if !isProjectUploaded {
		if err = config.AWS.UploadBytesToS3(projectBytes, config.Cluster.Bucket, projectKey); err != nil {
			return nil, err
		}
	}

	// This is done if user specifies RealtimeAPIs in same file as TrafficSplitter
	apiConfigs = append(ExclusiveFilterAPIsByKind(apiConfigs, userconfig.TrafficSplitterKind), InclusiveFilterAPIsByKind(apiConfigs, userconfig.TrafficSplitterKind)...)

	results := make([]schema.DeployResult, len(apiConfigs))
	for i, apiConfig := range apiConfigs {
		api, msg, err := UpdateAPI(&apiConfig, projectID, force)
		results[i].Message = msg
		if err != nil {
			results[i].Error = errors.Message(err)
		} else {
			results[i].API = *api
		}
	}

	return &schema.DeployResponse{
		Results: results,
	}, nil
}

func UpdateAPI(apiConfig *userconfig.API, projectID string, force bool) (*spec.API, string, error) {
	deployedResource, err := GetDeployedResourceByNameOrNil(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	if deployedResource != nil && deployedResource.Kind != apiConfig.Kind {
		return nil, "", ErrorCannotChangeKindOfDeployedAPI(apiConfig.Name, apiConfig.Kind, deployedResource.Kind)
	}

	switch apiConfig.Kind {
	case userconfig.RealtimeAPIKind:
		return realtimeapi.UpdateAPI(apiConfig, projectID, force)
	case userconfig.BatchAPIKind:
		return batchapi.UpdateAPI(apiConfig, projectID)
	case userconfig.TrafficSplitterKind:
		return trafficsplitter.UpdateAPI(apiConfig, projectID, force)
	default:
		return nil, "", ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind, userconfig.BatchAPIKind, userconfig.TrafficSplitterKind) // unexpected
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

func GetAPIs() (*schema.GetAPIsResponse, error) {
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

	batchAPIList, err := batchapi.GetAllAPIs(batchAPIVirtualServices, k8sJobs, batchAPIPods)
	if err != nil {
		return nil, err
	}

	trafficSplitterList, err := trafficsplitter.GetAllAPIs(trafficSplitterVirtualServices)
	if err != nil {
		return nil, err
	}
	return &schema.GetAPIsResponse{
		BatchAPIs:        batchAPIList,
		RealtimeAPIs:     realtimeAPIList,
		TrafficSplitters: trafficSplitterList,
	}, nil
}

func GetAPI(apiName string) (*schema.GetAPIResponse, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return nil, err
	}

	switch deployedResource.Kind {
	case userconfig.RealtimeAPIKind:
		return realtimeapi.GetAPIByName(deployedResource)
	case userconfig.BatchAPIKind:
		return batchapi.GetAPIByName(deployedResource)
	case userconfig.TrafficSplitterKind:
		return trafficsplitter.GetAPIByName(deployedResource)
	default:
		return nil, ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind, userconfig.BatchAPIKind) // unexpected
	}
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
