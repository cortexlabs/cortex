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
	"fmt"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/gorilla/mux"
)

func Delete(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]
	keepCache := getOptionalBoolQParam("keepCache", false, r)

	isDeployed, err := operator.IsAPIDeployed(apiName)
	if err != nil {
		respondError(w, err)
		return
	}

	if !isDeployed {
		// Delete anyways just to be sure everything is deleted
		go func() {
			err = operator.DeleteAPI(apiName, keepCache)
			if err != nil {
				telemetry.Error(err)
			}
		}()

		respondErrorCode(w, http.StatusNotFound, operator.ErrorAPINotDeployed(apiName))
		return
	}

	err = operator.DeleteAPI(apiName, keepCache)
	if err != nil {
		respondError(w, err)
		return
	}

	response := schema.DeleteResponse{
		Message: fmt.Sprintf("deleting %s api", apiName),
	}
	respond(w, response)
}
