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
	"github.com/cortexlabs/cortex/pkg/operator/pb"
	"github.com/golang/protobuf/ptypes/empty"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func ReadLogs(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]

	isDeployed, err := operator.IsAPIDeployed(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	} else if !isDeployed {
		respondError(w, r, operator.ErrorAPINotDeployed(apiName))
		return
	}

	upgrader := websocket.Upgrader{}
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		respondError(w, r, err)
		return
	}
	defer socket.Close()

	operator.ReadLogs(apiName, socket)
}

// TODO
func (ep *endpoint) ReadLogs(empty *empty.Empty, srv pb.EndPointService_ReadLogsServer) error {
	return nil
}