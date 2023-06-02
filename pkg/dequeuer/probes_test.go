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

package dequeuer

import (
	"testing"

	"github.com/cortexlabs/cortex/pkg/probe"
	"github.com/stretchr/testify/require"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestDefaultTCPProbeNotPresent(t *testing.T) {
	t.Parallel()

	log := newLogger(t)
	defer func() { _ = log.Sync() }()

	userPodPort := 8080

	probes := []*probe.Probe{
		probe.NewProbe(&kcore.Probe{
			ProbeHandler: kcore.ProbeHandler{
				Exec: &kcore.ExecAction{
					Command: []string{"/bin/bash", "python", "test.py"},
				},
			},
		}, log),
		probe.NewProbe(&kcore.Probe{
			ProbeHandler: kcore.ProbeHandler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "some-path",
					Port: intstr.FromInt(12345),
					Host: "localhost",
				},
			},
		}, log),
		probe.NewProbe(&kcore.Probe{
			ProbeHandler: kcore.ProbeHandler{
				TCPSocket: &kcore.TCPSocketAction{
					Port: intstr.FromInt(8447),
					Host: "localhost",
				},
			},
		}, log),
	}

	require.False(t, HasTCPProbeTargetingUserPod(probes, userPodPort))
}

func TestDefaultTCPProbePresent(t *testing.T) {
	t.Parallel()

	log := newLogger(t)
	defer func() { _ = log.Sync() }()

	userPodPort := intstr.FromInt(8080)

	probes := []*probe.Probe{
		probe.NewProbe(&kcore.Probe{
			ProbeHandler: kcore.ProbeHandler{
				Exec: &kcore.ExecAction{
					Command: []string{"/bin/bash", "python", "test.py"},
				},
			},
		}, log),
		probe.NewProbe(&kcore.Probe{
			ProbeHandler: kcore.ProbeHandler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "some-path",
					Port: intstr.FromInt(12345),
					Host: "localhost",
				},
			},
		}, log),
		probe.NewProbe(&kcore.Probe{
			ProbeHandler: kcore.ProbeHandler{
				TCPSocket: &kcore.TCPSocketAction{
					Port: userPodPort,
					Host: "localhost",
				},
			},
		}, log),
	}

	require.True(t, HasTCPProbeTargetingUserPod(probes, userPodPort.IntValue()))
}
