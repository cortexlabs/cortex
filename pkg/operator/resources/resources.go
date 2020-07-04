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
	"github.com/cortexlabs/cortex/pkg/operator/resources/syncapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/websocket"
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

func IsResourceUpdating(resource userconfig.Resource) (bool, error) {
	if resource.Kind == userconfig.SyncAPIKind {
		return syncapi.IsAPIUpdating(resource.Name)
	}

	return false, ErrorOperationNotSupportedForKind(resource.Kind)
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

	isProjectUploaded, err := config.AWS.IsS3File(config.Cluster.Bucket, projectKey)
	if err != nil {
		return nil, err
	}
	if !isProjectUploaded {
		if err = config.AWS.UploadBytesToS3(projectBytes, config.Cluster.Bucket, projectKey); err != nil {
			return nil, err
		}
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

	//order apiconfigs first syncAPIs then TrafficSplit
	apiConfigs = append(ApisWithoutAPISplitter(apiConfigs), ApisWithoutSyncAPI(apiConfigs)...)

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

	if apiConfig.Kind == userconfig.SyncAPIKind {
		return syncapi.UpdateAPI(apiConfig, projectID, force)
	}
	if apiConfig.Kind == userconfig.APISplitterKind {
		return apisplitter.UpdateAPI(apiConfig, projectID, force)
	}

	return nil, "", ErrorOperationNotSupportedForKind(apiConfig.Kind) // unexpected
}

func RefreshAPI(apiName string, force bool) (string, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return "", err
	} else if deployedResource == nil {
		return "", ErrorAPINotDeployed(apiName)
	}

	if deployedResource.Kind == userconfig.SyncAPIKind {
		return syncapi.RefreshAPI(apiName, force)
	}

	return "", ErrorOperationNotSupportedForKind(deployedResource.Kind) // unexpected
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
			)
			if err != nil {
				telemetry.Error(err)
			}
		}()

		return nil, ErrorAPINotDeployed(apiName)
	}
	fmt.Println(deployedResource.Kind)
	if deployedResource.Kind == userconfig.SyncAPIKind {
		err := syncapi.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	}
	if deployedResource.Kind == userconfig.APISplitterKind {
		err := apisplitter.DeleteAPI(apiName, keepCache)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrorOperationNotSupportedForKind(deployedResource.Kind) // unexpected
	}

	return &schema.DeleteResponse{
		Message: fmt.Sprintf("deleting %s", apiName),
	}, nil
}

func StreamLogs(deployedResource userconfig.Resource, socket *websocket.Conn) error {
	if deployedResource.Kind == userconfig.SyncAPIKind {
		syncapi.ReadLogs(deployedResource.Name, socket)
	} else {
		return ErrorOperationNotSupportedForKind(deployedResource.Kind) // unexpected
	}

	return nil
}

func GetAPIs() (*schema.GetAPIsResponse, error) {
	statuses, err := syncapi.GetAllStatuses()
	if err != nil {
		return nil, err
	}

	apiNames, apiIDs := namesAndIDsFromStatuses(statuses)
	apis, err := operator.DownloadAPISpecs(apiNames, apiIDs)
	if err != nil {
		return nil, err
	}

	allMetrics, err := syncapi.GetMultipleMetrics(apis)
	if err != nil {
		return nil, err
	}

	syncAPIs := make([]schema.SyncAPI, len(apis))

	for i, api := range apis {
		syncAPIs[i] = schema.SyncAPI{
			Spec:    api,
			Status:  statuses[i],
			Metrics: allMetrics[i],
		}
	}

	return &schema.GetAPIsResponse{
		SyncAPIs: syncAPIs,
	}, nil
}

func GetAPI(apiName string) (*schema.GetAPIResponse, error) {
	deployedResource, err := GetDeployedResourceByName(apiName)
	if err != nil {
		return nil, err
	} else if deployedResource == nil {
		return nil, ErrorAPINotDeployed(apiName)
	}

	if deployedResource.Kind == userconfig.SyncAPIKind {

		status, err := syncapi.GetStatus(apiName)
		if err != nil {
			return nil, err
		}

		api, err := operator.DownloadAPISpec(status.APIName, status.APIID)
		if err != nil {
			return nil, err
		}

		metrics, err := syncapi.GetMetrics(api)
		if err != nil {
			return nil, err
		}

		baseURL, err := syncapi.APIBaseURL(api)
		if err != nil {
			return nil, err
		}

		return &schema.GetAPIResponse{
			SyncAPI: &schema.SyncAPI{
				Spec:         *api,
				Status:       *status,
				Metrics:      *metrics,
				BaseURL:      baseURL,
				DashboardURL: syncapi.DashboardURL(),
			},
		}, nil

	}

	if deployedResource.Kind == userconfig.APISplitterKind {
		fmt.Println(deployedResource)
		status, err := apisplitter.GetStatus(apiName)
		if err != nil {
			return nil, err
		}
		fmt.Println(status)
		api, err := operator.DownloadAPISpec(status.APIName, status.APIID)
		if err != nil {
			return nil, err
		}
		fmt.Println(api)

		baseURL, err := apisplitter.APIBaseURL(api)
		if err != nil {
			return nil, err
		}

		return &schema.GetAPIResponse{
			APISplitter: &schema.APISplitter{
				Spec:    *api,
				Status:  *status,
				BaseURL: baseURL,
			},
		}, nil

	}

	return nil, ErrorOperationNotSupportedForKind(deployedResource.Kind) // unexpected

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
