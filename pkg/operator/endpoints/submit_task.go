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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/resources"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job/taskapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
)

func SubmitTaskJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiName := vars["apiName"]

	deployedResource, err := resources.GetDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}
	if deployedResource.Kind != userconfig.TaskAPIKind {
		respondError(w, r, resources.ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.TaskAPIKind))
		return
	}

	// max payload size, same as API Gateway
	rw := http.MaxBytesReader(w, r.Body, 10<<20)

	bodyBytes, err := ioutil.ReadAll(rw)
	if err != nil {
		respondError(w, r, err)
		return
	}

	submission := schema.TaskJobSubmission{}

	err = json.Unmarshal(bodyBytes, &submission)
	if err != nil {
		respondError(w, r, errors.Append(err, fmt.Sprintf("\n\ntask job submission schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor)))
		return
	}

	jobSpec, err := taskapi.SubmitJob(apiName, &submission)
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, jobSpec)
}
