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
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/syncapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
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
		return nil, "", ErrorOperationNotSupportedForKind(apiConfig.Kind) // unexpected
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
		return "", ErrorOperationNotSupportedForKind(deployedResource.Kind) // unexpected
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
		return nil, ErrorOperationNotSupportedForKind(deployedResource.Kind) // unexpected
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
		return ErrorOperationNotSupportedForKind(logRequest.Kind) // unexpected
	}

	return nil
}

func GetAPIs() (*schema.GetAPIsResponse, error) {
	var deployments []kapps.Deployment
	var jobs []kbatch.Job
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
			jobs, err = config.K8s.ListJobsWithLabelKeys("apiName")
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

	statuses, err := syncapi.GetAllStatuses(deployments, pods)
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
		baseURL, err := syncapi.APIBaseURL(&api)
		if err != nil {
			return nil, err

		}

		syncAPIs[i] = schema.SyncAPI{
			Spec:    api,
			Status:  statuses[i],
			Metrics: allMetrics[i],
			BaseURL: baseURL,
		}
	}

	batchAPIsMap := map[string]*schema.BatchAPI{}

	fmt.Println(len(virtualServices))

	for _, virtualService := range virtualServices {
		apiName := virtualService.GetLabels()["apiName"]
		apiID := virtualService.GetLabels()["apiID"]
		api, err := operator.DownloadAPISpec(apiName, apiID)
		if err != nil {
			return nil, err

		}

		baseURL, err := syncapi.APIBaseURL(api)
		if err != nil {
			return nil, err

		}

		batchAPIsMap[apiName] = &schema.BatchAPI{
			APISpec:     *api,
			BaseURL:     baseURL,
			JobStatuses: []status.JobStatus{},
		}
	}

	for _, job := range jobs {
		// TODO jobs take a while to delete because it takes a while to terminate pods
		apiName := job.Labels["apiName"]
		if _, ok := batchAPIsMap[apiName]; !ok {
			continue
		}

		jobID := job.Labels["jobID"]
		jobStatus, err := batchapi.GetJobStatusFromK8sJob(spec.JobID{APIName: apiName, ID: jobID}, &job)
		if err != nil {
			return nil, err
		}

		batchAPIsMap[apiName].JobStatuses = append(batchAPIsMap[apiName].JobStatuses, *jobStatus)
	}

	debug.Pp(batchAPIsMap)

	batchAPIList := make([]schema.BatchAPI, len(batchAPIsMap))

	i := 0
	for _, batchAPI := range batchAPIsMap {
		batchAPIList[i] = *batchAPI
		i++
	}

	debug.Pp(batchAPIList)

	return &schema.GetAPIsResponse{
		BatchAPIs: batchAPIList,
		SyncAPIs:  syncAPIs,
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
		return getSyncAPI(apiName)
	case userconfig.BatchAPIKind:
		return getBatchAPI(apiName)
	default:
		return nil, ErrorOperationNotSupportedForKind(deployedResource.Kind) // unexpected
	}
}

func getSyncAPI(apiName string) (*schema.GetAPIResponse, error) {
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

func getBatchAPI(apiName string) (*schema.GetAPIResponse, error) {
	virtualService, err := config.K8s.GetVirtualService(operator.K8sName(apiName))
	if err != nil {
		return nil, err // TODO
	}
	apiID := virtualService.GetLabels()["apiID"]
	api, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	jobs, err := config.K8s.ListJobsByLabel("apiName", apiName)
	if err != nil {
		return nil, err
	}

	baseURL, err := syncapi.APIBaseURL(api)
	if err != nil {
		return nil, err

	}

	jobIDSet := strset.New()
	jobStatuses := make([]status.JobStatus, 0, len(jobs))
	for _, job := range jobs {
		jobID := job.Labels["jobID"]
		fmt.Println("running ", jobID)
		jobStatus, err := batchapi.GetJobStatusFromK8sJob(spec.JobID{APIName: apiName, ID: jobID}, &job)
		if err != nil {
			return nil, err
		}
		jobIDSet.Add(jobID)
		jobStatuses = append(jobStatuses, *jobStatus)
	}

	if len(jobStatuses) < 10 {
		objects, err := config.AWS.ListS3Prefix(*&config.Cluster.Bucket, batchapi.APIJobPrefix(apiName), false, nil)
		if err != nil {
			return nil, err // TODO
		}
		for _, s3Obj := range objects {
			pathSplit := strings.Split(*s3Obj.Key, "/")
			jobID := pathSplit[len(pathSplit)-2]
			if jobIDSet.Has(jobID) {
				continue
			}
			jobIDSet.Add(jobID)
			jobStatus, err := batchapi.GetJobStatus(spec.JobID{APIName: apiName, ID: jobID})
			if err != nil {
				return nil, err
			}
			jobStatuses = append(jobStatuses, *jobStatus)
		}
	}

	return &schema.GetAPIResponse{
		BatchAPI: &schema.BatchAPI{
			APISpec:     *api,
			JobStatuses: jobStatuses,
			BaseURL:     baseURL,
		},
	}, nil
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
