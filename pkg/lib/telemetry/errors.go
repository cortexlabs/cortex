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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrUserIDNotSpecified
	ErrSentryFlushTimeoutExceeded
)

var _errorKinds = []string{
	"telemetry.unknown",
	"telemetry.user_id_not_specified",
	"telemetry.sentry_flush_timeout_exceeded",
}

var _ = [1]int{}[int(ErrSentryFlushTimeoutExceeded)-(len(_errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return _errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_errorKinds); i++ {
		if enum == _errorKinds[i] {
			*t = ErrorKind(i)
			return nil
		}
	}

	*t = ErrUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ErrorKind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ErrorKind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

func ErrorUserIDNotSpecified() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrUserIDNotSpecified,
		Message: "user ID must be specified to enable telemetry",
	})
}

func ErrorSentryFlushTimeoutExceeded() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrSentryFlushTimeoutExceeded,
		Message: "sentry flush timout exceeded",
	})
}
