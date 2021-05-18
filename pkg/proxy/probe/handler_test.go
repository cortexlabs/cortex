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

package probe_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cortexlabs/cortex/pkg/proxy/probe"
	"github.com/stretchr/testify/require"
)

func TestHandlerFailure(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	pb := probe.NewDefaultProbe(log, "http://127.0.0.1:12345")
	handler := probe.Handler(pb)

	r := httptest.NewRequest(http.MethodGet, "http://fake.cortex.dev/healthz", nil)
	w := httptest.NewRecorder()

	handler(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Equal(t, "unhealthy", w.Body.String())
}

func TestHandlerSuccess(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	var userHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	server := httptest.NewServer(userHandler)

	pb := probe.NewDefaultProbe(log, server.URL)
	handler := probe.Handler(pb)

	r := httptest.NewRequest(http.MethodGet, "http://fake.cortex.dev/healthz", nil)
	w := httptest.NewRecorder()

	handler(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "healthy", w.Body.String())
}
