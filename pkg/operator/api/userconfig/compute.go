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

	k8sresource "k8s.io/apimachinery/pkg/api/resource"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type SparkCompute struct {
	Executors           int32     `json:"executors" yaml:"executors"`
	DriverCPU           Quantity  `json:"driver_cpu" yaml:"driver_cpu"`
	DriverMem           Quantity  `json:"driver_mem" yaml:"driver_mem"`
	DriverMemOverhead   *Quantity `json:"driver_mem_overhead" yaml:"driver_mem_overhead"`
	ExecutorCPU         Quantity  `json:"executor_cpu" yaml:"executor_cpu"`
	ExecutorMem         Quantity  `json:"executor_mem" yaml:"executor_mem"`
	ExecutorMemOverhead *Quantity `json:"executor_mem_overhead" yaml:"executor_mem_overhead"`
	MemOverheadFactor   *float64  `json:"mem_overhead_factor" yaml:"mem_overhead_factor"`
}

var sparkComputeStructValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Executors",
			Int32Validation: &cr.Int32Validation{
				Default:     1,
				GreaterThan: pointer.Int32(0),
			},
		},
		{
			StructField: "DriverCPU",
			StringValidation: &cr.StringValidation{
				Default: "1",
			},
			Parser: QuantityParser(&QuantityValidation{
				GreaterThanOrEqualTo: k8sQuantityPtr(k8sresource.MustParse("1")),
			}),
		},
		{
			StructField: "ExecutorCPU",
			StringValidation: &cr.StringValidation{
				Default: "1",
			},
			Parser: QuantityParser(&QuantityValidation{
				GreaterThanOrEqualTo: k8sQuantityPtr(k8sresource.MustParse("1")),
				Int:                  true,
			}),
		},
		{
			StructField: "DriverMem",
			StringValidation: &cr.StringValidation{
				Default: "500Mi",
			},
			Parser: QuantityParser(&QuantityValidation{
				GreaterThanOrEqualTo: k8sQuantityPtr(k8sresource.MustParse("500Mi")),
			}),
		},
		{
			StructField: "ExecutorMem",
			StringValidation: &cr.StringValidation{
				Default: "500Mi",
			},
			Parser: QuantityParser(&QuantityValidation{
				GreaterThanOrEqualTo: k8sQuantityPtr(k8sresource.MustParse("500Mi")),
			}),
		},
		{
			StructField: "DriverMemOverhead",
			StringPtrValidation: &cr.StringPtrValidation{
				Default: nil, // min(DriverMem * 0.4, 384Mi)
			},
			Parser: QuantityParser(&QuantityValidation{
				GreaterThanOrEqualTo: k8sQuantityPtr(k8sresource.MustParse("0")),
			}),
		},
		{
			StructField: "ExecutorMemOverhead",
			StringPtrValidation: &cr.StringPtrValidation{
				Default: nil, // min(ExecutorMem * 0.4, 384Mi)
			},
			Parser: QuantityParser(&QuantityValidation{
				GreaterThanOrEqualTo: k8sQuantityPtr(k8sresource.MustParse("0")),
			}),
		},
		{
			StructField: "MemOverheadFactor",
			Float64PtrValidation: &cr.Float64PtrValidation{
				Default:              nil, // set to 0.4 by Spark
				GreaterThanOrEqualTo: pointer.Float64(0),
				LessThan:             pointer.Float64(1),
			},
		},
	},
}

func sparkComputeFieldValidation(fieldName string) *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField:      fieldName,
		StructValidation: sparkComputeStructValidation,
	}
}

func (sc *SparkCompute) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ExecutorsKey, s.Int32(sc.Executors)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", DriverCPUKey, sc.DriverCPU.UserString))
	sb.WriteString(fmt.Sprintf("%s: %s\n", DriverMemKey, sc.DriverMem.UserString))
	if sc.DriverMemOverhead != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", DriverMemOverheadKey, sc.DriverMemOverhead.UserString))
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", ExecutorCPUKey, sc.ExecutorCPU.UserString))
	sb.WriteString(fmt.Sprintf("%s: %s\n", ExecutorMemKey, sc.ExecutorMem.UserString))
	if sc.ExecutorMemOverhead != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ExecutorMemOverheadKey, sc.ExecutorMemOverhead.UserString))
	}
	if sc.MemOverheadFactor != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ExecutorMemOverheadKey, s.Float64(*sc.MemOverheadFactor)))
	}
	return sb.String()
}

func (sc *SparkCompute) ID() string {
	var buf bytes.Buffer
	buf.WriteString(s.Int32(sc.Executors))
	buf.WriteString(sc.DriverCPU.ID())
	buf.WriteString(sc.DriverMem.ID())
	buf.WriteString(QuantityPtrID(sc.DriverMemOverhead))
	buf.WriteString(sc.ExecutorCPU.ID())
	buf.WriteString(sc.ExecutorMem.ID())
	buf.WriteString(QuantityPtrID(sc.ExecutorMemOverhead))
	if sc.MemOverheadFactor == nil {
		buf.WriteString("nil")
	} else {
		buf.WriteString(s.Float64(*sc.MemOverheadFactor))
	}
	return hash.Bytes(buf.Bytes())
}

type TFCompute struct {
	CPU Quantity  `json:"cpu" yaml:"cpu"`
	Mem *Quantity `json:"mem" yaml:"mem"`
	GPU int64     `json:"gpu" yaml:"gpu"`
}

