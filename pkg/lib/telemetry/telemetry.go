/*
Copyright 2020 Cortex Labs, Inc.

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

package telemetry

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/getsentry/sentry-go"
	"gopkg.in/segmentio/analytics-go.v3"
)

var _sentryDSN = "https://5cea3d2d67194d028f7191fcc6ebca14@sentry.io/1825326"
var _segmentWriteKey = "BNhXifMk9EyhPICF2zAFpWYPCf4CRpV1"

var _segment analytics.Client
var _config *Config

type Config struct {
	Enabled     bool
	UserID      string
	Properties  map[string]interface{}
	Environment string
	LogErrors   bool
	BackoffMode BackoffMode
}

type BackoffMode int

const (
	NoBackoff BackoffMode = iota
	BackoffDuplicateMessages
	BackoffAnyMessages
)

type silentSegmentLogger struct{}

func (logger silentSegmentLogger) Logf(format string, args ...interface{}) {
	return
}

func (logger silentSegmentLogger) Errorf(format string, args ...interface{}) {
	return
}

type silentSentryLogger struct{}

func (logger silentSentryLogger) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func Init(telemetryConfig Config) error {
	if !telemetryConfig.Enabled {
		_config = nil
		return nil
	}

	if telemetryConfig.UserID == "" {
		return errors.New("user ID must be specified to enable telemetry")
	}

	dsn := _sentryDSN
	if envVar := os.Getenv("CORTEX_TELEMETRY_SENTRY_DSN"); envVar != "" {
		dsn = envVar
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Release:     consts.CortexVersion,
		Environment: telemetryConfig.Environment,
	})
	if err != nil {
		_config = nil
		return err
	}

	var segmentLogger analytics.Logger
	if !telemetryConfig.LogErrors {
		sentry.Logger.SetOutput(silentSentryLogger{})
		segmentLogger = silentSegmentLogger{}
	}

	writeKey := _segmentWriteKey
	if envVar := os.Getenv("CORTEX_TELEMETRY_SEGMENT_WRITE_KEY"); envVar != "" {
		writeKey = envVar
	}

	_segment, err = analytics.NewWithConfig(writeKey, analytics.Config{
		BatchSize: 1,
		Logger:    segmentLogger,
		DefaultContext: &analytics.Context{
			App: analytics.AppInfo{
				Version: consts.CortexVersion,
			},
			Device: analytics.DeviceInfo{
				Type: telemetryConfig.Environment,
			},
		},
	})
	if err != nil {
		_config = nil
		return err
	}

	_config = &telemetryConfig
	return nil
}

func Event(name string, properties ...map[string]interface{}) {
	integrations := map[string]interface{}{
		"All":   true,
		"Slack": false,
	}

	eventHelper(name, mergeProperties(properties...), integrations)
}

func EventNotify(name string, properties ...map[string]interface{}) {
	integrations := map[string]interface{}{
		"All": true,
	}

	eventHelper(name, mergeProperties(properties...), integrations)
}

func eventHelper(name string, properties map[string]interface{}, integrations map[string]interface{}) {
	if _config == nil || !_config.Enabled || strings.ToLower(os.Getenv("CORTEX_TELEMETRY_DISABLE")) == "true" {
		return
	}

	err := _segment.Enqueue(analytics.Track{
		Event:        name,
		UserId:       _config.UserID,
		Properties:   mergeProperties(properties, _config.Properties),
		Integrations: integrations,
	})
	if err != nil {
		Error(err)
	}
}

func Error(err error) {
	if err == nil || _config == nil {
		return
	}

	if shouldBlock(err, _config.BackoffMode) {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{ID: _config.UserID})
		scope.SetExtras(_config.Properties)
		scope.SetTag("is_user", strconv.FormatBool(errors.IsUser(err)))
		e := EventFromException(err)
		sentry.CaptureEvent(e)

		go sentry.Flush(10 * time.Second)
	})
}

func EventFromException(exception error) *sentry.Event {
	stacktrace := sentry.ExtractStacktrace(exception)

	if stacktrace == nil {
		stacktrace = sentry.NewStacktrace()
	}

	cause := exception
	// Handle wrapped errors for github.com/pingcap/errors and github.com/pkg/errors
	if ex, ok := exception.(interface{ Cause() error }); ok {
		cause = ex.Cause()
	}

	event := sentry.NewEvent()
	event.Level = sentry.LevelError

	errTypeString := reflect.TypeOf(cause).String()
	errKind := errors.GetKind(exception)
	if errKind != errors.ErrUnknown {
		errTypeString = errKind.String()
	}

	event.Exception = []sentry.Exception{{
		Value:      cause.Error(),
		Type:       errTypeString,
		Stacktrace: stacktrace,
	}}
	return event
}

func ErrorMessage(message string) {
	if _config == nil || !_config.Enabled || strings.ToLower(os.Getenv("CORTEX_TELEMETRY_DISABLE")) == "true" {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{ID: _config.UserID})
		scope.SetExtras(_config.Properties)
		sentry.CaptureMessage(message)
		go sentry.Flush(10 * time.Second)
	})
}

func RecordEmail(email string) {
	if _config == nil || !_config.Enabled || strings.ToLower(os.Getenv("CORTEX_TELEMETRY_DISABLE")) == "true" {
		return
	}

	_segment.Enqueue(analytics.Identify{
		UserId: _config.UserID,
		Traits: analytics.NewTraits().
			SetEmail(email),
	})
}

func RecordOperatorID(clientID string, operatorID string) {
	if _config == nil || !_config.Enabled || strings.ToLower(os.Getenv("CORTEX_TELEMETRY_DISABLE")) == "true" {
		return
	}

	_segment.Enqueue(analytics.Identify{
		UserId: clientID,
		Traits: analytics.NewTraits().
			Set("operator_id", operatorID),
	})
}

func closeSentry() error {
	if !sentry.Flush(5 * time.Second) {
		return errors.New("sentry flush timout exceeded")
	}
	return nil
}

func closeSegment() error {
	if _segment == nil {
		return nil
	}
	return _segment.Close()
}

func Close() {
	parallel.Run(closeSegment, closeSentry)
	_config = nil
}

func mergeProperties(properties ...map[string]interface{}) map[string]interface{} {
	mergedProperties := make(map[string]interface{})
	for _, p := range properties {
		for k, v := range p {
			mergedProperties[k] = v
		}
	}
	return mergedProperties
}
