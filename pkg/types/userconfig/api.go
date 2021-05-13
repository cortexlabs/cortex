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

package userconfig

import (
	"fmt"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/yaml"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type API struct {
	Resource

	Pod              *Pod            `json:"pod" yaml:"pod"`
	APIs             []*TrafficSplit `json:"apis" yaml:"apis"`
	Networking       *Networking     `json:"networking" yaml:"networking"`
	Autoscaling      *Autoscaling    `json:"autoscaling" yaml:"autoscaling"`
	UpdateStrategy   *UpdateStrategy `json:"update_strategy" yaml:"update_strategy"`
	Index            int             `json:"index" yaml:"-"`
	FileName         string          `json:"file_name" yaml:"-"`
	SubmittedAPISpec interface{}     `json:"submitted_api_spec" yaml:"submitted_api_spec"`
}

type Pod struct {
	NodeGroups []string      `json:"node_groups" yaml:"node_groups"`
	ShmSize    *k8s.Quantity `json:"shm_size" yaml:"shm_size"`
	Containers []Container   `json:"containers" yaml:"containers"`
}

type Container struct {
	Name  string            `json:"name" yaml:"name"`
	Image string            `json:"image" yaml:"image"`
	Env   map[string]string `json:"env" yaml:"env"`

	Command []string `json:"command" yaml:"command"`
	Args    []string `json:"args" yaml:"args"`

	Compute *Compute `json:"compute" yaml:"compute"`
}

type TrafficSplit struct {
	Name   string `json:"name" yaml:"name"`
	Weight int32  `json:"weight" yaml:"weight"`
	Shadow bool   `json:"shadow" yaml:"shadow"`
}

type Networking struct {
	Endpoint *string `json:"endpoint" yaml:"endpoint"`
}

type Compute struct {
	CPU *k8s.Quantity `json:"cpu" yaml:"cpu"`
	Mem *k8s.Quantity `json:"mem" yaml:"mem"`
	GPU int64         `json:"gpu" yaml:"gpu"`
	Inf int64         `json:"inf" yaml:"inf"`
}

type Autoscaling struct {
	MinReplicas                  int32         `json:"min_replicas" yaml:"min_replicas"`
	MaxReplicas                  int32         `json:"max_replicas" yaml:"max_replicas"`
	InitReplicas                 int32         `json:"init_replicas" yaml:"init_replicas"`
	MaxReplicaQueueLength        int64         `json:"max_replica_queue_length" yaml:"max_replica_queue_length"`
	MaxReplicaConcurrency        int64         `json:"max_replica_concurrency" yaml:"max_replica_concurrency"`
	TargetReplicaConcurrency     float64       `json:"target_replica_concurrency" yaml:"target_replica_concurrency"`
	Window                       time.Duration `json:"window" yaml:"window"`
	DownscaleStabilizationPeriod time.Duration `json:"downscale_stabilization_period" yaml:"downscale_stabilization_period"`
	UpscaleStabilizationPeriod   time.Duration `json:"upscale_stabilization_period" yaml:"upscale_stabilization_period"`
	MaxDownscaleFactor           float64       `json:"max_downscale_factor" yaml:"max_downscale_factor"`
	MaxUpscaleFactor             float64       `json:"max_upscale_factor" yaml:"max_upscale_factor"`
	DownscaleTolerance           float64       `json:"downscale_tolerance" yaml:"downscale_tolerance"`
	UpscaleTolerance             float64       `json:"upscale_tolerance" yaml:"upscale_tolerance"`
}

type UpdateStrategy struct {
	MaxSurge       string `json:"max_surge" yaml:"max_surge"`
	MaxUnavailable string `json:"max_unavailable" yaml:"max_unavailable"`
}

func (api *API) Identify() string {
	return IdentifyAPI(api.FileName, api.Name, api.Kind, api.Index)
}

func IdentifyAPI(filePath string, name string, kind Kind, index int) string {
	str := ""

	if filePath != "" {
		str += filePath + ": "
	}

	if name != "" {
		str += name
		if kind != UnknownKind {
			str += " (" + kind.String() + ")"
		}
		return str
	} else if index >= 0 {
		return str + "resource at " + s.Index(index)
	}
	return str + "resource"
}

