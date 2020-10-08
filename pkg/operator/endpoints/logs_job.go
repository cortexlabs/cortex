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

	"github.com/cortexlabs/cortex/pkg/operator/resources"
	"github.com/cortexlabs/cortex/pkg/operator/resources/batchapi"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func ReadJobLogs(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]
	jobID := mux.Vars(r)["jobID"]

	deployedResource, err := resources.GetDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}
	if deployedResource.Kind != userconfig.BatchAPIKind {
		respondError(w, r, resources.ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.BatchAPIKind))
		return
	}

	upgrader := websocket.Upgrader{}
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		respondError(w, r, err)
		return
	}
	defer socket.Close()

	batchapi.ReadLogs(spec.JobKey{
		APIName: deployedResource.Name,
		ID:      jobID,
	}, socket)
}
