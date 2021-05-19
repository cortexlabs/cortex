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

	"github.com/cortexlabs/cortex/pkg/proxy/probe"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
	logger, err := config.Build()
	require.NoError(t, err)

	log := logger.Sugar()

	return log
}

func TestDefaultProbeSuccess(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	server := httptest.NewServer(handler)
	pb := probe.NewDefaultProbe(log, server.URL)

	require.True(t, pb.ProbeContainer())
}

func TestDefaultProbeFailure(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	target := "http://127.0.0.1:12345"
	pb := probe.NewDefaultProbe(log, target)

	require.False(t, pb.ProbeContainer())
}

func TestProbeHTTPFailure(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	pb := probe.NewProbe(
		&kcore.Probe{
			Handler: kcore.Handler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromString("12345"),
					Host: "127.0.0.1",
				},
			},
			TimeoutSeconds: 3,
		}, log,
	)

	require.False(t, pb.ProbeContainer())
}

func TestProbeHTTPSuccess(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	server := httptest.NewServer(handler)
	targetURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	pb := probe.NewProbe(
		&kcore.Probe{
			Handler: kcore.Handler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromString(targetURL.Port()),
					Host: targetURL.Hostname(),
				},
			},
			TimeoutSeconds: 3,
		}, log,
	)

	require.True(t, pb.ProbeContainer())
}
