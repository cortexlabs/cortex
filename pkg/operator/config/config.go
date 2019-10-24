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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
)

var (
	Cortex          *CortexConfig
	AWS             *aws.Client
	Kubernetes      *k8s.Client
	IstioKubernetes *k8s.Client
	Telemetry       *telemetry.Client
)

type CortexConfig struct {
	ID                  string `json:"id"`
	APIVersion          string `json:"api_version"`
	Bucket              string `json:"bucket"`
	LogGroup            string `json:"log_group"`
	Region              string `json:"region"`
	Namespace           string `json:"namespace"`
	OperatorImage       string `json:"operator_image"`
	TFServeImage        string `json:"tf_serve_image"`
	TFAPIImage          string `json:"tf_api_image"`
	DownloaderImage     string `json:"downloader_image"`
	TFServeImageGPU     string `json:"tf_serve_image_gpu"`
	ONNXServeImage      string `json:"onnx_serve_image"`
	ONNXServeImageGPU   string `json:"onnx_serve_image_gpu"`
	PythonServeImage    string `json:"python_serve_image"`
	PythonServeImageGPU string `json:"python_serve_image_gpu"`

	TelemetryURL      string `json:"telemetry_url"`
	EnableTelemetry   bool   `json:"enable_telemetry"`
	OperatorInCluster bool   `json:"operator_in_cluster"`
}

func Init() error {
	Cortex = &CortexConfig{
		APIVersion:          consts.CortexVersion,
		Bucket:              getStr("BUCKET"),
		LogGroup:            getStr("LOG_GROUP"),
		Region:              getStr("REGION"),
		Namespace:           getStr("NAMESPACE"),
		OperatorImage:       getStr("IMAGE_OPERATOR"),
		TFServeImage:        getStr("IMAGE_TF_SERVE"),
		TFAPIImage:          getStr("IMAGE_TF_API"),
		DownloaderImage:     getStr("IMAGE_DOWNLOADER"),
		TFServeImageGPU:     getStr("IMAGE_TF_SERVE_GPU"),
		ONNXServeImage:      getStr("IMAGE_ONNX_SERVE"),
		ONNXServeImageGPU:   getStr("IMAGE_ONNX_SERVE_GPU"),
		PythonServeImage:    getStr("IMAGE_PYTHON"),
		PythonServeImageGPU: getStr("IMAGE_PYTHON_GPU"),

		TelemetryURL:      configreader.MustStringFromEnv("CORTEX_TELEMETRY_URL", &configreader.StringValidation{Required: false, Default: consts.TelemetryURL}),
		EnableTelemetry:   getBool("ENABLE_TELEMETRY", false),
		OperatorInCluster: getBool("OPERATOR_IN_CLUSTER", true),
	}
	Cortex.ID = hash.String(Cortex.Bucket + Cortex.Region + Cortex.LogGroup)

	var err error

	AWS, err = aws.New(Cortex.Region, Cortex.Bucket, true)
	if err != nil {
		errors.Exit(err)
	}
	Telemetry = telemetry.New(Cortex.TelemetryURL, AWS.HashedAccountID, Cortex.EnableTelemetry)

	if Kubernetes, err = k8s.New(Cortex.Namespace, Cortex.OperatorInCluster); err != nil {
		return err
	}

	if IstioKubernetes, err = k8s.New("istio-system", Cortex.OperatorInCluster); err != nil {
		return err
	}

	return nil
}

func getPaths(name string) (string, string) {
	envVarName := "CORTEX_" + name
	filePath := filepath.Join(consts.CortexConfigPath, name)
	return envVarName, filePath
}

func getStrDefault(name string, defaultVal string) string {
	v := &configreader.StringValidation{Required: false, Default: defaultVal}
	envVarName, filePath := getPaths(name)
	return configreader.MustStringFromEnvOrFile(envVarName, filePath, v)
}

func getStr(name string) string {
	v := &configreader.StringValidation{Required: true}
	envVarName, filePath := getPaths(name)
	return configreader.MustStringFromEnvOrFile(envVarName, filePath, v)
}

func getBool(name string, defaultVal bool) bool {
	envVarName, filePath := getPaths(name)
	v := &configreader.BoolValidation{Default: defaultVal}
	return configreader.MustBoolFromEnvOrFile(envVarName, filePath, v)
}
