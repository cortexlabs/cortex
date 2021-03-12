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

const (
	// API
	NameKey           = "name"
	KindKey           = "kind"
	PredictorKey      = "predictor"
	TaskDefinitionKey = "definition"
	NetworkingKey     = "networking"
	ComputeKey        = "compute"
	AutoscalingKey    = "autoscaling"
	UpdateStrategyKey = "update_strategy"

	// TrafficSplitter
	APIsKey   = "apis"
	WeightKey = "weight"
	ShadowKey = "shadow"

	// Predictor
	TypeKey                   = "type"
	PathKey                   = "path"
	ServerSideBatchingKey     = "server_side_batching"
	PythonPathKey             = "python_path"
	ImageKey                  = "image"
	TensorFlowServingImageKey = "tensorflow_serving_image"
	ProcessesPerReplicaKey    = "processes_per_replica"
	ThreadsPerProcessKey      = "threads_per_process"
	ShmSize                   = "shm_size"
	LogLevelKey               = "log_level"
	ConfigKey                 = "config"
	EnvKey                    = "env"

	// Predictor/TaskDefinition.Dependencies
	DependenciesKey = "dependencies"
	PipKey          = "pip"
	ShellKey        = "shell"
	CondaKey        = "conda"

	// MultiModelReloading
	MultiModelReloadingKey = "multi_model_reloading"

	// MultiModels
	ModelsKey              = "models"
	ModelsPathKey          = "path"
	ModelsPathsKey         = "paths"
	ModelsDirKey           = "dir"
	ModelsSignatureKeyKey  = "signature_key"
	ModelsCacheSizeKey     = "cache_size"
	ModelsDiskCacheSizeKey = "disk_cache_size"

	// ServerSideBatching
	MaxBatchSizeKey  = "max_batch_size"
	BatchIntervalKey = "batch_interval"

	// ModelResource
	ModelsNameKey = "name"

	// Networking
	EndpointKey = "endpoint"

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
