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

package batchapi

import (
	"fmt"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kbatch "k8s.io/api/batch/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

func UpdateAPI(apiConfig *userconfig.API, projectID string) (*spec.API, string, error) {
	prevVirtualService, err := getVirtualService(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	api := spec.GetAPISpec(apiConfig, projectID, "") // Deployment ID not needed for BatchAPI spec

	if prevVirtualService == nil {
		if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		err = applyK8sResources(api, prevVirtualService)
		if err != nil {
			go deleteK8sResources(api.Name)
			return nil, "", err
		}

		err = operator.AddAPIToAPIGateway(*api.Networking.Endpoint, api.Networking.APIGateway, true)
		if err != nil {
			operator.RemoveAPIFromAPIGateway(*api.Networking.Endpoint, api.Networking.APIGateway, true)
			return nil, "", err
		}
		return api, fmt.Sprintf("creating %s", api.Name), nil
	}

	if !areAPIsEqual(prevVirtualService, virtualServiceSpec(api)) {
		if err := config.AWS.UploadMsgpackToS3(api, config.Cluster.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		err = applyK8sResources(api, prevVirtualService)
		if err != nil {
			go deleteK8sResources(api.Name)
			return nil, "", err
		}

		if err := operator.UpdateAPIGatewayK8s(prevVirtualService, api, true); err != nil {
			operator.RemoveAPIFromAPIGateway(*api.Networking.Endpoint, api.Networking.APIGateway, true)
			go deleteK8sResources(api.Name) // Delete k8s if update fails?
			return nil, "", err
		}

		if err := operator.UpdateAPIGatewayK8s(prevVirtualService, api, true); err != nil {
			go deleteK8sResources(api.Name) // Delete k8s if update fails?
			return nil, "", err
		}
		return api, fmt.Sprintf("updating %s", api.Name), nil
	}

	return api, fmt.Sprintf("%s is up to date", api.Name), nil
}

func areAPIsEqual(v1, v2 *istioclientnetworking.VirtualService) bool {
	return v1.Labels["apiName"] == v2.Labels["apiName"] &&
		v1.Labels["apiID"] == v2.Labels["apiID"] &&
		operator.DoCortexAnnotationsMatch(v1, v2)
}

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
			deleteS3Resources(apiName)
			return nil
		},
		func() error {
			queues, _ := listQueuesPerAPI(apiName)
			for _, queueURL := range queues {
				deleteQueue(queueURL)
			}
			return nil
		},
		func() error {
			err := operator.RemoveAPIFromAPIGatewayK8s(virtualService, true)
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

func deleteK8sResources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			_, err := config.K8s.DeleteJobs(&kmeta.ListOptions{
				LabelSelector: klabels.SelectorFromSet(map[string]string{"apiName": apiName}).String(),
			})
			return err
		},
		func() error {
			_, err := config.K8s.DeleteVirtualService(operator.K8sName(apiName))
			return err
		},
	)
}

func deleteS3Resources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			prefix := filepath.Join("apis", apiName)
			return config.AWS.DeleteS3Dir(config.Cluster.Bucket, prefix, true)
		},
		func() error {
			prefix := spec.APIJobPrefix(apiName)
			return config.AWS.DeleteS3Dir(config.Cluster.Bucket, prefix, true)
		},
	)
}

func GetAllAPIs(virtualServices []istioclientnetworking.VirtualService, k8sJobs []kbatch.Job) ([]schema.BatchAPI, error) {
	batchAPIsMap := map[string]*schema.BatchAPI{}

	k8sJobMap := map[string]*kbatch.Job{}
	for _, job := range k8sJobs {
		k8sJobMap[job.Labels["jobID"]] = &job
	}

	for _, virtualService := range virtualServices {
		apiName := virtualService.GetLabels()["apiName"]
		apiID := virtualService.GetLabels()["apiID"]
		api, err := operator.DownloadAPISpec(apiName, apiID)
		if err != nil {
			return nil, err

		}

		baseURL, err := operator.APIBaseURL(api)
		if err != nil {
			return nil, err
		}

		jobStates, err := getMostRecentlySubmittedJobStates(apiName, 1)

		jobStatuses := []status.JobStatus{}
		if len(jobStates) != 0 {
			jobStatus, err := getJobStatusFromJobState(jobStates[0], k8sJobMap[jobStates[0].ID])
			if err != nil {
				return nil, err
			}

			jobStatuses = append(jobStatuses, *jobStatus)
		}

		batchAPIsMap[apiName] = &schema.BatchAPI{
			APISpec:     *api,
			BaseURL:     baseURL,
			JobStatuses: jobStatuses,
		}
	}

	inProgressJobIDs, err := listAllInProgressJobs()
	if err != nil {
		return nil, err
	}

	for _, jobKey := range inProgressJobIDs {
		alreadyAdded := false
		for _, jobStatus := range batchAPIsMap[jobKey.APIName].JobStatuses {
			if jobStatus.ID == jobKey.ID {
				alreadyAdded = true
				break
			}
		}

		if alreadyAdded {
			continue
		}

		jobStatus, err := getJobStatusFromK8sJob(jobKey, k8sJobMap[jobKey.ID])
		if err != nil {
			return nil, err
		}

		if jobStatus.Status.IsInProgressPhase() {
			batchAPIsMap[jobKey.APIName].JobStatuses = append(batchAPIsMap[jobKey.APIName].JobStatuses, *jobStatus)
		}
	}

	batchAPIList := make([]schema.BatchAPI, len(batchAPIsMap))

	i := 0
	for _, batchAPI := range batchAPIsMap {
		batchAPIList[i] = *batchAPI
		i++
	}

	return batchAPIList, nil
}

func GetAPIByName(apiName string) (*schema.GetAPIResponse, error) {
	virtualService, err := config.K8s.GetVirtualService(operator.K8sName(apiName))
	if err != nil {
		return nil, err
	}

	apiID := virtualService.GetLabels()["apiID"]
	api, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	k8sJobs, err := config.K8s.ListJobsByLabel("apiName", apiName)
	if err != nil {
		return nil, err
	}

	baseURL, err := operator.APIBaseURL(api)
	if err != nil {
		return nil, err

	}

	k8sJobMap := map[string]*kbatch.Job{}
	for _, job := range k8sJobs {
		k8sJobMap[job.Labels["jobID"]] = &job
	}

	inProgressJobIDs, err := listAllInProgressJobsByAPI(apiName)
	if err != nil {
		return nil, err
	}

	jobStatuses := []status.JobStatus{}
	jobIDSet := strset.New()
	for _, jobKey := range inProgressJobIDs {
		jobStatus, err := getJobStatusFromK8sJob(jobKey, k8sJobMap[jobKey.ID])
		if err != nil {
			return nil, err
		}

		jobStatuses = append(jobStatuses, *jobStatus)
		jobIDSet.Add(jobKey.ID)
	}

	if len(jobStatuses) < 10 {
		jobStates, err := getMostRecentlySubmittedJobStates(apiName, 10)
		if err != nil {
			return nil, err
		}
		for _, jobState := range jobStates {
			if jobIDSet.Has(jobState.ID) {
				continue
			}
			jobIDSet.Add(jobState.ID)

			jobStatus, err := getJobStatusFromJobState(jobState, nil)
			if err != nil {
				return nil, err
			}
			jobStatuses = append(jobStatuses, *jobStatus)

			if len(jobStatuses) >= 10 {
				break
			}
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
