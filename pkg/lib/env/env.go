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

package env

import (
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
)

func getPaths(name string) (string, string) {
	envVarName := "CORTEX_" + name
	filePath := filepath.Join(consts.CortexConfigPath, name)
	return envVarName, filePath
}

func GetStr(name string) string {
	envVarName, filePath := getPaths(name)
	v := &configreader.StringValidation{Required: true}
	return configreader.MustStringFromEnvOrFile(envVarName, filePath, v)
}

func GetStrWithDefault(name, defaultValue string) string {
	envVarName, filePath := getPaths(name)
	v := &configreader.StringValidation{Required: false, Default: defaultValue}
	return configreader.MustStringFromEnvOrFile(envVarName, filePath, v)
}

func GetBool(name string) bool {
	envVarName, filePath := getPaths(name)
	v := &configreader.BoolValidation{Default: false}
	return configreader.MustBoolFromEnvOrFile(envVarName, filePath, v)
}

func GetBoolWithDefault(name string, defaultValue bool) bool {
	envVarName, filePath := getPaths(name)
	v := &configreader.BoolValidation{Default: defaultValue}
	return configreader.MustBoolFromEnvOrFile(envVarName, filePath, v)
}
