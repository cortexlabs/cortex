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

package userconfig

import (
	"fmt"
	"strings"
	"time"

	"github.com/PEAT-AI/yaml"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type API struct {
	Resource

	Pod              *Pod            `json:"pod" yaml:"pod"`
	NodeGroups       []string        `json:"node_groups" yaml:"node_groups"`
	APIs             []*TrafficSplit `json:"apis" yaml:"apis"`
	Networking       *Networking     `json:"networking" yaml:"networking"`
	Autoscaling      *Autoscaling    `json:"autoscaling" yaml:"autoscaling"`
	UpdateStrategy   *UpdateStrategy `json:"update_strategy" yaml:"update_strategy"`
	Index            int             `json:"index" yaml:"-"`
	FileName         string          `json:"file_name" yaml:"-"`
	SubmittedAPISpec interface{}     `json:"submitted_api_spec" yaml:"submitted_api_spec"`
}

type Pod struct {
	Port           *int32       `json:"port" yaml:"port"`
	MaxQueueLength int64        `json:"max_queue_length" yaml:"max_queue_length"`
	MaxConcurrency int64        `json:"max_concurrency" yaml:"max_concurrency"`
	Containers     []*Container `json:"containers" yaml:"containers"`
}

type Container struct {
	Name  string            `json:"name" yaml:"name"`
	Image string            `json:"image" yaml:"image"`
	Env   map[string]string `json:"env" yaml:"env"`

	Command []string `json:"command" yaml:"command"`
	Args    []string `json:"args" yaml:"args"`

	ReadinessProbe *Probe   `json:"readiness_probe" yaml:"readiness_probe"`
	LivenessProbe  *Probe   `json:"liveness_probe" yaml:"liveness_probe"`
	PreStop        *PreStop `json:"pre_stop" yaml:"pre_stop"`

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

type Probe struct {
	HTTPGet             *HTTPGetHandler   `json:"http_get" yaml:"http_get"`
	TCPSocket           *TCPSocketHandler `json:"tcp_socket" yaml:"tcp_socket"`
	Exec                *ExecHandler      `json:"exec" yaml:"exec"`
	InitialDelaySeconds int32             `json:"initial_delay_seconds" yaml:"initial_delay_seconds"`
	TimeoutSeconds      int32             `json:"timeout_seconds" yaml:"timeout_seconds"`
	PeriodSeconds       int32             `json:"period_seconds" yaml:"period_seconds"`
	SuccessThreshold    int32             `json:"success_threshold" yaml:"success_threshold"`
	FailureThreshold    int32             `json:"failure_threshold" yaml:"failure_threshold"`
}

type PreStop struct {
	HTTPGet *HTTPGetHandler `json:"http_get" yaml:"http_get"`
	Exec    *ExecHandler    `json:"exec" yaml:"exec"`
}

type HTTPGetHandler struct {
	Path string `json:"path" yaml:"path"`
	Port int32  `json:"port" yaml:"port"`
}

type TCPSocketHandler struct {
	Port int32 `json:"port" yaml:"port"`
}

type ExecHandler struct {
	Command []string `json:"command" yaml:"command"`
}

type Compute struct {
	CPU *k8s.Quantity `json:"cpu" yaml:"cpu"`
	Mem *k8s.Quantity `json:"mem" yaml:"mem"`
	GPU int64         `json:"gpu" yaml:"gpu"`
	Inf int64         `json:"inf" yaml:"inf"`
	Shm *k8s.Quantity `json:"shm" yaml:"shm"`
}

type Autoscaling struct {
	MinReplicas                  int32         `json:"min_replicas" yaml:"min_replicas"`
	MaxReplicas                  int32         `json:"max_replicas" yaml:"max_replicas"`
	InitReplicas                 int32         `json:"init_replicas" yaml:"init_replicas"`
	TargetInFlight               *float64      `json:"target_in_flight" yaml:"target_in_flight"`
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

	if len(api.APIs) > 0 {
		annotations[NumTrafficSplitterTargetsAnnotationKey] = s.Int32(int32(len(api.APIs)))
	}

	if api.Pod != nil && api.Kind == RealtimeAPIKind {
		annotations[MaxConcurrencyAnnotationKey] = s.Int64(api.Pod.MaxConcurrency)
		annotations[MaxQueueLengthAnnotationKey] = s.Int64(api.Pod.MaxQueueLength)
	}

	if api.Pod != nil && api.Kind == AsyncAPIKind {
		annotations[MaxConcurrencyAnnotationKey] = s.Int64(api.Pod.MaxConcurrency)
	}

	if api.Networking != nil {
		annotations[EndpointAnnotationKey] = *api.Networking.Endpoint
	}

	if api.Autoscaling != nil {
		annotations[MinReplicasAnnotationKey] = s.Int32(api.Autoscaling.MinReplicas)
		annotations[MaxReplicasAnnotationKey] = s.Int32(api.Autoscaling.MaxReplicas)
		annotations[TargetInFlightAnnotationKey] = s.Float64(*api.Autoscaling.TargetInFlight)
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

	targetInFlight, err := k8s.ParseFloat64Annotation(k8sObj, TargetInFlightAnnotationKey)
	if err != nil {
		return nil, err
	}
	a.TargetInFlight = pointer.Float64(targetInFlight)

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

func TrafficSplitterTargetsFromAnnotations(k8sObj kmeta.Object) (int32, error) {
	targets, err := k8s.ParseInt32Annotation(k8sObj, NumTrafficSplitterTargetsAnnotationKey)
	if err != nil {
		return 0, err
	}
	return targets, nil
}

func EndpointFromAnnotation(k8sObj kmeta.Object) (string, error) {
	endpoint, err := k8s.GetAnnotation(k8sObj, EndpointAnnotationKey)
	if err != nil {
		return "", err
	}
	return endpoint, nil
}

func ConcurrencyFromAnnotations(k8sObj kmeta.Object) (int, int, error) {
	maxQueueLength, err := k8s.ParseIntAnnotation(k8sObj, MaxQueueLengthAnnotationKey)
	if err != nil {
		return 0, 0, err
	}

	maxConcurrency, err := k8s.ParseIntAnnotation(k8sObj, MaxConcurrencyAnnotationKey)
	if err != nil {
		return 0, 0, err
	}

	return maxQueueLength, maxConcurrency, nil
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
		sb.WriteString(fmt.Sprintf("%s:\n", PodKey))
		sb.WriteString(s.Indent(api.Pod.UserStr(api.Kind), "  "))
	}

	if api.Networking != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", NetworkingKey))
		sb.WriteString(s.Indent(api.Networking.UserStr(), "  "))
	}

	if api.Autoscaling != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", AutoscalingKey))
		sb.WriteString(s.Indent(api.Autoscaling.UserStr(), "  "))
	}

	if api.NodeGroups == nil {
		sb.WriteString(fmt.Sprintf("%s: null\n", NodeGroupsKey))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %s\n", NodeGroupsKey, s.ObjFlatNoQuotes(api.NodeGroups)))
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

func (pod *Pod) UserStr(kind Kind) string {
	var sb strings.Builder
	if pod.Port != nil {
		sb.WriteString(fmt.Sprintf("%s: %d\n", PortKey, *pod.Port))
	}

	if kind == RealtimeAPIKind {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MaxConcurrencyKey, s.Int64(pod.MaxConcurrency)))
		sb.WriteString(fmt.Sprintf("%s: %s\n", MaxQueueLengthKey, s.Int64(pod.MaxQueueLength)))
	}

	if kind == AsyncAPIKind {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MaxConcurrencyKey, s.Int64(pod.MaxConcurrency)))
	}

	sb.WriteString(fmt.Sprintf("%s:\n", ContainersKey))
	for _, container := range pod.Containers {
		containerUserStr := s.Indent(container.UserStr(), "    ")
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

	if container.Command != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", CommandKey, s.ObjFlatNoQuotes(container.Command)))
	}

	if container.Args != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ArgsKey, s.ObjFlatNoQuotes(container.Args)))
	}

	if container.ReadinessProbe != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ReadinessProbeKey))
		sb.WriteString(s.Indent(container.ReadinessProbe.UserStr(), "  "))
	}

	if container.LivenessProbe != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", LivenessProbeKey))
		sb.WriteString(s.Indent(container.LivenessProbe.UserStr(), "  "))
	}

	if container.PreStop != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", PreStopKey))
		sb.WriteString(s.Indent(container.PreStop.UserStr(), "  "))
	}

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

