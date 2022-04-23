/*
Copyright 2022 Cortex Labs, Inc.

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

package autoscaler

import (
	"encoding/json"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type Handler struct {
	autoscaler *Autoscaler
}

func NewHandler(autoscaler *Autoscaler) *Handler {
	return &Handler{autoscaler: autoscaler}
}

func (h *Handler) Awaken(w http.ResponseWriter, r *http.Request) {
	var api userconfig.Resource
	if err := json.NewDecoder(r.Body).Decode(&api); err != nil {
		http.Error(w, "failed to json decode request body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	if err := h.autoscaler.Awaken(api); err != nil {
		http.Error(w, errors.Wrap(err, "failed to awaken api").Error(), http.StatusInternalServerError)
		telemetry.Error(err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
