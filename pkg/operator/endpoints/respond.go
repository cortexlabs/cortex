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
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func respond(w http.ResponseWriter, response interface{}) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func respondError(w http.ResponseWriter, err error, strs ...string) {
	respondErrorCode(w, http.StatusBadRequest, err, strs...)
}

func respondErrorCode(w http.ResponseWriter, code int, err error, strs ...string) {
	err = errors.Wrap(err, strs...)
	errors.PrintError(err)

	w.WriteHeader(code)
	errors.PrintStacktrace(err)
	response := schema.ErrorResponse{
		Kind:    errors.GetKind(err).String(),
		User:    errors.IsUser(err),
		Message: errors.Message(err),
	}
	json.NewEncoder(w).Encode(response)
}

func recoverAndRespond(w http.ResponseWriter, strs ...string) {
	if errInterface := recover(); errInterface != nil {
		err := errors.CastRecoverError(errInterface, strs...)
		telemetry.Error(err)
		respondError(w, err)
	}
}
