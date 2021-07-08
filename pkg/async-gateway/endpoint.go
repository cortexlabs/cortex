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

package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/async"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Endpoint wraps an async-gateway Service with HTTP logic
type Endpoint struct {
	service Service
	logger  *zap.SugaredLogger
}

// NewEndpoint creates and initializes a new Endpoint struct
func NewEndpoint(svc Service, logger *zap.SugaredLogger) *Endpoint {
	return &Endpoint{
		service: svc,
		logger:  logger,
	}
}

// CreateWorkload is a handler for the async-gateway service workload creation route
func (e *Endpoint) CreateWorkload(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("x-request-id")
	if requestID == "" {
		respondPlainText(w, http.StatusBadRequest, "error: missing x-request-id key in request header")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		respondPlainText(w, http.StatusBadRequest, "error: missing Content-Type key in request header")
		return
	}

	body := r.Body
	defer func() {
		_ = r.Body.Close()
	}()

	log := e.logger.With(zap.String("id", requestID), zap.String("contentType", contentType))

	id, err := e.service.CreateWorkload(requestID, body, contentType)
	if err != nil {
		respondPlainText(w, http.StatusInternalServerError, fmt.Sprintf("error: %v", err))
		logErrorWithTelemetry(log, errors.Wrap(err, "failed to create workload"))
		return
	}

	if err = respondJSON(w, http.StatusOK, CreateWorkloadResponse{ID: id}); err != nil {
		logErrorWithTelemetry(log, errors.Wrap(err, "failed to encode json response"))
		return
	}
}

// GetWorkload is a handler for the async-gateway service workload retrieval route
func (e *Endpoint) GetWorkload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		respondPlainText(w, http.StatusBadRequest, "error: missing request id in url path")
		return
	}

	log := e.logger.With(zap.String("id", id))

	res, err := e.service.GetWorkload(id)
	if err != nil {
		respondPlainText(w, http.StatusInternalServerError, fmt.Sprintf("error: %v", err))
		logErrorWithTelemetry(log, errors.Wrap(err, "failed to get workload"))
		return
	}
	if res.Status == async.StatusNotFound {
		respondPlainText(w, http.StatusNotFound, fmt.Sprintf("error: id %s not found", res.ID))
		logErrorWithTelemetry(log, errors.ErrorUnexpected(fmt.Sprintf("error: id %s not found", res.ID)))
		return
	}

	if err = respondJSON(w, http.StatusOK, res); err != nil {
		logErrorWithTelemetry(log, errors.Wrap(err, "failed to encode json response"))
		return
	}
}

func respondPlainText(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(message))
}

func respondJSON(w http.ResponseWriter, statusCode int, s interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(s)
}

func logErrorWithTelemetry(log *zap.SugaredLogger, err error) {
	if err != nil && !errors.IsNoTelemetry(err) {
		telemetry.Error(err)
	}
	log.Error(err)
}
