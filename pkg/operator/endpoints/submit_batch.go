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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/resources"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
)

func SubmitBatchJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiName := vars["apiName"]
	dryRun := getOptionalBoolQParam("dryRun", false, r)

	deployedResource, err := resources.GetDeployedResourceByName(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}
	if deployedResource.Kind != userconfig.BatchAPIKind {
		respondError(w, r, resources.ErrorOperationIsOnlySupportedForKind(*deployedResource, userconfig.BatchAPIKind))
		return
	}

	// max payload size, same as API Gateway
	rw := http.MaxBytesReader(w, r.Body, 10<<20)

	bodyBytes, err := ioutil.ReadAll(rw)
	if err != nil {
		respondError(w, r, err)
		return
	}

	submission := schema.BatchJobSubmission{}

	err = json.Unmarshal(bodyBytes, &submission)
	if err != nil {
		respondError(w, r, errors.Append(err, fmt.Sprintf("\n\njob submission schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor)))
		return
	}

	if dryRun {
		// plain text response for dry run because it is typically consumed by people
		w.Header().Set("Content-type", "text/plain")

		fileNames, err := batchapi.DryRun(&submission)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, "\n"+err.Error()+"\n")
			return
		}

		for _, fileName := range fileNames {
			_, err := io.WriteString(w, fileName+"\n")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, "\n"+err.Error()+"\n")
				return
			}
		}

		_, _ = io.WriteString(w, "validations passed")
		return
	}

	jobSpec, err := batchapi.SubmitJob(apiName, &submission)
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, jobSpec)
}
