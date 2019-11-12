/*
Copyright 2019 Cortex Labs, Inc.

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

/*
https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
https://kubernetes.io/docs/tasks/configure-pod-container/assign-memory-resource/
https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/
Note: A t2.medium (4GiB memory) has ~ 4,000,000,000 bytes allocatable (i.e. base 10, not base 2)
*/

package userconfig

import (
	"bytes"
	"fmt"
	"strings"

	kresource "k8s.io/apimachinery/pkg/api/resource"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type APICompute struct {
	MinReplicas          int32         `json:"min_replicas" yaml:"min_replicas"`
	MaxReplicas          int32         `json:"max_replicas" yaml:"max_replicas"`
	InitReplicas         int32         `json:"init_replicas" yaml:"init_replicas"`
	TargetCPUUtilization int32         `json:"target_cpu_utilization" yaml:"target_cpu_utilization"`
	CPU                  k8s.Quantity  `json:"cpu" yaml:"cpu"`
	Mem                  *k8s.Quantity `json:"mem" yaml:"mem"`
	GPU                  int64         `json:"gpu" yaml:"gpu"`
}

var apiComputeFieldValidation = &cr.StructFieldValidation{
	StructField: "Compute",
	StructValidation: &cr.StructValidation{
		StructFieldValidations: []*cr.StructFieldValidation{
			{
				StructField: "MinReplicas",
				Int32Validation: &cr.Int32Validation{
					Default:     1,
					GreaterThan: pointer.Int32(0),
				},
			},
			{
				StructField: "MaxReplicas",
				Int32Validation: &cr.Int32Validation{
					Default:     100,
					GreaterThan: pointer.Int32(0),
				},
			},
			{
				StructField:  "InitReplicas",
				DefaultField: "MinReplicas",
				Int32Validation: &cr.Int32Validation{
					GreaterThan: pointer.Int32(0),
				},
			},
			{
				StructField: "TargetCPUUtilization",
				Int32Validation: &cr.Int32Validation{
					Default:           80,
					GreaterThan:       pointer.Int32(0),
					LessThanOrEqualTo: pointer.Int32(100),
				},
			},
			{
				StructField: "CPU",
				StringValidation: &cr.StringValidation{
					Default:     "200m",
					CastNumeric: true,
				},
				Parser: k8s.QuantityParser(&k8s.QuantityValidation{
					GreaterThan: k8s.QuantityPtr(kresource.MustParse("0")),
				}),
			},
			{
				StructField: "Mem",
				StringPtrValidation: &cr.StringPtrValidation{
					Default: nil,
				},
				Parser: k8s.QuantityParser(&k8s.QuantityValidation{
					GreaterThan: k8s.QuantityPtr(kresource.MustParse("0")),
				}),
			},
			{
				StructField: "GPU",
				Int64Validation: &cr.Int64Validation{
					Default:              0,
					GreaterThanOrEqualTo: pointer.Int64(0),
				},
			},
		},
	},
}

func (ac *APICompute) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", MinReplicasKey, s.Int32(ac.MinReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxReplicasKey, s.Int32(ac.MaxReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", InitReplicasKey, s.Int32(ac.InitReplicas)))
	if ac.MinReplicas != ac.MaxReplicas {
		sb.WriteString(fmt.Sprintf("%s: %s\n", TargetCPUUtilizationKey, s.Int32(ac.TargetCPUUtilization)))
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", CPUKey, ac.CPU.UserString))
	if ac.GPU > 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", GPUKey, s.Int64(ac.GPU)))
	}
	if ac.Mem != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MemKey, ac.Mem.UserString))
	}
	return sb.String()
}

func (ac *APICompute) Validate() error {
	if ac.MinReplicas > ac.MaxReplicas {
		return ErrorMinReplicasGreaterThanMax(ac.MinReplicas, ac.MaxReplicas)
	}

	if ac.InitReplicas > ac.MaxReplicas {
		return ErrorInitReplicasGreaterThanMax(ac.InitReplicas, ac.MaxReplicas)
	}

	if ac.InitReplicas < ac.MinReplicas {
		return ErrorInitReplicasLessThanMin(ac.InitReplicas, ac.MinReplicas)
	}

	return nil
}

func (ac *APICompute) ID() string {
	var buf bytes.Buffer
	buf.WriteString(s.Int32(ac.MinReplicas))
	buf.WriteString(s.Int32(ac.MaxReplicas))
	buf.WriteString(s.Int32(ac.InitReplicas))
	buf.WriteString(s.Int32(ac.TargetCPUUtilization))
	buf.WriteString(ac.CPU.ID())
	buf.WriteString(k8s.QuantityPtrID(ac.Mem))
	buf.WriteString(s.Int64(ac.GPU))
	return hash.Bytes(buf.Bytes())
}

// Only consider CPU, Mem, GPU
func (ac *APICompute) IDWithoutReplicas() string {
	var buf bytes.Buffer
	buf.WriteString(ac.CPU.ID())
	buf.WriteString(k8s.QuantityPtrID(ac.Mem))
	buf.WriteString(s.Int64(ac.GPU))
	return hash.Bytes(buf.Bytes())
}
