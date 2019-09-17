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

package userconfig

const (
	// Shared
	UnknownKey = "unknown"
	NameKey    = "name"
	KindKey    = "kind"
	TagsKey    = "tags"

	// API
	ModelKey          = "model"
	ModelFormatKey    = "model_format"
	RequestHandlerKey = "request_handler"
	TFSignatureKeyKey = "tf_signature_key"
	TrackerKey        = "tracker"
	ModelTypeKey      = "model_type"
	KeyKey            = "key"

	// compute
	ComputeKey              = "compute"
	MinReplicasKey          = "min_replicas"
	MaxReplicasKey          = "max_replicas"
	InitReplicasKey         = "init_replicas"
	TargetCPUUtilizationKey = "target_cpu_utilization"
	CPUKey                  = "cpu"
	GPUKey                  = "gpu"
	MemKey                  = "mem"
)