// InitReplicas was left out deliberately
func (api *API) ToK8sAnnotations() map[string]string {
	annotations := map[string]string{}
	if api.Networking != nil {
		annotations[EndpointAnnotationKey] = *api.Networking.Endpoint
	}

	if api.Autoscaling != nil {
		annotations[MinReplicasAnnotationKey] = s.Int32(api.Autoscaling.MinReplicas)
		annotations[MaxReplicasAnnotationKey] = s.Int32(api.Autoscaling.MaxReplicas)
		annotations[MaxReplicaQueueLengthAnnotationKey] = s.Int64(api.Autoscaling.MaxReplicaQueueLength)
		annotations[TargetReplicaConcurrencyAnnotationKey] = s.Float64(api.Autoscaling.TargetReplicaConcurrency)
		annotations[MaxReplicaConcurrencyAnnotationKey] = s.Int64(api.Autoscaling.MaxReplicaConcurrency)
		annotations[WindowAnnotationKey] = api.Autoscaling.Window.String()
		annotations[DownscaleStabilizationPeriodAnnotationKey] = api.Autoscaling.DownscaleStabilizationPeriod.String()
		annotations[UpscaleStabilizationPeriodAnnotationKey] = api.Autoscaling.UpscaleStabilizationPeriod.String()
		annotations[MaxDownscaleFactorAnnotationKey] = s.Float64(api.Autoscaling.MaxDownscaleFactor)
		annotations[MaxUpscaleFactorAnnotationKey] = s.Float64(api.Autoscaling.MaxUpscaleFactor)
		annotations[DownscaleToleranceAnnotationKey] = s.Float64(api.Autoscaling.DownscaleTolerance)
		annotations[UpscaleToleranceAnnotationKey] = s.Float64(api.Autoscaling.UpscaleTolerance)
	}
	return annotations
}

func AutoscalingFromAnnotations(k8sObj kmeta.Object) (*Autoscaling, error) {
	a := Autoscaling{}

	minReplicas, err := k8s.ParseInt32Annotation(k8sObj, MinReplicasAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MinReplicas = minReplicas

	maxReplicas, err := k8s.ParseInt32Annotation(k8sObj, MaxReplicasAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxReplicas = maxReplicas

	maxReplicaQueueLength, err := k8s.ParseInt64Annotation(k8sObj, MaxReplicaQueueLengthAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxReplicaQueueLength = maxReplicaQueueLength

	maxReplicaConcurrency, err := k8s.ParseInt64Annotation(k8sObj, MaxReplicaConcurrencyAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxReplicaConcurrency = maxReplicaConcurrency

	targetReplicaConcurrency, err := k8s.ParseFloat64Annotation(k8sObj, TargetReplicaConcurrencyAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.TargetReplicaConcurrency = targetReplicaConcurrency

	window, err := k8s.ParseDurationAnnotation(k8sObj, WindowAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.Window = window

	downscaleStabilizationPeriod, err := k8s.ParseDurationAnnotation(k8sObj, DownscaleStabilizationPeriodAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.DownscaleStabilizationPeriod = downscaleStabilizationPeriod

	upscaleStabilizationPeriod, err := k8s.ParseDurationAnnotation(k8sObj, UpscaleStabilizationPeriodAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.UpscaleStabilizationPeriod = upscaleStabilizationPeriod

	maxDownscaleFactor, err := k8s.ParseFloat64Annotation(k8sObj, MaxDownscaleFactorAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxDownscaleFactor = maxDownscaleFactor

	maxUpscaleFactor, err := k8s.ParseFloat64Annotation(k8sObj, MaxUpscaleFactorAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.MaxUpscaleFactor = maxUpscaleFactor

	downscaleTolerance, err := k8s.ParseFloat64Annotation(k8sObj, DownscaleToleranceAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.DownscaleTolerance = downscaleTolerance

	upscaleTolerance, err := k8s.ParseFloat64Annotation(k8sObj, UpscaleToleranceAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.UpscaleTolerance = upscaleTolerance

	return &a, nil
}

func (api *API) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", NameKey, api.Name))
	sb.WriteString(fmt.Sprintf("%s: %s\n", KindKey, api.Kind.String()))

	if api.Kind == TrafficSplitterKind {
		sb.WriteString(fmt.Sprintf("%s:\n", APIsKey))
		for _, api := range api.APIs {
			sb.WriteString(s.Indent(api.UserStr(), "  "))
		}
	}

	if api.Pod != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", HandlerKey))
		sb.WriteString(s.Indent(api.Pod.UserStr(), "  "))
	}

	if api.Networking != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", NetworkingKey))
		sb.WriteString(s.Indent(api.Networking.UserStr(), "  "))
	}

	if api.Autoscaling != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", AutoscalingKey))
		sb.WriteString(s.Indent(api.Autoscaling.UserStr(), "  "))
	}

	if api.UpdateStrategy != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", UpdateStrategyKey))
		sb.WriteString(s.Indent(api.UpdateStrategy.UserStr(), "  "))
	}

	return sb.String()
}

func (trafficSplit *TrafficSplit) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", NameKey, trafficSplit.Name))
	sb.WriteString(fmt.Sprintf("%s: %s\n", WeightKey, s.Int32(trafficSplit.Weight)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", ShadowKey, s.Bool(trafficSplit.Shadow)))
	return sb.String()
}

func (pod *Pod) UserStr() string {
	var sb strings.Builder

	if pod.ShmSize != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ShmSizeKey, pod.ShmSize.UserString))
	}
	if pod.NodeGroups == nil {
		sb.WriteString(fmt.Sprintf("%s: null\n", NodeGroupsKey))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %s\n", NodeGroupsKey, s.ObjFlatNoQuotes(pod.NodeGroups)))
	}

	sb.WriteString(fmt.Sprintf("%s:\n", ContainersKey))
	for _, container := range pod.Containers {
		containerUserStr := s.Indent(container.UserStr(), "  ")
		containerUserStr = containerUserStr[:2] + "-" + containerUserStr[3:]
		sb.WriteString(containerUserStr)
	}

	return sb.String()
}

func (container *Container) UserStr() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s: %s\n", ContainerNameKey, container.Name))
	sb.WriteString(fmt.Sprintf("%s: %s\n", ImageKey, container.Image))

	if len(container.Env) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", EnvKey))
		d, _ := yaml.Marshal(&container.Env)
		sb.WriteString(s.Indent(string(d), "  "))
	}

	sb.WriteString(fmt.Sprintf("%s: %s\n", CommandKey, s.ObjFlatNoQuotes(container.Command)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", ArgsKey, s.ObjFlatNoQuotes(container.Args)))

	if container.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(container.Compute.UserStr(), "  "))
	}

	return sb.String()
}

func (networking *Networking) UserStr() string {
	var sb strings.Builder
	if networking.Endpoint != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", EndpointKey, *networking.Endpoint))
	}
	return sb.String()
}

func (compute *Compute) UserStr() string {
	var sb strings.Builder
	if compute.CPU == nil {
		sb.WriteString(fmt.Sprintf("%s: null  # no limit\n", CPUKey))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %s\n", CPUKey, compute.CPU.UserString))
	}
	if compute.GPU > 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", GPUKey, s.Int64(compute.GPU)))
	}
	if compute.Inf > 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", InfKey, s.Int64(compute.Inf)))
	}
	if compute.Mem == nil {
		sb.WriteString(fmt.Sprintf("%s: null  # no limit\n", MemKey))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MemKey, compute.Mem.UserString))
	}
	return sb.String()
}

