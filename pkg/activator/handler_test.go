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

package activator

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActivatorHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	log := newLogger(t)

	apiName := "test"
	act := &activator{
		apiActivators: map[string]*apiActivator{
			apiName: newAPIActivator(apiName, 1, 1),
		},
		logger: log,
	}

	ah := NewHandler(act, log)

	var callCount int
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		}),
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "http://fake.cortex.dev/api", nil)
	r.Header.Set(CortexAPINameHeader, apiName)
	r.Header.Set(CortexTargetServiceHeader, server.URL)

	ah.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, callCount)
}
