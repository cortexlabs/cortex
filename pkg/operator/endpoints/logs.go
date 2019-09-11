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

	"github.com/gorilla/websocket"

	"github.com/cortexlabs/cortex/pkg/operator/workloads"
)

func ReadLogs(w http.ResponseWriter, r *http.Request) {
	appName, err := getRequiredQueryParam("appName", r)
	if err != nil {
		RespondError(w, err)
		return
	}

	ctx := workloads.CurrentContext(appName)
	if ctx == nil {
		RespondError(w, ErrorAppNotDeployed(appName))
		return
	}

	resourceName, err := getRequiredQueryParam("resourceName", r)
	if err != nil {
		RespondError(w, err)
		return
	}

	readLogs(w, r, appName, resourceName)
}

func readLogs(w http.ResponseWriter, r *http.Request, appName string, apiName string) {
	upgrader := websocket.Upgrader{}
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		RespondError(w, err)
		return
	}
	defer socket.Close()

	workloads.ReadLogs(appName, apiName, socket)
}