func (compute Compute) Equals(c2 *Compute) bool {
	if c2 == nil {
		return false
	}

	if compute.CPU == nil && c2.CPU != nil || compute.CPU != nil && c2.CPU == nil {
		return false
	}

	if compute.CPU != nil && c2.CPU != nil && !compute.CPU.Equal(*c2.CPU) {
		return false
	}

	if compute.Mem == nil && c2.Mem != nil || compute.Mem != nil && c2.Mem == nil {
		return false
	}

	if compute.Mem != nil && c2.Mem != nil && !compute.Mem.Equal(*c2.Mem) {
		return false
	}

	if compute.GPU != c2.GPU {
		return false
	}

	return true
}

func (autoscaling *Autoscaling) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", MinReplicasKey, s.Int32(autoscaling.MinReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxReplicasKey, s.Int32(autoscaling.MaxReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", InitReplicasKey, s.Int32(autoscaling.InitReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxReplicaQueueLengthKey, s.Int64(autoscaling.MaxReplicaQueueLength)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxReplicaConcurrencyKey, s.Int64(autoscaling.MaxReplicaConcurrency)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", TargetReplicaConcurrencyKey, s.Float64(autoscaling.TargetReplicaConcurrency)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", WindowKey, autoscaling.Window.String()))
	sb.WriteString(fmt.Sprintf("%s: %s\n", DownscaleStabilizationPeriodKey, autoscaling.DownscaleStabilizationPeriod.String()))
	sb.WriteString(fmt.Sprintf("%s: %s\n", UpscaleStabilizationPeriodKey, autoscaling.UpscaleStabilizationPeriod.String()))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxDownscaleFactorKey, s.Float64(autoscaling.MaxDownscaleFactor)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxUpscaleFactorKey, s.Float64(autoscaling.MaxUpscaleFactor)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", DownscaleToleranceKey, s.Float64(autoscaling.DownscaleTolerance)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", UpscaleToleranceKey, s.Float64(autoscaling.UpscaleTolerance)))

	return sb.String()
}

