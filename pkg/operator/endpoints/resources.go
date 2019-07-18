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

	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
)

func GetResources(w http.ResponseWriter, r *http.Request) {
	appName, err := getRequiredQueryParam("appName", r)
	if RespondIfError(w, err) {
		return
	}

	ctx := workloads.CurrentContext(appName)
	if ctx == nil {
		RespondError(w, ErrorAppNotDeployed(appName))
		return
	}

	dataStatuses, err := workloads.GetCurrentDataStatuses(ctx)
	if RespondIfError(w, err) {
		return
	}

	apiStatuses, apiGroupStatuses, err := workloads.GetCurrentAPIAndGroupStatuses(dataStatuses, ctx)
	if RespondIfError(w, err) {
		return
	}

	apisBaseURL, err := workloads.APIsBaseURL()
	if RespondIfError(w, err) {
		return
	}

	response := schema.GetResourcesResponse{
		Context:          ctx,
		DataStatuses:     dataStatuses,
		APIStatuses:      apiStatuses,
		APIGroupStatuses: apiGroupStatuses,
		APIsBaseURL:      apisBaseURL,
	}

	Respond(w, response)
}
