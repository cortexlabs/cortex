/*
Copyright 2021 Cortex Labs, Inc.

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

package operator

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	_loggerTTL          = time.Hour * 1
	_evictionCronPeriod = time.Minute * 10
)

var Logger *zap.SugaredLogger

type cachedLogger struct {
	value      *zap.SugaredLogger
	lastAccess time.Time
}

type loggerCache struct {
	m map[string]*cachedLogger
	sync.Mutex
}

var _loggerCache loggerCache

func (lc *loggerCache) GetFromCacheOrNil(key string) *zap.SugaredLogger {
	lc.Lock()
	defer lc.Unlock()

	item, ok := lc.m[key]
	if ok {
		item.lastAccess = time.Now()
		return item.value
	}
	return nil
}

func init() {
	_loggerCache = loggerCache{m: map[string]*cachedLogger{}}

	_loggerCache.Lock()
	_loggerCache.Unlock()
	operatorLogLevel := os.Getenv("CORTEX_OPERATOR_LOG_LEVEL")
	if operatorLogLevel == "" {
		operatorLogLevel = "info"
	}

	operatorCortexLogLevel := userconfig.LogLevelFromString(operatorLogLevel)
	if operatorCortexLogLevel == userconfig.UnknownLogLevel {
		panic(fmt.Sprintf("incompatible log level provided: %s", operatorLogLevel))
	}

	operatorZapConfig := defaultZapConfig(operatorCortexLogLevel)
	operatorLogger, err := operatorZapConfig.Build()
	if err != nil {
		panic(err)
	}

	Logger = operatorLogger.Sugar()

	go func() {
		for range time.Tick(_evictionCronPeriod) {
			_loggerCache.Lock()
			for k, v := range _loggerCache.m {
				if time.Since(v.lastAccess) > _loggerTTL {
					delete(_loggerCache.m, k)
				}
			}
			_loggerCache.Unlock()
		}
	}()
}

func defaultZapConfig(level userconfig.LogLevel, fields ...map[string]interface{}) zap.Config {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.MessageKey = "message"

	initialFields := map[string]interface{}{}
	for _, m := range fields {
		for k, v := range m {
			initialFields[k] = v
		}
	}

	return zap.Config{
		Level:            zap.NewAtomicLevelAt(toZapLogLevel(level)),
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		InitialFields:    initialFields,
	}
}

func GetRealtimeAPILogger(apiName string, apiID string) (*zap.SugaredLogger, error) {
	loggerKey := fmt.Sprintf("apiName=%s,apiID=%s", apiName, apiID)
	logger := _loggerCache.GetFromCacheOrNil(loggerKey)

	if logger != nil {
		return logger, nil
	}

	apiSpec, err := DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	return initializeLogger(loggerKey, apiSpec.Predictor.LogLevel, map[string]interface{}{
		"apiName": apiSpec.Name,
	})
}

func GetRealtimeAPILoggerFromSpec(apiSpec *spec.API) (*zap.SugaredLogger, error) {
	loggerKey := fmt.Sprintf("apiName=%s,apiID=%s", apiSpec.Name, apiSpec.ID)
	logger := _loggerCache.GetFromCacheOrNil(loggerKey)
	if logger != nil {
		return logger, nil
	}

	return initializeLogger(loggerKey, apiSpec.Predictor.LogLevel, map[string]interface{}{
		"apiName": apiSpec.Name,
	})
}

func GetJobLogger(jobKey spec.JobKey) (*zap.SugaredLogger, error) {
	loggerKey := fmt.Sprintf("apiName=%s,jobID=%s", jobKey.APIName, jobKey.ID)
	logger := _loggerCache.GetFromCacheOrNil(loggerKey)
	if logger != nil {
		return logger, nil
	}

	jobSpec, err := DownloadJobSpec(jobKey)
	if err != nil {
		return nil, err
	}

	apiSpec, err := DownloadAPISpec(jobKey.APIName, jobSpec.APIID)
	if err != nil {
		return nil, err
	}

	return initializeLogger(loggerKey, apiSpec.Predictor.LogLevel, map[string]interface{}{
		"apiName": jobKey.APIName,
		"jobID":   jobKey.ID,
	})
}

func GetJobLoggerFromSpec(apiSpec *spec.API, jobKey spec.JobKey) (*zap.SugaredLogger, error) {
	loggerKey := fmt.Sprintf("apiName=%s,jobID=%s", jobKey.APIName, jobKey.ID)
	logger := _loggerCache.GetFromCacheOrNil(loggerKey)
	if logger != nil {
		return logger, nil
	}

	return initializeLogger(loggerKey, apiSpec.Predictor.LogLevel, map[string]interface{}{
		"apiName": jobKey.APIName,
		"jobID":   jobKey.ID,
	})
}

func initializeLogger(key string, level userconfig.LogLevel, fields map[string]interface{}) (*zap.SugaredLogger, error) {
	logger, err := defaultZapConfig(level, fields).Build()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sugarLogger := logger.Sugar()

	_loggerCache.Lock()
	defer _loggerCache.Unlock()

	_loggerCache.m[key] = &cachedLogger{
		lastAccess: time.Now(),
		value:      sugarLogger,
	}

	return sugarLogger, nil
}

func toZapLogLevel(logLevel userconfig.LogLevel) zapcore.Level {
	switch logLevel {
	case userconfig.InfoLogLevel:
		return zapcore.InfoLevel
	case userconfig.WarningLogLevel:
		return zapcore.WarnLevel
	case userconfig.ErrorLogLevel:
		return zapcore.ErrorLevel
	default:
		return zapcore.DebugLevel
	}
}
