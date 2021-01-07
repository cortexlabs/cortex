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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrUserIDNotSpecified         = "telemetry.user_id_not_specified"
	ErrSentryFlushTimeoutExceeded = "telemetry.sentry_flush_timeout_exceeded"
)

func ErrorUserIDNotSpecified() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrUserIDNotSpecified,
		Message: "user ID must be specified to enable telemetry",
	})
}

func ErrorSentryFlushTimeoutExceeded() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSentryFlushTimeoutExceeded,
		Message: "sentry flush timout exceeded",
	})
}
