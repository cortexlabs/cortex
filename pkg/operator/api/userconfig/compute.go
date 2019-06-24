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
				Min: k8sresource.MustParse("1"),
			}),
		},
		{
			StructField: "ExecutorCPU",
			StringValidation: &cr.StringValidation{
				Default: "1",
			},
			Parser: QuantityParser(&QuantityValidation{
				Min: k8sresource.MustParse("1"),
				Int: true,
			}),
		},
		{
			StructField: "DriverMem",
			StringValidation: &cr.StringValidation{
				Default: "500Mi",
			},
			Parser: QuantityParser(&QuantityValidation{
				Min: k8sresource.MustParse("500Mi"),
			}),
		},
		{
			StructField: "ExecutorMem",
			StringValidation: &cr.StringValidation{
				Default: "500Mi",
			},
			Parser: QuantityParser(&QuantityValidation{
				Min: k8sresource.MustParse("500Mi"),
			}),
		},
		{
			StructField: "DriverMemOverhead",
			StringPtrValidation: &cr.StringPtrValidation{
				Default: nil, // min(DriverMem * 0.4, 384Mi)
			},
			Parser: QuantityParser(&QuantityValidation{
				Min: k8sresource.MustParse("0"),
			}),
		},
		{
			StructField: "ExecutorMemOverhead",
			StringPtrValidation: &cr.StringPtrValidation{
				Default: nil, // min(ExecutorMem * 0.4, 384Mi)
			},
			Parser: QuantityParser(&QuantityValidation{
				Min: k8sresource.MustParse("0"),
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

func (sparkCompute *SparkCompute) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ExecutorsKey, s.Int32(sparkCompute.Executors)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", DriverCPUKey, sparkCompute.DriverCPU.UserString))
	sb.WriteString(fmt.Sprintf("%s: %s\n", DriverMemKey, sparkCompute.DriverMem.UserString))
	if sparkCompute.DriverMemOverhead != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", DriverMemOverheadKey, sparkCompute.DriverMemOverhead.UserString))
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", ExecutorCPUKey, sparkCompute.ExecutorCPU.UserString))
	sb.WriteString(fmt.Sprintf("%s: %s\n", ExecutorMemKey, sparkCompute.ExecutorMem.UserString))
	if sparkCompute.ExecutorMemOverhead != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ExecutorMemOverheadKey, sparkCompute.ExecutorMemOverhead.UserString))
	}
	if sparkCompute.MemOverheadFactor != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ExecutorMemOverheadKey, s.Float64(*sparkCompute.MemOverheadFactor)))
	}
	return sb.String()
}

func (sparkCompute *SparkCompute) ID() string {
	var buf bytes.Buffer
	buf.WriteString(s.Int32(sparkCompute.Executors))
	buf.WriteString(sparkCompute.DriverCPU.ID())
	buf.WriteString(sparkCompute.DriverMem.ID())
	buf.WriteString(QuantityPtrID(sparkCompute.DriverMemOverhead))
	buf.WriteString(sparkCompute.ExecutorCPU.ID())
	buf.WriteString(sparkCompute.ExecutorMem.ID())
	buf.WriteString(QuantityPtrID(sparkCompute.ExecutorMemOverhead))
	if sparkCompute.MemOverheadFactor == nil {
		buf.WriteString("nil")
	} else {
		buf.WriteString(s.Float64(*sparkCompute.MemOverheadFactor))
	}
	return hash.Bytes(buf.Bytes())
}

type TFCompute struct {
	CPU *Quantity `json:"cpu" yaml:"cpu"`
	Mem *Quantity `json:"mem" yaml:"mem"`
	GPU *int64    `json:"gpu" yaml:"gpu"`
}

var tfComputeFieldValidation = &cr.StructFieldValidation{
	StructField: "Compute",
	StructValidation: &cr.StructValidation{
		StructFieldValidations: []*cr.StructFieldValidation{
			{
				StructField: "CPU",
				StringPtrValidation: &cr.StringPtrValidation{
					Default: nil,
				},
				Parser: QuantityParser(&QuantityValidation{
					Min: k8sresource.MustParse("0"),
				}),
			},
			{
				StructField: "Mem",
				StringPtrValidation: &cr.StringPtrValidation{
					Default: nil,
				},
				Parser: QuantityParser(&QuantityValidation{
					Min: k8sresource.MustParse("0"),
				}),
			},
			{
				StructField: "GPU",
				Int64PtrValidation: &cr.Int64PtrValidation{
					Default:     nil,
					GreaterThan: pointer.Int64(0),
				},
			},
		},
	},
}

