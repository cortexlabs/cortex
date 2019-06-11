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
	"github.com/cortexlabs/cortex/pkg/lib/argo"
	"github.com/cortexlabs/cortex/pkg/lib/cloud"
	"github.com/cortexlabs/cortex/pkg/lib/cloud/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cloud/local"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/spark"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
)

var (
	Cortex     *CortexConfig
	Cloud      cloud.Client
	Kubernetes *k8s.Client
	Telemetry  *telemetry.Client
	Argo       *argo.Client
	Spark      *spark.Client
)

type CortexConfig struct {
	CloudProvider       cloud.ProviderType `json:"cloud_provider_type"`
	CloudConfig         *cloud.Config      `json:"cloud_config"`
	APIVersion          string             `json:"api_version"`
	Namespace           string             `json:"namespace"`
	OperatorImage       string             `json:"operator_image"`
	SparkImage          string             `json:"spark_image"`
	TFTrainImage        string             `json:"tf_train_image"`
	TFServeImage        string             `json:"tf_serve_image"`
	TFAPIImage          string             `json:"tf_api_image"`
	PythonPackagerImage string             `json:"python_packager_image"`
	TFTrainImageGPU     string             `json:"tf_train_image_gpu"`
	TFServeImageGPU     string             `json:"tf_serve_image_gpu"`
	TelemetryURL        string             `json:"telemetry_url"`
	EnableTelemetry     bool               `json:"enable_telemetry"`
}

func InitCortexConfig() {
	cloudTypeStr := getStr("CLOUD_PROVIDER_TYPE")
	cloudProviderType := cloud.ProviderTypeFromString(cloudTypeStr)
	if cloudProviderType == cloud.UnknownProviderType {
		panic(cloud.ErrorUnsupportedProviderType(cloudTypeStr))
	}

	config := cloud.Config{
		Bucket:             getStr("BUCKET"),
		Region:             getStrPtr("REGION"),
		LogGroup:           getStrPtr("LOG_GROUP"),
		OperatorLocalMount: getStrPtr("OPERATOR_LOCAL_MOUNT"),
		OperatorHostIP:     getStrPtr("OPERATOR_HOST_IP"),
		OperatorInCluster:  configreader.MustBoolFromEnv("CONST_OPERATOR_IN_CLUSTER", &configreader.BoolValidation{Default: true}),
	}

	Cortex = &CortexConfig{
		CloudProvider:       cloudProviderType,
		CloudConfig:         &config,
		APIVersion:          consts.CortexVersion,
		Namespace:           getStr("NAMESPACE"),
		OperatorImage:       getStr("IMAGE_OPERATOR"),
		SparkImage:          getStr("IMAGE_SPARK"),
		TFTrainImage:        getStr("IMAGE_TF_TRAIN"),
		TFServeImage:        getStr("IMAGE_TF_SERVE"),
		TFAPIImage:          getStr("IMAGE_TF_API"),
		PythonPackagerImage: getStr("IMAGE_PYTHON_PACKAGER"),
		TFTrainImageGPU:     getStr("IMAGE_TF_TRAIN_GPU"),
		TFServeImageGPU:     getStr("IMAGE_TF_SERVE_GPU"),
		TelemetryURL:        configreader.MustStringFromEnv("CORTEX_TELEMETRY_URL", &configreader.StringValidation{Required: false, Default: consts.TelemetryURL}),
		EnableTelemetry:     getBool("ENABLE_TELEMETRY"),
	}
}

func InitCloud(cortexConfig *CortexConfig) error {
	var err error

	switch cortexConfig.CloudProvider {
	case cloud.LocalProviderType:
		if Cloud, err = local.NewFromConfig(cortexConfig.CloudConfig); err != nil {
			return err
		}
	case cloud.AWSProviderType:
		if Cloud, err = aws.NewFromConfig(cortexConfig.CloudConfig); err != nil {
			return err
		}
	default:
		return cloud.ErrorUnsupportedProviderType(cortexConfig.CloudProvider.String())
	}
	return nil
}

func InitClients(cloud cloud.Client) error {
	var err error

	Telemetry = telemetry.New(Cortex.TelemetryURL, cloud.HashedAccountID(), Cortex.EnableTelemetry)

	if Kubernetes, err = k8s.New(Cortex.Namespace, Cortex.CloudConfig.OperatorInCluster); err != nil {
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

func getStrPtr(name string) *string {
	envVarName, filePath := getPaths(name)
	v := &configreader.StringPtrValidation{Default: nil}
	return configreader.MustStringPtrFromEnvOrFile(envVarName, filePath, v)
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
