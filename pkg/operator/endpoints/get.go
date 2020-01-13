/*
Copyright 2019 Cortex Labs, Inc.

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
	"time"

	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
)

func GetAPIs(w http.ResponseWriter, r *http.Request) {
	statuses, err := operator.GetAllStatuses()
	if err != nil {
		respondError(w, err)
		return
	}

	apiNames, apiIDs := namesAndIDsFromStatuses(statuses)
	apis, err := operator.DownloadAPISpecs(apiNames, apiIDs)
	if err != nil {
		respondError(w, err)
		return
	}

	allMetrics, err := operator.GetMultipleMetrics(apis)
	if err != nil {
		respondError(w, err)
		return
	}

	baseURL, err := operator.APIsBaseURL()
	if err != nil {
		respondError(w, err)
		return
	}

	// TOOD CLI should handle no APIs
	Respond(w, schema.GetAPIsResponse{
		APIs: apis,
		Statuses: statuses,
		AllMetrics: allMetrics,
		BaseURL: baseURL,
	})
}

func GetAPI(w http.ResponseWriter, r *http.Request) {
	apiName, err := getRequiredQueryParam("apiName", r)
	if err != nil {
		respondError(w, err)
		return
	}

	status, err := operator.GetStatus(apiName)
	if err != nil {
		respondError(w, err)
		return
	}

	if status == nil {
		respondErrorCode(w, http.StatusNotFound, ErrorAPINotDeployed(apiName))
		return
	}

	api, err := operator.DownloadAPISpec(status.APIName, status.APIID)
	if err != nil {
		respondError(w, err)
		return
	}

	metrics, err := operator.GetMetrics(&api)
	if err != nil {
		respondError(w, err)
		return
	}

	baseURL, err := operator.APIsBaseURL()
	if err != nil {
		respondError(w, err)
		return
	}

	Respond(w, schema.GetAPIsResponse{
		API: api,
		Status: status,
		Metrics: metrics,
		BaseURL: baseURL,
	})
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
