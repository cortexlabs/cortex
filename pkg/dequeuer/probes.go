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
	"github.com/cortexlabs/cortex/pkg/lib/files"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/probe"
	"go.uber.org/zap"
	kcore "k8s.io/api/core/v1"
)

func ProbesFromFile(probesPath string, logger *zap.SugaredLogger) ([]*probe.Probe, error) {
	fileBytes, err := files.ReadFileBytes(probesPath)
	if err != nil {
		return nil, err
	}

	probesMap := map[string]kcore.Probe{}
	if err := libjson.Unmarshal(fileBytes, &probesMap); err != nil {
		return nil, err
	}

	probesSlice := make([]*probe.Probe, len(probesMap))
	var i int
	for _, p := range probesMap {
		auxProbe := p
		probesSlice[i] = probe.NewProbe(&auxProbe, logger)
		i++
	}
	return probesSlice, nil
}

func HasTCPProbeTargetingUserPod(probes []*probe.Probe, userPort int) bool {
	for _, pb := range probes {
		if pb == nil {
			continue
		}
		if pb.ProbeHandler.TCPSocket != nil && pb.ProbeHandler.TCPSocket.Port.IntValue() == userPort {
			return true
		}
	}
	return false
}
