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

package telemetry

import (
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
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
	Properties  map[string]string
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

func getSentryDSN() string {
	if envVar := os.Getenv("CORTEX_TELEMETRY_SENTRY_DSN"); envVar != "" {
		return envVar
	}
	return _sentryDSN
}

func Init(telemetryConfig Config) error {
	if !telemetryConfig.Enabled {
		_config = nil
		return nil
	}

	if telemetryConfig.UserID == "" {
		return ErrorUserIDNotSpecified()
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:         getSentryDSN(),
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

	eventHelper(name, maps.MergeStrInterfaceMaps(properties...), integrations)
}

func EventNotify(name string, properties ...map[string]interface{}) {
	integrations := map[string]interface{}{
		"All": true,
	}

	eventHelper(name, maps.MergeStrInterfaceMaps(properties...), integrations)
}

func eventHelper(name string, properties map[string]interface{}, integrations map[string]interface{}) {
	if _config == nil || !_config.Enabled || strings.ToLower(os.Getenv("CORTEX_TELEMETRY_DISABLE")) == "true" {
		return
	}

	mergedProperties := maps.MergeStrInterfaceMaps(properties, cast.StrMapToStrInterfaceMap(_config.Properties))

	err := _segment.Enqueue(analytics.Track{
		Event:        name,
		UserId:       _config.UserID,
		Properties:   mergedProperties,
		Integrations: integrations,
	})
	if err != nil {
		Error(err)
	}
}

func Error(err error, tags ...map[string]string) {
	if err == nil || _config == nil {
		return
	}

	if shouldBlock(err, _config.BackoffMode) {
		return
	}

	mergedTags := maps.MergeStrMaps(tags...)

	sentry.WithScope(func(scope *sentry.Scope) {
		e := EventFromException(err)
		scope.SetUser(sentry.User{ID: _config.UserID})
		scope.SetTags(maps.MergeStrMaps(_config.Properties, mergedTags))
		scope.SetTags(map[string]string{"error_type": e.Exception[0].Type})
		sentry.CaptureEvent(e)

		go sentry.Flush(10 * time.Second)
	})
}

func EventFromException(exception error) *sentry.Event {
	stacktrace := sentry.ExtractStacktrace(exception)

	if stacktrace == nil {
		stacktrace = sentry.NewStacktrace()
	}

	errTypeString := reflect.TypeOf(errors.CauseOrSelf(exception)).String()

	errKind := errors.GetKind(exception)
	if errKind != "" && errKind != errors.ErrNotCortexError {
		errTypeString = errKind
	}

	event := sentry.NewEvent()
	event.Level = sentry.LevelError

	value := errors.Message(exception)
	if metadata := errors.GetMetadata(exception); metadata != nil {
		value = value + "\n\n########## metadata ##########\n" + s.ObjStripped(metadata)
	}

	event.Exception = []sentry.Exception{{
		Value:      value,
		Type:       errTypeString,
		Stacktrace: stacktrace,
	}}
	return event
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
		return ErrorSentryFlushTimeoutExceeded()
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
