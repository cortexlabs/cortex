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

const (
	// API
	NameKey           = "name"
	KindKey           = "kind"
	NetworkingKey     = "networking"
	ComputeKey        = "compute"
	AutoscalingKey    = "autoscaling"
	UpdateStrategyKey = "update_strategy"

	// TrafficSplitter
	APIsKey   = "apis"
	WeightKey = "weight"
	ShadowKey = "shadow"

	// Pod
	PodKey            = "pod"
	NodeGroupsKey     = "node_groups"
	PortKey           = "port"
	MaxConcurrencyKey = "max_concurrency"
	MaxQueueLengthKey = "max_queue_length"
	ContainersKey     = "containers"

	// Containers
	ContainerNameKey  = "name"
	ImageKey          = "image"
	EnvKey            = "env"
	CommandKey        = "command"
	ArgsKey           = "args"
	ReadinessProbeKey = "readiness_probe"
	LivenessProbeKey  = "liveness_probe"
	PreStopKey        = "pre_stop"

	// Probe
	HTTPGetKey             = "http_get"
	TCPSocketKey           = "tcp_socket"
	ExecKey                = "exec"
	InitialDelaySecondsKey = "initial_delay_seconds"
	TimeoutSecondsKey      = "timeout_seconds"
	PeriodSecondsKey       = "period_seconds"
	SuccessThresholdKey    = "success_threshold"
	FailureThresholdKey    = "failure_threshold"

	// Probe types
	PathKey = "path"

	// Compute
	CPUKey = "cpu"
	MemKey = "mem"
	GPUKey = "gpu"
	InfKey = "inf"
	ShmKey = "shm"

	// Networking
	EndpointKey = "endpoint"

	// Autoscaling
	MinReplicasKey                  = "min_replicas"
	MaxReplicasKey                  = "max_replicas"
	InitReplicasKey                 = "init_replicas"
	TargetInFlightKey               = "target_in_flight"
	WindowKey                       = "window"
	DownscaleStabilizationPeriodKey = "downscale_stabilization_period"
	UpscaleStabilizationPeriodKey   = "upscale_stabilization_period"
	MaxDownscaleFactorKey           = "max_downscale_factor"
	MaxUpscaleFactorKey             = "max_upscale_factor"
	DownscaleToleranceKey           = "downscale_tolerance"
	UpscaleToleranceKey             = "upscale_tolerance"

	// UpdateStrategy
	MaxSurgeKey       = "max_surge"
	MaxUnavailableKey = "max_unavailable"

	// K8s annotation
	EndpointAnnotationKey                     = "networking.cortex.dev/endpoint"
	MaxConcurrencyAnnotationKey               = "pod.cortex.dev/max-concurrency"
	MaxQueueLengthAnnotationKey               = "pod.cortex.dev/max-queue-length"
	NumTrafficSplitterTargetsAnnotationKey    = "apis.cortex.dev/traffic-splitter-targets"
	MinReplicasAnnotationKey                  = "autoscaling.cortex.dev/min-replicas"
	MaxReplicasAnnotationKey                  = "autoscaling.cortex.dev/max-replicas"
	TargetInFlightAnnotationKey               = "autoscaling.cortex.dev/target-in-flight"
	WindowAnnotationKey                       = "autoscaling.cortex.dev/window"
	DownscaleStabilizationPeriodAnnotationKey = "autoscaling.cortex.dev/downscale-stabilization-period"
	UpscaleStabilizationPeriodAnnotationKey   = "autoscaling.cortex.dev/upscale-stabilization-period"
	MaxDownscaleFactorAnnotationKey           = "autoscaling.cortex.dev/max-downscale-factor"
	MaxUpscaleFactorAnnotationKey             = "autoscaling.cortex.dev/max-upscale-factor"
	DownscaleToleranceAnnotationKey           = "autoscaling.cortex.dev/downscale-tolerance"
	UpscaleToleranceAnnotationKey             = "autoscaling.cortex.dev/upscale-tolerance"
)