func (probe *Probe) UserStr() string {
	var sb strings.Builder

	if probe.HTTPGet != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", HTTPGetKey))
		sb.WriteString(s.Indent(probe.HTTPGet.UserStr(), "  "))
	}
	if probe.TCPSocket != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", TCPSocketKey))
		sb.WriteString(s.Indent(probe.TCPSocket.UserStr(), "  "))
	}
	if probe.Exec != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ExecKey))
		sb.WriteString(s.Indent(probe.Exec.UserStr(), "  "))
	}

	sb.WriteString(fmt.Sprintf("%s: %d\n", InitialDelaySecondsKey, probe.InitialDelaySeconds))
	sb.WriteString(fmt.Sprintf("%s: %d\n", TimeoutSecondsKey, probe.TimeoutSeconds))
	sb.WriteString(fmt.Sprintf("%s: %d\n", PeriodSecondsKey, probe.PeriodSeconds))
	sb.WriteString(fmt.Sprintf("%s: %d\n", SuccessThresholdKey, probe.SuccessThreshold))
	sb.WriteString(fmt.Sprintf("%s: %d\n", FailureThresholdKey, probe.FailureThreshold))

	return sb.String()
}

func (preStop *PreStop) UserStr() string {
	var sb strings.Builder

	if preStop.HTTPGet != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", HTTPGetKey))
		sb.WriteString(s.Indent(preStop.HTTPGet.UserStr(), "  "))
	}
	if preStop.Exec != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ExecKey))
		sb.WriteString(s.Indent(preStop.Exec.UserStr(), "  "))
	}

	return sb.String()
}

