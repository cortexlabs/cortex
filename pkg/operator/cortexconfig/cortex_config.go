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
	cr "github.com/cortexlabs/cortex/pkg/utils/configreader"
)

var (
	LogGroup            string
	Bucket              string
	Region              string
	Namespace           string
	OperatorImage       string
	SparkImage          string
	TFTrainImage        string
	TFServeImage        string
	TFAPIImage          string
	PythonPackagerImage string
)

func init() {
	LogGroup = getStr("LOG_GROUP")
	Bucket = getStr("BUCKET")
	Region = getStr("REGION")
	Namespace = getStr("NAMESPACE")
	OperatorImage = getStr("IMAGE_OPERATOR")
	SparkImage = getStr("IMAGE_SPARK")
	TFTrainImage = getStr("IMAGE_TF_TRAIN")
	TFServeImage = getStr("IMAGE_TF_SERVE")
	TFAPIImage = getStr("IMAGE_TF_API")
	PythonPackagerImage = getStr("IMAGE_PYTHON_PACKAGER")
}

//
// Helpers
//

func getPaths(configName string) (string, string) {
	envVarName := "CORTEX_" + configName
	filePath := filepath.Join(consts.CortexConfigPath, configName)
	return envVarName, filePath
}

func getStr(configName string) string {
	envVarName, filePath := getPaths(configName)
	v := &cr.StringValidation{Required: true}
	return cr.MustStringFromEnvOrFile(envVarName, filePath, v)
}
