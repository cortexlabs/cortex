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
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources"
	"github.com/cortexlabs/cortex/pkg/operator/resources/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
)

func GetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiName := vars["apiName"]
	jobID := vars["jobID"]

	deployedResource, err := resources.GetDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}
	if deployedResource.Kind != userconfig.BatchAPIKind {
		respondError(w, r, resources.ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.BatchAPIKind))
		return
	}

	jobKey := spec.JobKey{APIName: apiName, ID: jobID}

	jobStatus, err := batchapi.GetJobStatus(jobKey)
	if err != nil {
		respondError(w, r, err)
		return
	}

	spec, err := operator.DownloadAPISpec(jobStatus.APIName, jobStatus.APIID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	endpoint, err := operator.APIEndpoint(spec)
	if err != nil {
		respondError(w, r, err)
		return
	}

	response := schema.GetJobResponse{
		JobStatus: *jobStatus,
		APISpec:   *spec,
		Endpoint:  urls.Join(endpoint, jobKey.ID),
	}

	respond(w, response)
}
