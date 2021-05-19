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

package probe

import (
	"encoding/json"
	"errors"

	kcore "k8s.io/api/core/v1"
)

// DecodeJSON takes a json serialised *kcore.Probe and returns a Probe or an error.
func DecodeJSON(jsonProbe string) (*kcore.Probe, error) {
	pb := &kcore.Probe{}
	if err := json.Unmarshal([]byte(jsonProbe), pb); err != nil {
		return nil, err
	}
	return pb, nil
}

// EncodeJSON takes *kcore.Probe object and returns marshalled Probe JSON string and an error.
func EncodeJSON(pb *kcore.Probe) (string, error) {
	if pb == nil {
		return "", errors.New("cannot encode nil probe")
	}

	probeJSON, err := json.Marshal(pb)
	if err != nil {
		return "", err
	}
	return string(probeJSON), nil
}
