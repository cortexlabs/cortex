/*
Copyright 2022 Cortex Labs, Inc.

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

package logging

import (
	"os"
	"strings"
	"sync"

	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger
var loggerLock sync.Mutex

func initializeLogger() {
	logLevel := strings.ToLower(os.Getenv("CORTEX_LOG_LEVEL"))
	if logLevel == "" {
		logLevel = "info"
	}

	cortexLogLevel := userconfig.LogLevelFromString(logLevel)
	if cortexLogLevel == userconfig.UnknownLogLevel {
		panic(ErrorInvalidLogLevel(logLevel, userconfig.LogLevelTypes()))
	}

	zapConfig := DefaultZapConfig(cortexLogLevel)

	disableJSONLogging := strings.ToLower(os.Getenv("CORTEX_DISABLE_JSON_LOGGING"))
	if disableJSONLogging == "true" {
		zapConfig.Encoding = "console"
	}

	zapLogger, err := zapConfig.Build()
	if err != nil {
		panic(err)
	}

	logger = zapLogger.Sugar()
}

func GetLogger() *zap.SugaredLogger {
	loggerLock.Lock()
	defer loggerLock.Unlock()

	if logger == nil {
		initializeLogger()
	}
	return logger
}

func DefaultZapConfig(level userconfig.LogLevel, fields ...map[string]interface{}) zap.Config {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.MessageKey = "message"

	labels := map[string]interface{}{}
	for _, m := range fields {
		for k, v := range m {
			labels[k] = v
		}
	}

	initialFields := map[string]interface{}{}
	if len(labels) > 0 {
		initialFields["cortex.labels"] = labels
	}

	return zap.Config{
		Level:            zap.NewAtomicLevelAt(userconfig.ToZapLogLevel(level)),
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		InitialFields:    initialFields,
	}
}