func (tfCompute *TFCompute) UserConfigStr() string {
	var sb strings.Builder
	if tfCompute.CPU != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", CPUKey, tfCompute.CPU.UserString))
	}
	if tfCompute.GPU != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", GPUKey, s.Int64(*tfCompute.GPU)))
	}
	if tfCompute.Mem != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MemKey, tfCompute.Mem.UserString))
	}
	return sb.String()
}

func (tfCompute *TFCompute) ID() string {
	var buf bytes.Buffer
	buf.WriteString(QuantityPtrID(tfCompute.CPU))
	buf.WriteString(QuantityPtrID(tfCompute.Mem))
	return hash.Bytes(buf.Bytes())
}

type APICompute struct {
	Replicas int32     `json:"replicas" yaml:"replicas"`
	CPU      *Quantity `json:"cpu" yaml:"cpu"`
	Mem      *Quantity `json:"mem" yaml:"mem"`
	GPU      int64     `json:"gpu" yaml:"gpu"`
}

var apiComputeFieldValidation = &cr.StructFieldValidation{
	StructField: "Compute",
	StructValidation: &cr.StructValidation{
		StructFieldValidations: []*cr.StructFieldValidation{
			{
				StructField: "Replicas",
				Int32Validation: &cr.Int32Validation{
					Default:     1,
					GreaterThan: pointer.Int32(0),
				},
			},
			{
				StructField: "CPU",
				StringPtrValidation: &cr.StringPtrValidation{
					Default: nil,
				},
				Parser: QuantityParser(&QuantityValidation{
					Min: k8sresource.MustParse("0"),
				}),
			},
			{
				StructField: "Mem",
				StringPtrValidation: &cr.StringPtrValidation{
					Default: nil,
				},
				Parser: QuantityParser(&QuantityValidation{
					Min: k8sresource.MustParse("0"),
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

func (apiCompute *APICompute) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ReplicasKey, s.Int32(apiCompute.Replicas)))
	if apiCompute.CPU != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", CPUKey, apiCompute.CPU.UserString))
	}
	if apiCompute.GPU != 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", GPUKey, s.Int64(apiCompute.GPU)))
	}
	if apiCompute.Mem != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MemKey, apiCompute.Mem.UserString))
	}
	return sb.String()
}

func (apiCompute *APICompute) ID() string {
	var buf bytes.Buffer
	buf.WriteString(s.Int32(apiCompute.Replicas))
	buf.WriteString(QuantityPtrID(apiCompute.CPU))
	buf.WriteString(QuantityPtrID(apiCompute.Mem))
	buf.WriteString(s.Int64(apiCompute.GPU))
	return hash.Bytes(buf.Bytes())
}

func (apiCompute *APICompute) IDWithoutReplicas() string {
	var buf bytes.Buffer
	buf.WriteString(QuantityPtrID(apiCompute.CPU))
	buf.WriteString(QuantityPtrID(apiCompute.Mem))
	buf.WriteString(s.Int64(apiCompute.GPU))
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

	for _, tfCompute := range tfComputes {
		if tfCompute.CPU != nil {
			if aggregated.CPU == nil || tfCompute.CPU.Cmp(aggregated.CPU.Quantity) > 0 {
				aggregated.CPU = tfCompute.CPU
			}
		}
		if tfCompute.Mem != nil {
			if aggregated.Mem == nil || tfCompute.Mem.Cmp(aggregated.Mem.Quantity) > 0 {
				aggregated.Mem = tfCompute.Mem
			}
		}
		if tfCompute.GPU != nil {
			if aggregated.GPU == nil || *tfCompute.GPU > *aggregated.GPU {
				aggregated.GPU = tfCompute.GPU
			}
		}
	}

	return &aggregated
}

func (apiCompute *APICompute) Equal(apiCompute2 APICompute) bool {
	if apiCompute.Replicas != apiCompute2.Replicas {
		return false
	}
	if !QuantityPtrsEqual(apiCompute.CPU, apiCompute2.CPU) {
		return false
	}
	if !QuantityPtrsEqual(apiCompute.Mem, apiCompute2.Mem) {
		return false
	}

	if apiCompute.GPU != apiCompute2.GPU {
		return false
	}

	return true
}
