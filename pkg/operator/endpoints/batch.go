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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources"
	"github.com/cortexlabs/cortex/pkg/operator/resources/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
)

func SubmitJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	apiName := vars["apiName"]

	deployedResource, err := resources.GetDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if deployedResource == nil {
		respondError(w, r, resources.ErrorAPINotDeployed(apiName))
		return
	}

	if deployedResource.Kind == userconfig.SyncAPIKind {
		respondError(w, r, resources.ErrorOperationNotSupportedForKind(*deployedResource, userconfig.BatchAPIKind))
		return
	}

	rw := http.MaxBytesReader(w, r.Body, 32<<10)

	bodyBytes, err := ioutil.ReadAll(rw)
	if err != nil {
		respondError(w, r, err)
		return
	}

	sub := userconfig.JobSubmission{}

	err = json.Unmarshal(bodyBytes, &sub)
	if err != nil {
		respondError(w, r, err)
		return
	}

	jobSpec, err := batchapi.SubmitJob(apiName, sub)
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, jobSpec)
}

func DeleteJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	err := batchapi.StopJob(spec.JobID{APIName: vars["apiName"], ID: vars["jobID"]})
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, schema.DeleteResponse{
		Message: fmt.Sprintf("deleted job %s", vars["jobID"]),
	})
}

func GetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiName := vars["apiName"]
	deployedResource, err := resources.GetDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if deployedResource == nil {
		respondError(w, r, resources.ErrorAPINotDeployed(apiName))
		return
	}

	if deployedResource.Kind == userconfig.SyncAPIKind {
		respondError(w, r, resources.ErrorOperationNotSupportedForKind(*deployedResource, userconfig.BatchAPIKind))
		return
	}

	jobID := spec.JobID{APIName: apiName, ID: vars["jobID"]}

	jobStatus, err := batchapi.GetJobStatus(jobID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	spec, err := operator.DownloadAPISpec(jobStatus.APIName, jobStatus.APIID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	response := schema.JobResponse{
		JobStatus: *jobStatus,
		APISpec:   *spec,
	}

	respond(w, response)
}