var tfComputeFieldValidation = &cr.StructFieldValidation{
	StructField: "Compute",
	StructValidation: &cr.StructValidation{
		StructFieldValidations: []*cr.StructFieldValidation{
			{
				StructField: "CPU",
				StringValidation: &cr.StringValidation{
					Default: "400m",
				},
				Parser: QuantityParser(&QuantityValidation{
					GreaterThan: k8sQuantityPtr(k8sresource.MustParse("0")),
				}),
			},
			{
				StructField: "Mem",
				StringPtrValidation: &cr.StringPtrValidation{
					Default: nil,
				},
				Parser: QuantityParser(&QuantityValidation{
					GreaterThan: k8sQuantityPtr(k8sresource.MustParse("0")),
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

func (tc *TFCompute) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", CPUKey, tc.CPU.UserString))
	if tc.GPU > 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", GPUKey, s.Int64(tc.GPU)))
	}
	if tc.Mem != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MemKey, tc.Mem.UserString))
	}
	return sb.String()
}

func (tc *TFCompute) ID() string {
	var buf bytes.Buffer
	buf.WriteString(tc.CPU.ID())
	buf.WriteString(QuantityPtrID(tc.Mem))
	return hash.Bytes(buf.Bytes())
}

type APICompute struct {
	MinReplicas          int32     `json:"min_replicas" yaml:"min_replicas"`
	MaxReplicas          int32     `json:"max_replicas" yaml:"max_replicas"`
	InitReplicas         int32     `json:"init_replicas" yaml:"init_replicas"`
	TargetCPUUtilization int32     `json:"target_cpu_utilization" yaml:"target_cpu_utilization"`
	CPU                  Quantity  `json:"cpu" yaml:"cpu"`
	Mem                  *Quantity `json:"mem" yaml:"mem"`
	GPU                  int64     `json:"gpu" yaml:"gpu"`
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
					Default: "400m",
				},
				Parser: QuantityParser(&QuantityValidation{
					GreaterThan: k8sQuantityPtr(k8sresource.MustParse("0")),
				}),
			},
			{
				StructField: "Mem",
				StringPtrValidation: &cr.StringPtrValidation{
					Default: nil,
				},
				Parser: QuantityParser(&QuantityValidation{
					GreaterThan: k8sQuantityPtr(k8sresource.MustParse("0")),
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
	buf.WriteString(QuantityPtrID(ac.Mem))
	buf.WriteString(s.Int64(ac.GPU))
	return hash.Bytes(buf.Bytes())
}

// Only consider CPU, Mem, GPU
func (ac *APICompute) IDWithoutReplicas() string {
	var buf bytes.Buffer
	buf.WriteString(ac.CPU.ID())
	buf.WriteString(QuantityPtrID(ac.Mem))
	buf.WriteString(s.Int64(ac.GPU))
	return hash.Bytes(buf.Bytes())
}

func MaxSparkCompute(sparkComputes ...*SparkCompute) *SparkCompute {
	aggregated := SparkCompute{}

	for _, sparkCompute := range sparkComputes {
		if sparkCompute.Executors > aggregated.Executors {
			aggregated.Executors = sparkCompute.Executors
		}
		if sparkCompute.DriverCPU.Cmp(aggregated.DriverCPU.Quantity) > 0 {
			aggregated.DriverCPU = sparkCompute.DriverCPU
		}
		if sparkCompute.ExecutorCPU.Cmp(aggregated.ExecutorCPU.Quantity) > 0 {
			aggregated.ExecutorCPU = sparkCompute.ExecutorCPU
		}
		if sparkCompute.DriverMem.Cmp(aggregated.DriverMem.Quantity) > 0 {
			aggregated.DriverMem = sparkCompute.DriverMem
		}
		if sparkCompute.ExecutorMem.Cmp(aggregated.ExecutorMem.Quantity) > 0 {
			aggregated.ExecutorMem = sparkCompute.ExecutorMem
		}
		if sparkCompute.DriverMemOverhead != nil {
			if aggregated.DriverMemOverhead == nil || sparkCompute.DriverMemOverhead.Cmp(aggregated.DriverMemOverhead.Quantity) > 0 {
				aggregated.DriverMemOverhead = sparkCompute.DriverMemOverhead
			}
		}
		if sparkCompute.ExecutorMemOverhead != nil {
			if aggregated.ExecutorMemOverhead == nil || sparkCompute.ExecutorMemOverhead.Cmp(aggregated.ExecutorMemOverhead.Quantity) > 0 {
				aggregated.ExecutorMemOverhead = sparkCompute.ExecutorMemOverhead
			}
		}
		if sparkCompute.MemOverheadFactor != nil {
			if aggregated.MemOverheadFactor == nil || *sparkCompute.MemOverheadFactor > *aggregated.MemOverheadFactor {
				aggregated.MemOverheadFactor = sparkCompute.MemOverheadFactor
			}
		}
	}

	return &aggregated
}

func MaxTFCompute(tfComputes ...*TFCompute) *TFCompute {
	aggregated := TFCompute{}

	for _, tc := range tfComputes {
		if tc.CPU.Cmp(aggregated.CPU.Quantity) > 0 {
			aggregated.CPU = tc.CPU
		}
		if tc.Mem != nil {
			if aggregated.Mem == nil || tc.Mem.Cmp(aggregated.Mem.Quantity) > 0 {
				aggregated.Mem = tc.Mem
			}
		}
		if tc.GPU > aggregated.GPU {
			aggregated.GPU = tc.GPU
		}
	}

	return &aggregated
}
