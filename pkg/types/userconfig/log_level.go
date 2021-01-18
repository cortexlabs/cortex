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

package userconfig

import "go.uber.org/zap/zapcore"

type LogLevel int

const (
	UnknownLogLevel LogLevel = iota
	DebugLogLevel
	InfoLogLevel
	WarningLogLevel
	ErrorLogLevel
)

var _logLevels = []string{
	"unknown",
	"debug",
	"info",
	"warning",
	"error",
}

func LogLevelFromString(s string) LogLevel {
	for i := 0; i < len(_logLevels); i++ {
		if s == _logLevels[i] {
			return LogLevel(i)
		}
	}
	return UnknownLogLevel
}

func LogLevelTypes() []string {
	return _logLevels[1:]
}

func (t LogLevel) String() string {
	return _logLevels[t]
}

// MarshalText satisfies TextMarshaler
func (t LogLevel) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *LogLevel) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_logLevels); i++ {
		if enum == _logLevels[i] {
			*t = LogLevel(i)
			return nil
		}
	}

	*t = UnknownLogLevel
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *LogLevel) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t LogLevel) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

// Value to set for TF_CPP_MIN_LOG_LEVEL environment variable. Default is 0.
func TFNumericLogLevelFromLogLevel(logLevel LogLevel) int {
	var tfLogLevelNumber int
	switch logLevel.String() {
	case "debug":
		tfLogLevelNumber = 0
	case "info":
		tfLogLevelNumber = 0
	case "warning":
		tfLogLevelNumber = 1
	case "error":
		tfLogLevelNumber = 2
	}
	return tfLogLevelNumber
}

func ToZapLogLevel(logLevel LogLevel) zapcore.Level {
	switch logLevel {
	case InfoLogLevel:
		return zapcore.InfoLevel
	case WarningLogLevel:
		return zapcore.WarnLevel
	case ErrorLogLevel:
		return zapcore.ErrorLevel
	default:
		return zapcore.DebugLevel
	}
}