func (httpHandler *HTTPGetHandler) UserStr() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s: %s\n", PathKey, httpHandler.Path))
	sb.WriteString(fmt.Sprintf("%s: %d\n", PortKey, httpHandler.Port))

	return sb.String()
}

func (tcpSocketHandler *TCPSocketHandler) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %d\n", PortKey, tcpSocketHandler.Port))
	return sb.String()
}

func (execHandler *ExecHandler) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", CommandKey, s.ObjFlatNoQuotes(execHandler.Command)))
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
	if compute.Shm == nil {
		sb.WriteString(fmt.Sprintf("%s: null  # not configured\n", ShmKey))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ShmKey, compute.Shm.UserString))
	}
	return sb.String()
}

func (autoscaling *Autoscaling) UserStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", MinReplicasKey, s.Int32(autoscaling.MinReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", MaxReplicasKey, s.Int32(autoscaling.MaxReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", InitReplicasKey, s.Int32(autoscaling.InitReplicas)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", TargetInFlightKey, s.Float64(*autoscaling.TargetInFlight)))
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

func GetPodComputeRequest(api *API) Compute {
	var cpuQtys []kresource.Quantity
	var memQtys []kresource.Quantity
	var shmQtys []kresource.Quantity
	var totalGPU int64
	var totalInf int64

	for _, container := range api.Pod.Containers {
		if container == nil || container.Compute == nil {
			continue
		}
		if container.Compute.CPU != nil {
			cpuQtys = append(cpuQtys, container.Compute.CPU.Quantity)
		}
		if container.Compute.Mem != nil {
			memQtys = append(memQtys, container.Compute.Mem.Quantity)
		}
		if container.Compute.Shm != nil {
			shmQtys = append(shmQtys, container.Compute.Shm.Quantity)
		}
		totalGPU += container.Compute.GPU
		totalInf += container.Compute.Inf
	}

	if api.Kind == RealtimeAPIKind {
		cpuQtys = append(cpuQtys, consts.CortexProxyCPU)
		memQtys = append(memQtys, consts.CortexProxyMem)
	} else if api.Kind == AsyncAPIKind || api.Kind == BatchAPIKind {
		cpuQtys = append(cpuQtys, consts.CortexDequeuerCPU)
		memQtys = append(memQtys, consts.CortexDequeuerMem)
	}

	return Compute{
		CPU: k8s.NewSummed(cpuQtys...),
		Mem: k8s.NewSummed(memQtys...),
		Shm: k8s.NewSummed(shmQtys...),
		GPU: totalGPU,
		Inf: totalInf,
	}
}

func GetContainerNames(containers []*Container) strset.Set {
	containerNames := strset.New()
	for _, container := range containers {
		if container != nil {
			containerNames.Add(container.Name)
		}
	}
	return containerNames
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
		if api.Pod.Port != nil {
			event["pod.port"] = *api.Pod.Port
		}

		event["pod.max_concurrency"] = api.Pod.MaxConcurrency
		event["pod.max_queue_length"] = api.Pod.MaxQueueLength

		event["pod.containers._len"] = len(api.Pod.Containers)

		var numReadinessProbes int
		var numLivenessProbes int
		var numPreStops int
		for _, container := range api.Pod.Containers {
			if container.ReadinessProbe != nil {
				numReadinessProbes++
			}
			if container.LivenessProbe != nil {
				numLivenessProbes++
			}
			if container.PreStop != nil {
				numPreStops++
			}
		}

		event["pod.containers._num_readiness_probes"] = numReadinessProbes
		event["pod.containers._num_liveness_probes"] = numLivenessProbes
		event["pod.containers._num_pre_stops"] = numPreStops

		totalCompute := GetPodComputeRequest(api)
		if totalCompute.CPU != nil {
			event["pod.containers.compute.cpu._is_defined"] = true
			event["pod.containers.compute.cpu"] = float64(totalCompute.CPU.MilliValue()) / 1000
		}
		if totalCompute.Mem != nil {
			event["pod.containers.compute.mem._is_defined"] = true
			event["pod.containers.compute.mem"] = totalCompute.Mem.Value()
		}
		if totalCompute.Shm != nil {
			event["pod.containers.compute.shm._is_defined"] = true
			event["pod.containers.compute.shm"] = totalCompute.Shm.Value()
		}
		event["pod.containers.compute.gpu"] = totalCompute.GPU
		event["pod.containers.compute.inf"] = totalCompute.Inf
	}

	event["node_groups._len"] = len(api.NodeGroups)

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
		if api.Autoscaling.TargetInFlight != nil {
			event["autoscaling.target_in_flight._is_defined"] = true
			event["autoscaling.target_in_flight"] = *api.Autoscaling.TargetInFlight
		}
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
