/*
Copyright 2020 Cortex Labs, Inc.

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
	PredictorKey      = "predictor"
	MonitoringKey     = "monitoring"
	NetworkingKey     = "networking"
	ComputeKey        = "compute"
	AutoscalingKey    = "autoscaling"
	UpdateStrategyKey = "update_strategy"

	// TrafficSplitter
	APIsKey   = "apis"
	WeightKey = "weight"

	// Predictor
	TypeKey                   = "type"
	PathKey                   = "path"
	ModelPathKey              = "model_path"
	ServerSideBatchingKey     = "server_side_batching"
	ProcessesPerReplicaKey    = "processes_per_replica"
	ThreadsPerProcessKey      = "threads_per_process"
	ModelsKey                 = "models"
	PythonPathKey             = "python_path"
	ImageKey                  = "image"
	TensorFlowServingImageKey = "tensorflow_serving_image"
	ConfigKey                 = "config"
	EnvKey                    = "env"
	SignatureKeyKey           = "signature_key"

	// ServerSideBatching
	MaxBatchSizeKey  = "max_batch_size"
	BatchIntervalKey = "batch_interval"

	// ModelResource
	ModelsNameKey = "name"

	// Monitoring
	KeyKey       = "key"
	ModelTypeKey = "model_type"

	// Networking
	APIGatewayKey = "api_gateway"
	EndpointKey   = "endpoint"
	LocalPortKey  = "local_port"

	// Compute
	CPUKey = "cpu"
	MemKey = "mem"
	GPUKey = "gpu"
	InfKey = "inf"

	// Autoscaling
	MinReplicasKey                  = "min_replicas"
	MaxReplicasKey                  = "max_replicas"
	InitReplicasKey                 = "init_replicas"
	TargetReplicaConcurrencyKey     = "target_replica_concurrency"
	MaxReplicaConcurrencyKey        = "max_replica_concurrency"
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
	APIGatewayAnnotationKey                   = "networking.cortex.dev/api-gateway"
	ProcessesPerReplicaAnnotationKey          = "predictor.cortex.dev/processes-per-replica"
	ThreadsPerProcessAnnotationKey            = "predictor.cortex.dev/threads-per-process"
	MinReplicasAnnotationKey                  = "autoscaling.cortex.dev/min-replicas"
	MaxReplicasAnnotationKey                  = "autoscaling.cortex.dev/max-replicas"
	TargetReplicaConcurrencyAnnotationKey     = "autoscaling.cortex.dev/target-replica-concurrency"
	MaxReplicaConcurrencyAnnotationKey        = "autoscaling.cortex.dev/max-replica-concurrency"
	WindowAnnotationKey                       = "autoscaling.cortex.dev/window"
	DownscaleStabilizationPeriodAnnotationKey = "autoscaling.cortex.dev/downscale-stabilization-period"
	UpscaleStabilizationPeriodAnnotationKey   = "autoscaling.cortex.dev/upscale-stabilization-period"
	MaxDownscaleFactorAnnotationKey           = "autoscaling.cortex.dev/max-downscale-factor"
	MaxUpscaleFactorAnnotationKey             = "autoscaling.cortex.dev/max-upscale-factor"
	DownscaleToleranceAnnotationKey           = "autoscaling.cortex.dev/downscale-tolerance"
	UpscaleToleranceAnnotationKey             = "autoscaling.cortex.dev/upscale-tolerance"
)
