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
	defer func() { _ = log.Sync() }()

	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	server := httptest.NewServer(handler)
	pb := probe.NewDefaultProbe(server.URL, log)

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

	require.True(t, pb.IsHealthy())
}

func TestDefaultProbeFailure(t *testing.T) {
	t.Parallel()

	log := newLogger(t)
	defer func() { _ = log.Sync() }()

	target := "http://127.0.0.1:12345"
	pb := probe.NewDefaultProbe(target, log)

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

	require.False(t, pb.IsHealthy())
}

func TestProbeHTTPFailure(t *testing.T) {
	t.Parallel()

	log := newLogger(t)
	defer func() { _ = log.Sync() }()

	pb := probe.NewProbe(
		&kcore.Probe{
			ProbeHandler: kcore.ProbeHandler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromString("12345"),
					Host: "127.0.0.1",
				},
			},
			InitialDelaySeconds: 1,
			TimeoutSeconds:      3,
			PeriodSeconds:       1,
			SuccessThreshold:    1,
			FailureThreshold:    1,
		}, log,
	)

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

	require.False(t, pb.IsHealthy())
}

func TestProbeHTTPSuccess(t *testing.T) {
	t.Parallel()

	log := newLogger(t)
	defer func() { _ = log.Sync() }()

	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	server := httptest.NewServer(handler)
	targetURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	pb := probe.NewProbe(
		&kcore.Probe{
			ProbeHandler: kcore.ProbeHandler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromString(targetURL.Port()),
					Host: targetURL.Hostname(),
				},
			},
			InitialDelaySeconds: 1,
			TimeoutSeconds:      3,
			PeriodSeconds:       1,
			SuccessThreshold:    1,
			FailureThreshold:    1,
		}, log,
	)

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

	require.True(t, pb.IsHealthy())
}
