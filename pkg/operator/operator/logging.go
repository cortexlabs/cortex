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
	"strings"
	"sync"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/lib/logging"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"go.uber.org/zap"
)

const (
	_loggerTTL          = time.Hour * 1
	_evictionCronPeriod = time.Minute * 10
)

type cachedLogger struct {
	value      *zap.SugaredLogger
	lastAccess time.Time
}

type loggerCache struct {
	m map[string]*cachedLogger
	sync.Mutex
}

var _loggerCache loggerCache

func init() {
	_loggerCache = loggerCache{m: map[string]*cachedLogger{}}

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

func getFromCacheOrNil(key string) *zap.SugaredLogger {
	_loggerCache.Lock()
	defer _loggerCache.Unlock()

	item, ok := _loggerCache.m[key]
	if ok {
		item.lastAccess = time.Now()
		return item.value
	}
	return nil
}

func initializeLogger(key string, level userconfig.LogLevel, fields map[string]interface{}) (*zap.SugaredLogger, error) {
	loggerConfig := logging.DefaultZapConfig(level, fields)

	disableJSONLogging := strings.ToLower(os.Getenv("CORTEX_DISABLE_JSON_LOGGING"))
	if disableJSONLogging == "true" {
		loggerConfig.Encoding = "console"
	}

	logger, err := loggerConfig.Build()
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

func GetRealtimeAPILogger(apiName string, apiID string) (*zap.SugaredLogger, error) {
	loggerCacheKey := fmt.Sprintf("apiName=%s,apiID=%s", apiName, apiID)
	logger := getFromCacheOrNil(loggerCacheKey)

	if logger != nil {
		return logger, nil
	}

	apiSpec, err := DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	return initializeLogger(loggerCacheKey, apiSpec.Predictor.LogLevel, map[string]interface{}{
		"apiName": apiSpec.Name,
		"apiKind": apiSpec.Kind.String(),
		"apiID":   apiSpec.ID,
	})
}

func GetRealtimeAPILoggerFromSpec(apiSpec *spec.API) (*zap.SugaredLogger, error) {
	loggerCacheKey := fmt.Sprintf("apiName=%s,apiID=%s", apiSpec.Name, apiSpec.ID)
	logger := getFromCacheOrNil(loggerCacheKey)
	if logger != nil {
		return logger, nil
	}

	return initializeLogger(loggerCacheKey, apiSpec.Predictor.LogLevel, map[string]interface{}{
		"apiName": apiSpec.Name,
		"apiKind": apiSpec.Kind.String(),
		"apiID":   apiSpec.ID,
	})
}

func GetJobLogger(jobKey spec.JobKey) (*zap.SugaredLogger, error) {
	loggerCacheKey := fmt.Sprintf("apiName=%s,jobID=%s", jobKey.APIName, jobKey.ID)
	logger := getFromCacheOrNil(loggerCacheKey)
	if logger != nil {
		return logger, nil
	}

	apiName := jobKey.APIName
	var logLevel userconfig.LogLevel
	switch jobKey.Kind {
	case userconfig.BatchAPIKind:
		jobSpec, err := DownloadBatchJobSpec(jobKey)
		if err != nil {
			return nil, err
		}
		apiSpec, err := DownloadAPISpec(apiName, jobSpec.APIID)
		if err != nil {
			return nil, err
		}
		logLevel = apiSpec.Predictor.LogLevel
	case userconfig.TaskAPIKind:
		jobSpec, err := DownloadTaskJobSpec(jobKey)
		if err != nil {
			return nil, err
		}
		apiSpec, err := DownloadAPISpec(apiName, jobSpec.APIID)
		if err != nil {
			return nil, err
		}
		logLevel = apiSpec.TaskDefinition.LogLevel
	default:
		return nil, errors.ErrorUnexpected("unexpected kind", jobKey.Kind.String())
	}

	return initializeLogger(loggerCacheKey, logLevel, map[string]interface{}{
		"apiName": jobKey.APIName,
		"apiKind": jobKey.Kind.String(),
		"jobID":   jobKey.ID,
	})
}

func GetJobLoggerFromSpec(apiSpec *spec.API, jobKey spec.JobKey) (*zap.SugaredLogger, error) {
	loggerCacheKey := fmt.Sprintf("apiName=%s,jobID=%s", jobKey.APIName, jobKey.ID)
	logger := getFromCacheOrNil(loggerCacheKey)
	if logger != nil {
		return logger, nil
	}

	return initializeLogger(loggerCacheKey, apiSpec.Predictor.LogLevel, map[string]interface{}{
		"apiName": jobKey.APIName,
		"apiKind": jobKey.Kind.String(),
		"jobID":   jobKey.ID,
	})
}
