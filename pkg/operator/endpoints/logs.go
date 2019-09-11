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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
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

	workloadID := getOptionalQParam("workloadID", r)
	resourceID := getOptionalQParam("resourceID", r)
	resourceName := getOptionalQParam("resourceName", r)
	resourceType := getOptionalQParam("resourceType", r)

	podLabels := map[string]string{
		"appName":    appName,
		"userFacing": "true",
	}

	if workloadID != "" {
		podLabels["workloadID"] = workloadID
		readLogs(w, r, podLabels, appName)
		return
	}

	if resourceID != "" {
		workloadID, err = workloads.GetLatestWorkloadID(resourceID, appName)
		if err != nil {
			RespondError(w, err)
			return
		}
		if workloadID == "" {
			RespondError(w, errors.Wrap(workloads.ErrorNotFound(), appName, "latest workload ID", resourceID))
			return
		}

		podLabels["workloadID"] = workloadID
		readLogs(w, r, podLabels, appName)
		return
	}

	if resourceName == "" {
		RespondError(w, ErrorAnyQueryParamRequired("workloadID", "resourceID", "resourceName"))
		return
	}

	if resourceType != "" {
		res, err := ctx.VisibleResourceByNameAndType(resourceName, resourceType)
		if err != nil {
			RespondError(w, err)
			return
		}
		if res.GetResourceType() == resource.APIType {
			podLabels["apiName"] = res.GetName()
		} else {
			podLabels["workloadID"] = res.GetWorkloadID()
		}
		readLogs(w, r, podLabels, appName)
		return
	}

	res, err := ctx.VisibleResourceByName(resourceName)

	if err == nil {
		workloadID = res.GetWorkloadID()
		if res.GetResourceType() == resource.APIType {
			podLabels["apiName"] = res.GetName()
		} else {
			podLabels["workloadID"] = res.GetWorkloadID()
		}
		readLogs(w, r, podLabels, appName)
		return
	}

	// Check for duplicate resources with same workload ID
	var workloadIDs []string
	for _, res := range ctx.VisibleResourcesByName(resourceName) {
		workloadIDs = append(workloadIDs, res.GetWorkloadID())
	}
	workloadIDs = slices.UniqueStrings(workloadIDs)
	if len(workloadIDs) == 1 {
		podLabels["workloadID"] = workloadIDs[0]
		readLogs(w, r, podLabels, appName)
		return
	}

	RespondError(w, err)
	return
}

func readLogs(w http.ResponseWriter, r *http.Request, podLabels map[string]string, appName string) {
	upgrader := websocket.Upgrader{}
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		RespondError(w, err)
		return
	}
	defer socket.Close()

	workloads.ReadLogs(appName, podLabels, socket)
}
