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
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/spark"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/argo"
)

var (
	Cortex     *CortexConfig
	AWS        *aws.Client
	Kubernetes *k8s.Client
	Telemetry  *telemetry.Client
	Argo       *argo.Client
	Spark      *spark.Client
)

type CortexConfig struct {
	ID                  string `json:"id"`
	APIVersion          string `json:"api_version"`
	Bucket              string `json:"bucket"`
	LogGroup            string `json:"log_group"`
	Region              string `json:"region"`
	Namespace           string `json:"namespace"`
	OperatorImage       string `json:"operator_image"`
	SparkImage          string `json:"spark_image"`
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

func Init() error {
	Cortex = &CortexConfig{
		APIVersion:          consts.CortexVersion,
		Bucket:              getStr("BUCKET"),
		LogGroup:            getStr("LOG_GROUP"),
		Region:              getStr("REGION"),
		Namespace:           getStr("NAMESPACE"),
		OperatorImage:       getStr("IMAGE_OPERATOR"),
		SparkImage:          getStr("IMAGE_SPARK"),
		TFTrainImage:        getStr("IMAGE_TF_TRAIN"),
		TFServeImage:        getStr("IMAGE_TF_SERVE"),
		TFAPIImage:          getStr("IMAGE_TF_API"),
		PythonPackagerImage: getStr("IMAGE_PYTHON_PACKAGER"),
		TFTrainImageGPU:     getStr("IMAGE_TF_TRAIN_GPU"),
		TFServeImageGPU:     getStr("IMAGE_TF_SERVE_GPU"),
		TelemetryURL:        configreader.MustStringFromEnv("CONST_TELEMETRY_URL", &configreader.StringValidation{Required: false, Default: consts.TelemetryURL}),
		EnableTelemetry:     getBool("ENABLE_TELEMETRY"),
		OperatorInCluster:   configreader.MustBoolFromEnv("CONST_OPERATOR_IN_CLUSTER", &configreader.BoolValidation{Default: true}),
	}
	Cortex.ID = hash.String(Cortex.Bucket + Cortex.Region + Cortex.LogGroup)

	AWS = aws.New(Cortex.Region, Cortex.Bucket)
	Telemetry = telemetry.New(Cortex.TelemetryURL, AWS.HashedAccountID, Cortex.EnableTelemetry)

	var err error
	if Kubernetes, err = k8s.New(Cortex.Namespace, Cortex.OperatorInCluster); err != nil {
		return err
	}

	Argo = argo.New(Kubernetes.RestConfig, Kubernetes.Namespace)

	if Spark, err = spark.New(Kubernetes.RestConfig, Kubernetes.Namespace); err != nil {
		return err
	}

	return nil
}

func getPaths(name string) (string, string) {
	envVarName := "CORTEX_" + name
	filePath := filepath.Join(consts.CortexConfigPath, name)
	return envVarName, filePath
}

func getStr(name string) string {
	envVarName, filePath := getPaths(name)
	v := &configreader.StringValidation{Required: true}
	return configreader.MustStringFromEnvOrFile(envVarName, filePath, v)
}

func getBool(name string) bool {
	envVarName, filePath := getPaths(name)
	v := &configreader.BoolValidation{Default: false}
	return configreader.MustBoolFromEnvOrFile(envVarName, filePath, v)
}
