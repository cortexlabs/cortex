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

package telemetry

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/getsentry/sentry-go"
	"gopkg.in/segmentio/analytics-go.v3"
)

var _segment analytics.Client
var _config *Config

type Config struct {
	Enabled         bool
	UserID          string
	Environment     string
	ShouldLogErrors bool
}

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

	err := sentry.Init(sentry.ClientOptions{
		Dsn:         "https://5cea3d2d67194d028f7191fcc6ebca14@sentry.io/1825326",
		Release:     consts.CortexVersion,
		Environment: telemetryConfig.Environment,
	})
	if err != nil {
		_config = nil
		return err
	}

	var segmentLogger analytics.Logger
	if !telemetryConfig.ShouldLogErrors {
		sentry.Logger.SetOutput(silentSentryLogger{})
		segmentLogger = silentSegmentLogger{}
	}

	_segment, err = analytics.NewWithConfig("0WvoJyCey9z1W2EW7rYTPJUMRYat46dl", analytics.Config{
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
	eventHelper(name, mergeProperties(properties...), nil)
}

func InternalEvent(name string, properties ...map[string]interface{}) {
	integrations := map[string]interface{}{
		"All": false,
		"S3":  true,
	}

	eventHelper(name, mergeProperties(properties...), integrations)
}

func eventHelper(name string, properties map[string]interface{}, integrations map[string]interface{}) {
	if _config == nil || !_config.Enabled {
		return
	}

	err := _segment.Enqueue(analytics.Track{
		Event:        name,
		UserId:       _config.UserID,
		Properties:   properties,
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

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{ID: _config.UserID})
		sentry.CaptureException(err)
		go sentry.Flush(10 * time.Second)
	})
}

func ErrorMessage(message string) {
	if _config == nil || !_config.Enabled {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{ID: _config.UserID})
		sentry.CaptureMessage(message)
		go sentry.Flush(10 * time.Second)
	})
}

func RecordEmail(email string) {
	if _config == nil || !_config.Enabled {
		return
	}

	_segment.Enqueue(analytics.Identify{
		UserId: _config.UserID,
		Traits: analytics.NewTraits().
			SetEmail(email),
	})
}

func RecordOperatorID(clientID string, operatorID string) {
	if _config == nil || !_config.Enabled {
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
