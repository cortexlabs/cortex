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
	"fmt"
	"os"
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
	Enable          bool
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
	if !telemetryConfig.Enable {
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
			Extra: map[string]interface{}{
				"direct": true,
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

func ReportEvent(name string, properties map[string]interface{}) {
	fmt.Println("RECORDING EVENT:", name)

	if _config == nil {
		return
	}

	err := _segment.Enqueue(analytics.Track{
		Event:      name,
		UserId:     _config.UserID,
		Properties: properties,
	})

	if err != nil {
		ReportError(err)
	}
}

func ReportError(err error) {
	fmt.Println("RECORDING ERROR:", err.Error())

	if err == nil || _config == nil {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{ID: _config.UserID})
		sentry.CaptureException(err)
		go sentry.Flush(5 * time.Second)
	})
}

func RecordEmail(userID string, email string) {
	fmt.Println("RECORDING EMAIL:", email)

	if _config == nil {
		return
	}

	_segment.Enqueue(analytics.Identify{
		UserId: userID,
		Traits: analytics.NewTraits().
			SetEmail(email),
	})
}

func RecordOperatorID(userID string, operatorID string) {
	fmt.Println("RECORDING OPERATOR ID:", operatorID)

	if _config == nil {
		return
	}

	_segment.Enqueue(analytics.Identify{
		UserId: userID,
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

func ExitErr(errs ...interface{}) {
	err := errors.MergeErrItems(errs...)
	if err == nil {
		err = errors.New("blank")
	}
	ReportError(err)

	Close()

	errors.Exit(errs...)
}

func ExitOk() {
	Close()
	os.Exit(0)
}
