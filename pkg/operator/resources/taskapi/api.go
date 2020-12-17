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

package taskapi

import (
	"fmt"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
)

// UpdateAPI deploys or update a task api without triggering any task
func UpdateAPI(apiConfig *userconfig.API, projectID string) (*spec.API, string, error) {
	prevVirtualService, err := config.K8s.GetVirtualService(operator.K8sName(apiConfig.Name))
	if err != nil {
		return nil, "", err
	}

	api := spec.GetAPISpec(apiConfig, projectID, "", config.Cluster.ClusterName) // Deployment ID not needed for TaskAPI spec

	if prevVirtualService == nil {
		if err := config.AWS.UploadJSONToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		err = applyK8sResources(api, prevVirtualService)
		if err != nil {
			go func() {
				_ = deleteK8sResources(api.Name)
			}()
			return nil, "", err
		}

		err = operator.AddAPIToAPIGateway(*api.Networking.Endpoint, api.Networking.APIGateway)
		if err != nil {
			go func() {
				_ = deleteK8sResources(api.Name)
			}()
			go func() {
				_ = operator.RemoveAPIFromAPIGateway(*api.Networking.Endpoint, api.Networking.APIGateway)
			}()
			return nil, "", err
		}

		return api, fmt.Sprintf("created %s", api.Resource.UserString()), nil
	}

	if prevVirtualService.Labels["specID"] != api.SpecID {
		if err := config.AWS.UploadJSONToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		err = applyK8sResources(api, prevVirtualService)
		if err != nil {
			return nil, "", err
		}

		if err := operator.UpdateAPIGatewayK8s(prevVirtualService, api); err != nil {
			return nil, "", err
		}

		return api, fmt.Sprintf("updated %s", api.Resource.UserString()), nil
	}

	// TODO: add log group here

	return api, fmt.Sprintf("%s is up to date", api.Resource.UserString()), nil
}

// GetAllAPIs returns all task APIs, for each API returning the most recently submitted job and all running jobs
func GetAllAPIs(virtualServices []istioclientnetworking.VirtualService) ([]schema.APIResponse, error) {
	taskAPIsMap := map[string]*schema.APIResponse{}

	//jobIDToK8sJobMap := map[string]*kbatch.Job{}
	//for _, job := range k8sJobs {
	//	jobIDToK8sJobMap[job.Labels["jobID"]] = &job
	//}
	//
	//jobIDToPodsMap := map[string][]kcore.Pod{}
	//for _, pod := range pods {
	//	if pod.Labels["jobID"] != "" {
	//		jobIDToPodsMap[pod.Labels["jobID"]] = append(jobIDToPodsMap[pod.Labels["jobID"]], pod)
	//	}
	//}

	for _, virtualService := range virtualServices {
		apiName := virtualService.Labels["apiName"]
		apiID := virtualService.Labels["apiID"]

		api, err := operator.DownloadAPISpec(apiName, apiID)
		if err != nil {
			return nil, err
		}

		endpoint, err := operator.APIEndpoint(api)
		if err != nil {
			return nil, err
		}

		//jobStates, err := getMostRecentlySubmittedJobStates(apiName, 1)
		//
		//jobStatuses := []status.JobStatus{}
		//if len(jobStates) > 0 {
		//	jobStatus, err := getJobStatusFromJobState(jobStates[0], jobIDToK8sJobMap[jobStates[0].ID], jobIDToPodsMap[jobStates[0].ID])
		//	if err != nil {
		//		return nil, err
		//	}
		//
		//	jobStatuses = append(jobStatuses, *jobStatus)
		//}

		taskAPIsMap[apiName] = &schema.APIResponse{
			Spec:     *api,
			Endpoint: endpoint,
			//JobStatuses: jobStatuses,
		}
	}

	//inProgressJobKeys, err := listAllInProgressJobKeys()
	//if err != nil {
	//	return nil, err
	//}
	//
	//for _, jobKey := range inProgressJobKeys {
	//	alreadyAdded := false
	//	for _, jobStatus := range taskAPIsMap[jobKey.APIName].JobStatuses {
	//		if jobStatus.ID == jobKey.ID {
	//			alreadyAdded = true
	//			break
	//		}
	//	}
	//
	//	if alreadyAdded {
	//		continue
	//	}
	//
	//	jobStatus, err := getJobStatusFromK8sJob(jobKey, jobIDToK8sJobMap[jobKey.ID], jobIDToPodsMap[jobKey.ID])
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	if jobStatus.Status.IsInProgress() {
	//		taskAPIsMap[jobKey.APIName].JobStatuses = append(taskAPIsMap[jobKey.APIName].JobStatuses, *jobStatus)
	//	}
	//}

	batchAPIList := make([]schema.APIResponse, 0, len(taskAPIsMap))

	for _, batchAPI := range taskAPIsMap {
		batchAPIList = append(batchAPIList, *batchAPI)
	}

	return batchAPIList, nil
}

// DeleteAPI deletes a task api
func DeleteAPI(apiName string, keepCache bool) error {
	// best effort deletion, so don't handle error yet
	virtualService, vsErr := config.K8s.GetVirtualService(operator.K8sName(apiName))

	err := parallel.RunFirstErr(
		func() error {
			return vsErr
		},
		func() error {
			return deleteK8sResources(apiName)
		},
		func() error {
			if keepCache {
				return nil
			}
			return deleteS3Resources(apiName)
		},
		func() error {
			err := operator.RemoveAPIFromAPIGatewayK8s(virtualService)
			if err != nil {
				return err
			}

			return nil
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func deleteS3Resources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			prefix := filepath.Join(config.Cluster.ClusterName, "apis", apiName)
			return config.AWS.DeleteS3Dir(config.Cluster.Bucket, prefix, true)
		},
		//func() error {
		//	prefix := spec.BatchAPIJobPrefix(apiName, config.Cluster.ClusterName)
		//	go func() {
		//		_ = config.AWS.DeleteS3Dir(config.Cluster.Bucket, prefix, true) // deleting job files may take a while
		//	}()
		//	return nil
		//},
		//func() error {
		//	deleteAllInProgressFilesByAPI(apiName) // not useful xml error is thrown, swallow the error
		//	return nil
		//},
	)
}
