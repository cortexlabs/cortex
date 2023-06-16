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

package batchapi

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const _batchDashboardUID = "batchapi"

func UpdateAPI(apiConfig *userconfig.API) (*spec.API, string, error) {
	prevVirtualService, err := config.K8s.GetVirtualService(workloads.K8sName(apiConfig.Name))
	if err != nil {
		return nil, "", err
	}

	initialDeploymentTime := time.Now().UnixNano()
	if prevVirtualService != nil && prevVirtualService.Labels["initialDeploymentTime"] != "" {
		var err error
		initialDeploymentTime, err = k8s.ParseInt64Label(prevVirtualService, "initialDeploymentTime")
		if err != nil {
			return nil, "", err
		}
	}

	api := spec.GetAPISpec(apiConfig, initialDeploymentTime, "", config.ClusterConfig.ClusterUID) // Deployment ID not needed for BatchAPI spec

	if prevVirtualService == nil {
		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		err = applyK8sResources(api, prevVirtualService)
		if err != nil {
			routines.RunWithPanicHandler(func() {
				_ = deleteK8sResources(api.Name)
			})
			return nil, "", err
		}

		return api, fmt.Sprintf("created %s", api.Resource.UserString()), nil
	}

	if prevVirtualService.Labels["specID"] != api.SpecID {
		if err := config.AWS.UploadJSONToS3(api, config.ClusterConfig.Bucket, api.Key); err != nil {
			return nil, "", errors.Wrap(err, "upload api spec")
		}

		err = applyK8sResources(api, prevVirtualService)
		if err != nil {
			return nil, "", err
		}

		return api, fmt.Sprintf("updated %s", api.Resource.UserString()), nil
	}

	return api, fmt.Sprintf("%s is up to date", api.Resource.UserString()), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	// best effort deletion, so don't handle error yet
	err := parallel.RunFirstErr(
		func() error {
			return deleteK8sResources(apiName)
		},
		func() error {
			if keepCache {
				return nil
			}
			return deleteS3Resources(apiName)
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
			prefix := filepath.Join(config.ClusterConfig.ClusterUID, "apis", apiName)
			return config.AWS.DeleteS3Dir(config.ClusterConfig.Bucket, prefix, true)
		},
		func() error {
			prefix := spec.JobAPIPrefix(config.ClusterConfig.ClusterUID, userconfig.BatchAPIKind, apiName)
			routines.RunWithPanicHandler(func() {
				_ = config.AWS.DeleteS3Dir(config.ClusterConfig.Bucket, prefix, true) // deleting job files may take a while
			})
			return nil
		},
	)
}

// GetAllAPIs returns all batch apis, for each API returning the most recently submitted job and all running jobs
func GetAllAPIs(virtualServices []*istioclientnetworking.VirtualService, batchJobList []batch.BatchJob) ([]schema.APIResponse, error) {
	batchAPIsMap := map[string]*schema.APIResponse{}

	jobIDToBatchJobMap := map[string]*batch.BatchJob{}
	apiNameToBatchJobsMap := map[string][]*batch.BatchJob{}
	for i, batchJob := range batchJobList {
		jobIDToBatchJobMap[batchJob.Name] = &batchJobList[i]
		apiNameToBatchJobsMap[batchJob.Spec.APIName] = append(apiNameToBatchJobsMap[batchJob.Spec.APIName], &batchJobList[i])
	}

	for i := range virtualServices {
		apiName := virtualServices[i].Labels["apiName"]
		metadata, err := spec.MetadataFromVirtualService(virtualServices[i])
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("api %s", apiName))
		}

		var jobStatuses []status.BatchJobStatus
		batchJobs := apiNameToBatchJobsMap[metadata.Name]

		if len(batchJobs) == 0 {
			jobStates, err := job.GetMostRecentlySubmittedJobStates(metadata.Name, 1, userconfig.BatchAPIKind)
			if err != nil {
				return nil, err
			}

			if len(jobStates) > 0 {
				jobStatus, err := getJobStatusFromJobState(jobStates[0])
				if err != nil {
					return nil, err
				}

				jobStatuses = append(jobStatuses, *jobStatus)
			}
		} else {
			for i := range batchJobs {
				batchJob := batchJobs[i]
				jobStatus, err := getJobStatusFromBatchJob(*batchJob)
				if err != nil {
					return nil, err
				}

				jobStatuses = append(jobStatuses, *jobStatus)
			}
		}

		batchAPIsMap[metadata.Name] = &schema.APIResponse{
			Metadata:         metadata,
			BatchJobStatuses: jobStatuses,
		}
	}

	batchAPIList := make([]schema.APIResponse, 0, len(batchAPIsMap))

	for _, batchAPI := range batchAPIsMap {
		batchAPIList = append(batchAPIList, *batchAPI)
	}

	return batchAPIList, nil
}

func GetAPIByName(deployedResource *operator.DeployedResource) ([]schema.APIResponse, error) {
	metadata, err := spec.MetadataFromVirtualService(deployedResource.VirtualService)
	if err != nil {
		return nil, err
	}

	api, err := operator.DownloadAPISpec(deployedResource.Name, metadata.APIID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	batchJobList := batch.BatchJobList{}
	if err = config.K8s.List(
		ctx, &batchJobList,
		client.InNamespace(config.K8s.Namespace),
		client.MatchingLabels{"apiName": deployedResource.Name},
	); err != nil {
		return nil, err
	}

	endpoint, err := operator.APIEndpoint(api)
	if err != nil {
		return nil, err
	}

	var jobStatuses []status.BatchJobStatus
	jobIDSet := strset.New()
	for _, batchJob := range batchJobList.Items {
		jobStatus, err := getJobStatusFromBatchJob(batchJob)
		if err != nil {
			return nil, err
		}
		jobStatuses = append(jobStatuses, *jobStatus)
		jobIDSet.Add(batchJob.Name)
	}

	if len(jobStatuses) < 10 {
		jobStates, err := job.GetMostRecentlySubmittedJobStates(deployedResource.Name, 10+len(jobStatuses), userconfig.BatchAPIKind)
		if err != nil {
			return nil, err
		}
		for _, jobState := range jobStates {
			if jobIDSet.Has(jobState.ID) {
				continue
			}
			jobIDSet.Add(jobState.ID)

			jobStatus, err := getJobStatusFromJobState(jobState)
			if err != nil {
				return nil, err
			}

			if jobStatus != nil {
				jobStatuses = append(jobStatuses, *jobStatus)
				if len(jobStatuses) == 10 {
					break
				}
			}
		}
	}

	dashboardURL := pointer.String(getDashboardURL(api.Name))

	return []schema.APIResponse{
		{
			Spec:             api,
			Metadata:         metadata,
			BatchJobStatuses: jobStatuses,
			Endpoint:         &endpoint,
			DashboardURL:     dashboardURL,
		},
	}, nil
}

func getDashboardURL(apiName string) string {
	loadBalancerURL, err := operator.LoadBalancerURL()
	if err != nil {
		return ""
	}

	dashboardURL := fmt.Sprintf(
		"%s/dashboard/d/%s/batchapi?orgId=1&refresh=30s&var-api_name=%s",
		loadBalancerURL, _batchDashboardUID, apiName,
	)

	return dashboardURL
}
