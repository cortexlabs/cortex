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

package endpoints

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/cloud"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	ok8s "github.com/cortexlabs/cortex/pkg/operator/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/resources"
	"github.com/cortexlabs/cortex/pkg/operator/resources/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/syncapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kapps "k8s.io/api/apps/v1"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
)

func GetAPIs(w http.ResponseWriter, r *http.Request) {
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
			virtualServices, err = config.K8s.ListVirtualServicesByLabel("apiKind", "batch_api")
			return err
		},
	)
	if err != nil {
		respondError(w, r, err)
		return
	}

	statuses, err := syncapi.GetAllStatuses(deployments, pods)
	if err != nil {
		respondError(w, r, err)
		return
	}

	apiNames, apiIDs := namesAndIDsFromStatuses(statuses)
	apis, err := cloud.DownloadAPISpecs(apiNames, apiIDs)
	if err != nil {
		respondError(w, r, err)
		return
	}

	allMetrics, err := syncapi.GetMultipleMetrics(apis)
	if err != nil {
		respondError(w, r, err)
		return
	}

	syncAPIs := make([]schema.SyncAPI, len(apis))

	for i, api := range apis {
		baseURL, err := syncapi.APIBaseURL(&api)
		if err != nil {
			respondError(w, r, err)
			return
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
		api, err := cloud.DownloadAPISpec(apiName, apiID)
		if err != nil {
			respondError(w, r, err)
			return
		}

		baseURL, err := syncapi.APIBaseURL(api)
		if err != nil {
			respondError(w, r, err)
			return
		}

		batchAPIsMap[apiName] = &schema.BatchAPI{
			Spec:    *api,
			BaseURL: baseURL,
		}
	}

	for _, job := range jobs {
		apiName := job.Labels["apiName"]
		if _, ok := batchAPIsMap[apiName]; !ok {
			continue
		}

		jobID := job.Labels["jobID"]
		status, err := batchapi.GetJobStatus(apiName, jobID, pods)
		if err != nil {
			respondError(w, r, err)
			return
		}

		jobsForAPI := batchAPIsMap[apiName].Jobs
		jobsForAPI = append(jobsForAPI, *status)

		batchAPIsMap[apiName].Jobs = jobsForAPI
	}

	debug.Pp(batchAPIsMap)

	batchAPIList := make([]schema.BatchAPI, len(batchAPIsMap))

	i := 0
	for _, batchAPI := range batchAPIsMap {
		batchAPIList[i] = *batchAPI
		i++
	}

	debug.Pp(batchAPIList)

	respond(w, schema.GetAPIsResponse{
		BatchAPIs: batchAPIList,
		SyncAPIs:  syncAPIs,
	})
}

func GetAPI(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]

	resource, err := resources.FindDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if resource == nil {
		respondError(w, r, resources.ErrorAPINotDeployed(apiName)) // TODO return error when not find in FindDeployedResourceByName?
		return
	}

	if resource.Kind == userconfig.SyncAPIKind {
		apiResponse, err := getSyncAPI(apiName)
		if err != nil {
			respondError(w, r, err)
			return
		}
		respond(w, apiResponse) // TODO
		return
	} else if resource.Kind == userconfig.BatchAPIKind {
		apiResponse, err := getBatchAPI(apiName)
		if err != nil {
			respondError(w, r, err)
			return
		}
		respond(w, apiResponse) // TODO
		return
	}

	respond(w, "ok") // TODO
}

func getSyncAPI(apiName string) (*schema.GetAPIResponse, error) {
	status, err := syncapi.GetStatus(apiName)
	if err != nil {
		return nil, err
	}

	api, err := cloud.DownloadAPISpec(status.APIName, status.APIID)
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
	virtualService, err := config.K8s.GetVirtualService(ok8s.Name(apiName))
	if err != nil {
		return nil, err // TODO
	}
	apiID := virtualService.GetLabels()["apiID"]
	api, err := cloud.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	jobs, err := config.K8s.ListJobsByLabel("apiName", apiName)
	if err != nil {
		return nil, err
	}

	allBatchAPIPods, err := config.K8s.ListPodsByLabel("apiKind", userconfig.BatchAPIKind.String())
	if err != nil {
		return nil, err
	}

	baseURL, err := syncapi.APIBaseURL(api)
	if err != nil {
		return nil, err

	}

	jobIDSet := strset.New()
	jobStatuses := make([]status.JobStatus, len(jobs))
	for _, job := range jobs {
		jobID := job.Labels["jobID"]
		status, err := batchapi.GetJobStatus(apiName, jobID, allBatchAPIPods)
		if err != nil {
			return nil, err
		}
		jobIDSet.Add(jobID)
		jobStatuses = append(jobStatuses, *status)
	}

	if len(job) < 10 {
		objects, err := config.AWS.ListS3Prefix(*&config.Cluster.Bucket, batchapi.APIJobPrefix(apiName), false, pointer.Int64(10))
		if err != nil {
			return nil, err // TODO
		}
		if jobID strings.Split(*objects[0].Key, "/")[-1]
	}

	return &schema.GetAPIResponse{
		BatchAPI: &schema.BatchAPI{
			Spec:    *api,
			Jobs:    jobStatuses,
			BaseURL: baseURL,
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
