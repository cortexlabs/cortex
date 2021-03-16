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
	predictorID := deployedResource.VirtualService.Labels["predictorID"]

	upgrader := websocket.Upgrader{}
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		respondError(w, r, err)
		return
	}
	defer socket.Close()

	labels := map[string]string{"apiName": apiName, "deploymentID": deploymentID, "predictorID": predictorID}

	if deployedResource.Kind == userconfig.AsyncAPIKind {
		labels["cortex.dev/async"] = "api"
	}
	operator.StreamLogsFromRandomPod(labels, socket)
}
