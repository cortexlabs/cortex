/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Modifications Copyright 2022 Cortex Labs, Inc.
*/

package proxy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cortexlabs/cortex/pkg/proxy"
	"github.com/stretchr/testify/require"
)

func TestNewReverseProxy(t *testing.T) {
	var isHandlerCalled bool
	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		isHandlerCalled = true
	}

	server := httptest.NewServer(handler)
	httpProxy := proxy.NewReverseProxy(server.URL, 1000, 1000)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://user-container.cortex.dev", nil)
	httpProxy.ServeHTTP(resp, req)

	require.True(t, isHandlerCalled)
}
