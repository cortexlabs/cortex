package env

import (
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
)

func getPaths(configName string) (string, string) {
	envVarName := "CORTEX_" + configName
	filePath := filepath.Join(consts.CortexConfigPath, configName)
	return envVarName, filePath
}

func GetStr(configName string) string {
	envVarName, filePath := getPaths(configName)
	v := &configreader.StringValidation{Required: true}
	return configreader.MustStringFromEnvOrFile(envVarName, filePath, v)
}

func GetBool(configName string) bool {
	envVarName, filePath := getPaths(configName)
	v := &configreader.BoolValidation{Default: false}
	return configreader.MustBoolFromEnvOrFile(envVarName, filePath, v)
}
