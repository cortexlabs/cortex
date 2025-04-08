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

package probe_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/cortexlabs/cortex/pkg/probe"
	"github.com/stretchr/testify/require"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func generateHandler(pb *probe.Probe) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !pb.IsHealthy() {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("unhealthy"))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("healthy"))
	}
}

func TestHandlerSuccessTCP(t *testing.T) {
	t.Parallel()

	log := newLogger(t)
	defer func() { _ = log.Sync() }()

	var userHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	server := httptest.NewServer(userHandler)

	pb := probe.NewDefaultProbe(server.URL, log)
	handler := generateHandler(pb)

	r := httptest.NewRequest(http.MethodGet, "http://fake.cortex.dev/healthz", nil)
	w := httptest.NewRecorder()

	stopper := pb.StartProbing()
	defer func() {
		stopper <- struct{}{}
	}()

	for {
		if pb.HasRunOnce() {
			break
		}
		time.Sleep(time.Second)
	}

	handler(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "healthy", w.Body.String())
}

func TestHandlerSuccessHTTP(t *testing.T) {
	t.Parallel()

	log := newLogger(t)
	defer func() { _ = log.Sync() }()

	headers := []kcore.HTTPHeader{
		{
			Name:  "X-Cortex-Blah",
			Value: "Blah",
		},
	}

	var userHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		require.True(t, probe.IsRequestKubeletProbe(r))
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
			ProbeHandler: kcore.ProbeHandler{
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
	handler := generateHandler(pb)

	r := httptest.NewRequest(http.MethodGet, "http://fake.cortex.dev/healthz", nil)
	w := httptest.NewRecorder()

	stopper := pb.StartProbing()
	defer func() {
		stopper <- struct{}{}
	}()

	for {
		if pb.HasRunOnce() {
			break
		}
		time.Sleep(time.Second)
	}

	handler(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "healthy", w.Body.String())
}
