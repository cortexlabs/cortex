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
	"net/url"
	"testing"

	"github.com/cortexlabs/cortex/pkg/proxy"
	"github.com/cortexlabs/cortex/pkg/proxy/probe"
	"github.com/stretchr/testify/require"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

func TestHandlerSuccessTCP(t *testing.T) {
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

func TestHandlerSuccessHTTP(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	headers := []kcore.HTTPHeader{
		{
			Name:  "X-Cortex-Blah",
			Value: "Blah",
		},
	}

	var userHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.Header.Get(proxy.UserAgentKey), proxy.KubeProbeUserAgentPrefix)
		for _, header := range headers {
			require.Equal(t, header.Value, r.Header.Get(header.Name))
		}

		w.WriteHeader(http.StatusOK)
	}
	server := httptest.NewServer(userHandler)
	targetURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	pb := probe.NewProbe(
		&kcore.Probe{
			Handler: kcore.Handler{
				HTTPGet: &kcore.HTTPGetAction{
					Path:        "/",
					Port:        intstr.FromString(targetURL.Port()),
					Host:        targetURL.Hostname(),
					HTTPHeaders: headers,
				},
			},
			TimeoutSeconds:   3,
			PeriodSeconds:    1,
			SuccessThreshold: 1,
			FailureThreshold: 3,
		}, log,
	)
	handler := probe.Handler(pb)

	r := httptest.NewRequest(http.MethodGet, "http://fake.cortex.dev/healthz", nil)
	w := httptest.NewRecorder()

	handler(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "healthy", w.Body.String())
}
