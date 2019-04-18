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

package config

import "github.com/cortexlabs/cortex/pkg/lib/env"

var (
	Region              string
	Namespace           string
	OperatorImage       string
	SparkImage          string
	TFTrainImage        string
	TFServeImage        string
	TFAPIImage          string
	PythonPackagerImage string
	TFTrainImageGPU     string
	TFServeImageGPU     string
	EnableTelemetry     bool
)

func init() {
	Region = env.GetStr("REGION")
	Namespace = env.GetStr("NAMESPACE")
	OperatorImage = env.GetStr("IMAGE_OPERATOR")
	SparkImage = env.GetStr("IMAGE_SPARK")
	TFTrainImage = env.GetStr("IMAGE_TF_TRAIN")
	TFServeImage = env.GetStr("IMAGE_TF_SERVE")
	TFAPIImage = env.GetStr("IMAGE_TF_API")
	PythonPackagerImage = env.GetStr("IMAGE_PYTHON_PACKAGER")
	TFTrainImageGPU = env.GetStr("IMAGE_TF_TRAIN_GPU")
	TFServeImageGPU = env.GetStr("IMAGE_TF_SERVE_GPU")
	EnableTelemetry = env.GetBool("ENABLE_TELEMETRY")
}
