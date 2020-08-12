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
	"github.com/cortexlabs/cortex/pkg/operator/resources/apisplitter"
	"github.com/cortexlabs/cortex/pkg/operator/resources/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/syncapi"
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
		err = errors.Append(err, fmt.Sprintf("\n\napi configuration schema for Sync API can be found at https://docs.cortex.dev/v/%s/deployments/syncapi/api-configuration and for Batch API at https://docs.cortex.dev/v/%s/deployments/batchapi/api-configuration", consts.CortexVersionMinor, consts.CortexVersionMinor))
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

	// order SyncAPIs apiconfigs first then APISplitters
	// This is done if user specifies SyncAPIs in same file as APISplitter
	apiConfigs = append(InclusiveFilterAPIsByKind(apiConfigs, userconfig.SyncAPIKind), InclusiveFilterAPIsByKind(apiConfigs, userconfig.APISplitterKind)...)

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
	case userconfig.SyncAPIKind:
		return syncapi.UpdateAPI(apiConfig, projectID, force)
	case userconfig.BatchAPIKind:
		return batchapi.UpdateAPI(apiConfig, projectID)
	case userconfig.APISplitterKind:
		return apisplitter.UpdateAPI(apiConfig, projectID, force)
	default:
		return nil, "", ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.SyncAPIKind, userconfig.BatchAPIKind, userconfig.APISplitterKind) // unexpected
	}
}

func RefreshAPI(apiName string, force bool) (string, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return "", err
	}

	switch deployedResource.Kind {
	case userconfig.SyncAPIKind:
		return syncapi.RefreshAPI(apiName, force)
	default:
		return "", ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.SyncAPIKind)
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
					return syncapi.DeleteAPI(apiName, keepCache)
				},
				func() error {
					return batchapi.DeleteAPI(apiName, keepCache)
				},
				func() error {
					return apisplitter.DeleteAPI(apiName, keepCache)
				},
			)
			if err != nil {
				telemetry.Error(err)
			}
		}()
		return nil, err
	}

	switch deployedResource.Kind {
	case userconfig.SyncAPIKind:
		err := checkIfUsedByAPISplitter(apiName)
		if err != nil {
			return nil, err
		}
		err = syncapi.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	case userconfig.APISplitterKind:
		err := apisplitter.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	case userconfig.BatchAPIKind:
		err := batchapi.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.SyncAPIKind, userconfig.BatchAPIKind, userconfig.APISplitterKind) // unexpected
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
			virtualServices, err = config.K8s.ListVirtualServicesByLabel("apiKind", userconfig.BatchAPIKind.String())
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	syncAPIPods := []kcore.Pod{}
	batchAPIPods := []kcore.Pod{}
	for _, pod := range pods {
		switch pod.Labels["apiKind"] {
		case userconfig.SyncAPIKind.String():
			syncAPIPods = append(syncAPIPods, pod)
		case userconfig.BatchAPIKind.String():
			batchAPIPods = append(batchAPIPods, pod)
		}
	}

	syncAPIList, err := syncapi.GetAllAPIs(syncAPIPods, deployments)
	if err != nil {
		return nil, err
	}

	batchAPIList, err := batchapi.GetAllAPIs(virtualServices, k8sJobs, batchAPIPods)
	if err != nil {
		return nil, err
	}

	apiSplitterList, err := apisplitter.GetAllAPIs(virtualServices)
	if err != nil {
		return nil, err
	}

	return &schema.GetAPIsResponse{
		BatchAPIs:    batchAPIList,
		SyncAPIs:     syncAPIList,
		APISplitters: apiSplitterList,
	}, nil
}

func GetAPI(apiName string) (*schema.GetAPIResponse, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return nil, err
	}

	switch deployedResource.Kind {
	case userconfig.SyncAPIKind:
		return syncapi.GetAPIByName(deployedResource)
	case userconfig.BatchAPIKind:
		return batchapi.GetAPIByName(deployedResource)
	case userconfig.APISplitterKind:
		return apisplitter.GetAPIByName(deployedResource)
	default:
		return nil, ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.SyncAPIKind, userconfig.BatchAPIKind) // unexpected
	}
}

// checks if api is used by a deployed APISplitter
func checkIfUsedByAPISplitter(apiName string) error {
	virtualServices, err := config.K8s.ListVirtualServicesByLabel("apiKind", userconfig.APISplitterKind.String())
	if err != nil {
		return err
	}

	var usedByAPISplitters []string
	for _, vs := range virtualServices {
		apiSplitterSpec, err := operator.DownloadAPISpec(vs.Labels["apiName"], vs.Labels["apiID"])
		if err != nil {
			return err
		}
		for _, api := range apiSplitterSpec.APIs {
			if apiName == api.Name {
				usedByAPISplitters = append(usedByAPISplitters, apiSplitterSpec.Name)
			}
		}
	}
	if len(usedByAPISplitters) > 0 {
		return ErrorAPIUsedByAPISplitter(usedByAPISplitters)
	}
	return nil
}
