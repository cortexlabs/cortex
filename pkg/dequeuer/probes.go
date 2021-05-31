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

package dequeuer

import (
	"encoding/json"

	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/probe"
	"go.uber.org/zap"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ProbesFromFile(probesPath string, logger *zap.SugaredLogger) ([]*probe.Probe, error) {
	fileBytes, err := files.ReadFileBytes(probesPath)
	if err != nil {
		return nil, err
	}

	probesMap := map[string]kcore.Probe{}
	if err := json.Unmarshal(fileBytes, &probesMap); err != nil {
		return nil, err
	}

	probesSlice := []*probe.Probe{}
	for _, p := range probesMap {
		auxProbe := p
		probesSlice = append(probesSlice, probe.NewProbe(&auxProbe, logger))
	}
	return probesSlice, nil
}

func HasTCPProbeTargetingUserPod(probes []*probe.Probe, userPort intstr.IntOrString) bool {
	for _, probe := range probes {
		if probe == nil {
			continue
		}
		if probe.Handler.TCPSocket != nil && probe.Handler.TCPSocket.Port == userPort {
			return true
		}
	}
	return false
}
