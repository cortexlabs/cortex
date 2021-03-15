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
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/lib/logging"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

var operatorLogger = logging.GetOperatorLogger()

func respond(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func respondPlainText(w http.ResponseWriter, response string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func respondError(w http.ResponseWriter, r *http.Request, err error, strs ...string) {
	respondErrorCode(w, r, http.StatusBadRequest, err, strs...)
}

func respondErrorCode(w http.ResponseWriter, r *http.Request, code int, err error, strs ...string) {
	err = errors.Wrap(err, strs...)

	if !errors.IsNoTelemetry(err) {
		errTags := map[string]string{}
		if clientID := r.Context().Value(ctxKeyClient); clientID != nil {
			if clientIDStr, ok := clientID.(string); ok {
				errTags["client_id"] = clientIDStr
			}
		}
		telemetry.Error(err, errTags)
	}

	if !errors.IsNoPrint(err) {
		operatorLogger.Error(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := schema.ErrorResponse{
		Kind:    errors.GetKind(err),
		Message: errors.Message(err),
	}
	json.NewEncoder(w).Encode(response)
}

func recoverAndRespond(w http.ResponseWriter, r *http.Request, strs ...string) {
	if errInterface := recover(); errInterface != nil {
		err := errors.CastRecoverError(errInterface, strs...)
		operatorLogger.Error(err)
		telemetry.Error(err)
		respondError(w, r, err)
	}
}
