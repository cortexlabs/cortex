/*
Copyright 2021 Cortex Labs, Inc.

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

	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job/taskapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func ReadJobLogs(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]
	jobID, err := getRequiredQueryParam("jobID", r)
	if err != nil {
		respondError(w, r, err)
		return
	}

	deployedResource, err := resources.GetDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}
	if deployedResource.Kind != userconfig.BatchAPIKind && deployedResource.Kind != userconfig.TaskAPIKind {
		respondError(w, r, resources.ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.BatchAPIKind, userconfig.TaskAPIKind))
		return
	}

	upgrader := websocket.Upgrader{}
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		respondError(w, r, err)
		return
	}
	defer socket.Close()

	labels := map[string]string{"apiName": apiName, "jobID": jobID}

	if deployedResource.Kind == userconfig.BatchAPIKind {
		labels["cortex.dev/batch"] = "worker"
	}

	operator.StreamLogsFromRandomPod(labels, socket)
}

func JobLogURL(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]
	jobID, err := getRequiredQueryParam("jobID", r)
	if err != nil {
		respondError(w, r, err)
		return
	}

	deployedResource, err := resources.GetDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}

	switch deployedResource.Kind {
	case userconfig.BatchAPIKind:
		jobStatus, err := batchapi.GetJobStatus(spec.JobKey{
			ID:      jobID,
			APIName: apiName,
			Kind:    userconfig.BatchAPIKind,
		})
		if err != nil {
			respondError(w, r, err)
			return
		}
		logURL, err := operator.BatchJobLogURL(apiName, *jobStatus)
		if err != nil {
			respondError(w, r, err)
			return
		}
		respond(w, schema.LogResponse{
			LogURL: logURL,
		})
	case userconfig.TaskAPIKind:
		jobStatus, err := taskapi.GetJobStatus(spec.JobKey{
			ID:      jobID,
			APIName: apiName,
			Kind:    userconfig.TaskAPIKind,
		})
		if err != nil {
			respondError(w, r, err)
			return
		}
		logURL, err := operator.TaskJobLogURL(apiName, *jobStatus)
		if err != nil {
			respondError(w, r, err)
			return
		}
		respond(w, schema.LogResponse{
			LogURL: logURL,
		})
	default:
		respondError(w, r, resources.ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.BatchAPIKind, userconfig.TaskAPIKind))
	}
}
