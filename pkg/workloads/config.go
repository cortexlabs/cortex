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

package workloads

import (
	"encoding/json"

	"github.com/cortexlabs/cortex/pkg/types/spec"
	kcore "k8s.io/api/core/v1"
)

type ConfigMapConfig struct {
	BatchJob *spec.BatchJob
	TaskJob  *spec.TaskJob
	Probes   map[string]kcore.Probe
}

func (c *ConfigMapConfig) GenerateConfigMapData() (map[string]string, error) {
	if c == nil {
		return nil, nil
	}

	data := map[string]string{}
	if len(c.Probes) > 0 {
		probesEncoded, err := json.MarshalIndent(c.Probes, "", "\t")
		if err != nil {
			return nil, err
		}

		data["probes.json"] = string(probesEncoded)
	}

	if c.TaskJob != nil {
		jobSpecEncoded, err := json.MarshalIndent(*c.TaskJob, "", "\t")
		if err != nil {
			return nil, err
		}

		data["job_spec.json"] = string(jobSpecEncoded)
	}

	if c.BatchJob != nil {
		jobSpecEncoded, err := json.MarshalIndent(*c.BatchJob, "", "\t")
		if err != nil {
			return nil, err
		}

		data["job_spec.json"] = string(jobSpecEncoded)
	}

	return data, nil
}
