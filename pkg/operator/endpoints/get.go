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

	"github.com/cortexlabs/cortex/pkg/operator/resources"
	"github.com/gorilla/mux"
)

func GetAPIs(w http.ResponseWriter, r *http.Request) {
	response, err := resources.GetAPIs()
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, response)
}

func GetAPI(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]

	response, err := resources.GetAPI(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, response)
}

func GetAPIByID(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]
	apiID := mux.Vars(r)["apiID"]

	response, err := resources.GetAPIByID(apiName, apiID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	respond(w, response)
}
