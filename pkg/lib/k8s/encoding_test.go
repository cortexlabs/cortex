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

package k8s_test

import (
	"encoding/json"
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestDecodeProbeSuccess(t *testing.T) {
	t.Parallel()

	expectedProbe := &kcore.Probe{
		PeriodSeconds:    1,
		TimeoutSeconds:   2,
		SuccessThreshold: 1,
		FailureThreshold: 1,
		Handler: kcore.Handler{
			TCPSocket: &kcore.TCPSocketAction{
				Host: "127.0.0.1",
				Port: intstr.FromString("8080"),
			},
		},
	}
	probeBytes, err := json.Marshal(expectedProbe)
	require.NoError(t, err)

	gotProbe, err := k8s.DecodeJSONProbe(string(probeBytes))
	require.NoError(t, err)

	require.Equal(t, expectedProbe, gotProbe)
}

func TestDecodeProbeFailure(t *testing.T) {
	t.Parallel()

	probeBytes, err := json.Marshal("blah")
	require.NoError(t, err)

	_, err = k8s.DecodeJSONProbe(string(probeBytes))
	require.Error(t, err)
}

func TestEncodeProbe(t *testing.T) {
	t.Parallel()

	pb := &kcore.Probe{
		SuccessThreshold: 1,
		Handler: kcore.Handler{
			TCPSocket: &kcore.TCPSocketAction{
				Host: "127.0.0.1",
				Port: intstr.FromString("8080"),
			},
		},
	}

	jsonProbe, err := k8s.EncodeJSONProbe(pb)
	require.NoError(t, err)

	wantProbe := `{"tcpSocket":{"port":"8080","host":"127.0.0.1"},"successThreshold":1}`
	require.Equal(t, wantProbe, jsonProbe)
}

func TestEncodeNilProbe(t *testing.T) {
	t.Parallel()

	jsonProbe, err := k8s.EncodeJSONProbe(nil)
	assert.Error(t, err)
	assert.Empty(t, jsonProbe)
}
