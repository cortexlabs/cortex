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
