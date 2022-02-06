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

package endpoints

import (
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources"
	"github.com/cortexlabs/cortex/pkg/operator/resources/asyncapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/realtimeapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func ReadLogs(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]
	jobID := getOptionalQParam("jobID", r)

	if jobID != "" {
		ReadJobLogs(w, r)
		return
	}

	deployedResource, err := resources.GetDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if deployedResource.Kind == userconfig.BatchAPIKind || deployedResource.Kind == userconfig.TaskAPIKind {
		respondError(w, r, ErrorLogsJobIDRequired(*deployedResource))
		return
	} else if deployedResource.Kind != userconfig.RealtimeAPIKind && deployedResource.Kind != userconfig.AsyncAPIKind {
		respondError(w, r, resources.ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind))
		return
	}

	deploymentID := deployedResource.VirtualService.Labels["deploymentID"]
	podID := deployedResource.VirtualService.Labels["podID"]

	upgrader := websocket.Upgrader{}
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		respondError(w, r, err)
		return
	}
	defer socket.Close()

	labels := map[string]string{"apiName": apiName, "deploymentID": deploymentID, "podID": podID}

	operator.StreamLogsFromRandomPod(labels, socket)
}

func GetLogURL(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]
	jobID := getOptionalQParam("jobID", r)

	if jobID != "" {
		GetJobLogURL(w, r)
		return
	}

	deployedResource, err := resources.GetDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if deployedResource.Kind == userconfig.BatchAPIKind || deployedResource.Kind == userconfig.TaskAPIKind {
		respondError(w, r, ErrorLogsJobIDRequired(*deployedResource))
		return
	}

	switch deployedResource.Kind {
	case userconfig.AsyncAPIKind:
		apiResponse, err := asyncapi.GetAPIByName(deployedResource)
		if err != nil {
			respondError(w, r, err)
			return
		}
		if apiResponse[0].Spec == nil {
			respondError(w, r, errors.ErrorUnexpected("unable to get api spec", apiName))
		}
		logURL, err := operator.APILogURL(*apiResponse[0].Spec)
		if err != nil {
			respondError(w, r, err)
			return
		}
		respondJSON(w, r, schema.LogResponse{
			LogURL: logURL,
		})
	case userconfig.RealtimeAPIKind:
		apiResponse, err := realtimeapi.GetAPIByName(deployedResource)
		if err != nil {
			respondError(w, r, err)
			return
		}
		if apiResponse[0].Spec == nil {
			respondError(w, r, errors.ErrorUnexpected("unable to get api spec", apiName))
		}
		logURL, err := operator.APILogURL(*apiResponse[0].Spec)
		if err != nil {
			respondError(w, r, err)
			return
		}
		respondJSON(w, r, schema.LogResponse{
			LogURL: logURL,
		})
	default:
		respondError(w, r, resources.ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.RealtimeAPIKind, userconfig.AsyncAPIKind))
	}
}
