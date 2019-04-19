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

import (
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/env"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
)

var (
	Cortex *CortexConfig
)

type CortexConfig struct {
	ID                  string `json:"id"`
	APIVersion          string `json:"api_version"`
	Bucket              string `json:"bucket"`
	LogGroup            string `json:"log_group"`
	Region              string `json:"region"`
	Namespace           string `json:"namespace"`
	OperatorImage       string `json:"operator_image"`
	SparkImage          string `json:"SparkImage"`
	TFTrainImage        string `json:"tf_train_image"`
	TFServeImage        string `json:"tf_serve_image"`
	TFAPIImage          string `json:"tf_api_image"`
	PythonPackagerImage string `json:"python_packager_image"`
	TFTrainImageGPU     string `json:"tf_train_image_gpu"`
	TFServeImageGPU     string `json:"tf_serve_image_gpu"`
	TelemetryURL        string `json:"telemetry_url"`
	EnableTelemetry     bool   `json:"enable_telemetry"`
	OperatorInCluster   bool   `json:"operator_in_cluster"`
}

func NewFromEnv() *CortexConfig {
	Cortex = &CortexConfig{
		APIVersion:          consts.CortexVersion,
		Bucket:              env.GetStr("BUCKET"),
		LogGroup:            env.GetStr("LOG_GROUP"),
		Region:              env.GetStr("REGION"),
		Namespace:           env.GetStr("NAMESPACE"),
		OperatorImage:       env.GetStr("IMAGE_OPERATOR"),
		SparkImage:          env.GetStr("IMAGE_SPARK"),
		TFTrainImage:        env.GetStr("IMAGE_TF_TRAIN"),
		TFServeImage:        env.GetStr("IMAGE_TF_SERVE"),
		TFAPIImage:          env.GetStr("IMAGE_TF_API"),
		PythonPackagerImage: env.GetStr("IMAGE_PYTHON_PACKAGER"),
		TFTrainImageGPU:     env.GetStr("IMAGE_TF_TRAIN_GPU"),
		TFServeImageGPU:     env.GetStr("IMAGE_TF_SERVE_GPU"),
		TelemetryURL:        configreader.MustStringFromEnv("CONST_TELEMETRY_URL", &configreader.StringValidation{Required: false, Default: consts.TelemetryURL}),
		EnableTelemetry:     env.GetBool("ENABLE_TELEMETRY"),
		OperatorInCluster:   configreader.MustBoolFromEnv("CONST_OPERATOR_IN_CLUSTER", &configreader.BoolValidation{Default: true}),
	}
	Cortex.ID = hash.String(Cortex.Bucket + Cortex.Region + Cortex.LogGroup)
	return Cortex
}