func (updateStrategy *UpdateStrategy) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxSurgeKey, updateStrategy.MaxSurge))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxUnavailableKey, updateStrategy.MaxUnavailable))
	return sb.String()
}

func ZeroCompute() Compute {
	return Compute{
		CPU: &k8s.Quantity{},
		Mem: &k8s.Quantity{},
		GPU: 0,
	}
}

func GetTotalComputeFromContainers(containers []Container) Compute {
	compute := Compute{}

	for _, container := range containers {
		if container.Compute == nil {
			continue
		}

		if container.Compute.CPU != nil {
			newCPUQuantity := k8s.NewQuantity(container.Compute.CPU.Value())
			if compute.CPU != nil {
				compute.CPU = newCPUQuantity
			} else if newCPUQuantity != nil {
				compute.CPU.AddQty(*newCPUQuantity)
			}
		}

		if container.Compute.Mem != nil {
			newMemQuantity := k8s.NewQuantity(container.Compute.Mem.Value())
			if compute.CPU != nil {
				compute.Mem = newMemQuantity
			} else if newMemQuantity != nil {
				compute.Mem.AddQty(*newMemQuantity)
			}
		}

		compute.GPU += container.Compute.GPU
		compute.Inf += container.Compute.Inf
	}

	return compute
}

func (api *API) TelemetryEvent() map[string]interface{} {
	event := map[string]interface{}{"kind": api.Kind}

	if len(api.APIs) > 0 {
		event["apis._is_defined"] = true
		event["apis._len"] = len(api.APIs)
	}

	if api.Networking != nil {
		event["networking._is_defined"] = true
		if api.Networking.Endpoint != nil {
			event["networking.endpoint._is_defined"] = true
			if urls.CanonicalizeEndpoint(api.Name) != *api.Networking.Endpoint {
				event["networking.endpoint._is_custom"] = true
			}
		}
	}

	if api.Pod != nil {
		event["pod._is_defined"] = true
		if api.Pod.ShmSize != nil {
			event["pod.shm_size"] = api.Pod.ShmSize.String()
		}
		event["pod.node_groups._is_defined"] = len(api.Pod.NodeGroups) > 0
		event["pod.node_groups._len"] = len(api.Pod.NodeGroups)
		event["pod.containers._len"] = len(api.Pod.Containers)

		totalCompute := GetTotalComputeFromContainers(api.Pod.Containers)
		event["pod.containers.compute._is_defined"] = true
		if totalCompute.CPU != nil {
			event["pod.containers.compute.cpu._is_defined"] = true
			event["pod.containers.compute.cpu"] = float64(totalCompute.CPU.MilliValue()) / 1000
		}
		if totalCompute.Mem != nil {
			event["pod.containers.compute.mem._is_defined"] = true
			event["pod.containers.compute.mem"] = totalCompute.Mem.Value()
		}
		event["pod.containers.compute.gpu"] = totalCompute.GPU
		event["pod.containers.compute.inf"] = totalCompute.Inf
	}

	if api.UpdateStrategy != nil {
		event["update_strategy._is_defined"] = true
		event["update_strategy.max_surge"] = api.UpdateStrategy.MaxSurge
		event["update_strategy.max_unavailable"] = api.UpdateStrategy.MaxUnavailable
	}

	if api.Autoscaling != nil {
		event["autoscaling._is_defined"] = true
		event["autoscaling.min_replicas"] = api.Autoscaling.MinReplicas
		event["autoscaling.max_replicas"] = api.Autoscaling.MaxReplicas
		event["autoscaling.init_replicas"] = api.Autoscaling.InitReplicas
		event["autoscaling.max_replica_queue_length"] = api.Autoscaling.MaxReplicaQueueLength
		event["autoscaling.max_replica_concurrency"] = api.Autoscaling.MaxReplicaConcurrency
		event["autoscaling.target_replica_concurrency"] = api.Autoscaling.TargetReplicaConcurrency
		event["autoscaling.window"] = api.Autoscaling.Window.Seconds()
		event["autoscaling.downscale_stabilization_period"] = api.Autoscaling.DownscaleStabilizationPeriod.Seconds()
		event["autoscaling.upscale_stabilization_period"] = api.Autoscaling.UpscaleStabilizationPeriod.Seconds()
		event["autoscaling.max_downscale_factor"] = api.Autoscaling.MaxDownscaleFactor
		event["autoscaling.max_upscale_factor"] = api.Autoscaling.MaxUpscaleFactor
		event["autoscaling.downscale_tolerance"] = api.Autoscaling.DownscaleTolerance
		event["autoscaling.upscale_tolerance"] = api.Autoscaling.UpscaleTolerance
	}

	return event
}
