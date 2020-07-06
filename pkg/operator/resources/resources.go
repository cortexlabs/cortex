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
	"github.com/cortexlabs/cortex/pkg/operator/resources/syncapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/websocket"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kapps "k8s.io/api/apps/v1"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
)

func GetDeployedResourceByName(resourceName string) (*userconfig.Resource, error) {
	virtualService, err := config.K8s.GetVirtualService(operator.K8sName(resourceName))
	if err != nil {
		return nil, err
	}

	if virtualService == nil {
		return nil, nil
	}

	return &userconfig.Resource{
		Name: virtualService.Labels["apiName"],
		Kind: userconfig.KindFromString(virtualService.Labels["apiKind"]),
	}, nil
}

func Deploy(projectBytes []byte, configPath string, configBytes []byte, force bool) (*schema.DeployResponse, error) {
	projectID := hash.Bytes(projectBytes)
	projectKey := spec.ProjectKey(projectID)
	projectFileMap, err := zip.UnzipMemToMem(projectBytes)
	if err != nil {
		return nil, err
	}

	projectFiles := ProjectFiles{
		ProjectByteMap: projectFileMap,
		ConfigFilePath: configPath,
	}
	apiConfigs, err := spec.ExtractAPIConfigs(configBytes, types.AWSProviderType, projectFiles, configPath)
	if err != nil {
		return nil, err
	}

	err = ValidateClusterAPIs(apiConfigs, projectFiles)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found here: https://docs.cortex.dev/v/%s/deployments/api-configuration", consts.CortexVersionMinor))
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
	deployedResource, err := GetDeployedResourceByName(apiConfig.Name)
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
	default:
		return nil, "", ErrorOperationNotSupportedForKind(*deployedResource, userconfig.SyncAPIKind, userconfig.BatchAPIKind) // unexpected
	}
}

func RefreshAPI(apiName string, force bool) (string, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return "", err
	} else if deployedResource == nil {
		return "", ErrorAPINotDeployed(apiName)
	}

	switch deployedResource.Kind {
	case userconfig.SyncAPIKind:
		return syncapi.RefreshAPI(apiName, force)
	default:
		return "", ErrorOperationNotSupportedForKind(*deployedResource, userconfig.SyncAPIKind)
	}
}

func DeleteAPI(apiName string, keepCache bool) (*schema.DeleteResponse, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
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
			)
			if err != nil {
				telemetry.Error(err)
			}
		}()

		return nil, ErrorAPINotDeployed(apiName)
	}

	switch deployedResource.Kind {
	case userconfig.SyncAPIKind:
		err := syncapi.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	case userconfig.BatchAPIKind:
		err := batchapi.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrorOperationNotSupportedForKind(*deployedResource, userconfig.SyncAPIKind, userconfig.BatchAPIKind) // unexpected
	}

	return &schema.DeleteResponse{
		Message: fmt.Sprintf("deleting %s", apiName),
	}, nil
}

func StreamLogs(logRequest schema.LogRequest, socket *websocket.Conn) error {
	switch logRequest.Kind {
	case userconfig.SyncAPIKind:
		syncapi.ReadLogs(logRequest.Name, socket)
	case userconfig.BatchAPIKind:
		batchapi.ReadLogs(logRequest, socket)
	default:
		return ErrorOperationNotSupportedForKind(logRequest.Resource, userconfig.SyncAPIKind, userconfig.BatchAPIKind) // unexpected
	}

	return nil
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

	syncAPIList, err := syncapi.GetAllAPIs(pods, deployments)
	if err != nil {
		return nil, err
	}

	batchAPIList, err := batchapi.GetAllAPIs(virtualServices, k8sJobs)
	if err != nil {
		return nil, err
	}

	return &schema.GetAPIsResponse{
		BatchAPIs: batchAPIList,
		SyncAPIs:  syncAPIList,
	}, nil
}

func GetAPI(apiName string) (*schema.GetAPIResponse, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return nil, err
	} else if deployedResource == nil {
		return nil, ErrorAPINotDeployed(apiName)
	}

	switch deployedResource.Kind {
	case userconfig.SyncAPIKind:
		return syncapi.GetAPIByName(apiName)
	case userconfig.BatchAPIKind:
		return batchapi.GetAPIByName(apiName)
	default:
		return nil, ErrorOperationNotSupportedForKind(*deployedResource, userconfig.SyncAPIKind, userconfig.BatchAPIKind) // unexpected
	}
}
