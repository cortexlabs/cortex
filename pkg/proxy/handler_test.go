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

package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxyHandler(t *testing.T) {
	userContainerHost := "http://user-container.cortex.dev"
	originHost := "origin.cortex.dev"

	stats := &RequestStats{}

	var httpHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, r.Host, originHost)
		require.Equal(t, stats.inFlightCounter, uint64(1))
		w.WriteHeader(http.StatusOK)
	}

	server := httptest.NewServer(httpHandler)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	httpProxy := httputil.NewSingleHostReverseProxy(serverURL)
	handler := ProxyHandler(stats, httpProxy)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, userContainerHost, nil)
	r.Host = originHost

	handler(w, r)
}
