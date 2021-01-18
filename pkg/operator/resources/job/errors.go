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

package job

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	ErrInvalidJobKind           = "job.invalid_kind"
	ErrJobNotFound              = "job.not_found"
	ErrJobIsNotInProgress       = "job.job_is_not_in_progress"
	ErrJobHasAlreadyBeenStopped = "job.job_has_already_been_stopped"
	ErrConflictingFields        = "job.conflicting_fields"
	ErrSpecifyExactlyOneKey     = "job.specify_exactly_one_key"
)

func ErrorInvalidJobKind(kind userconfig.Kind) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidJobKind,
		Message: fmt.Sprintf("invalid job kind %s", kind.String()),
	})
}

func ErrorJobNotFound(jobKey spec.JobKey) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrJobNotFound,
		Message: fmt.Sprintf("unable to find %s job %s", jobKey.Kind.String(), jobKey.UserString()),
	})
}

func ErrorJobIsNotInProgress(kind userconfig.Kind) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrJobIsNotInProgress,
		Message: fmt.Sprintf("cannot stop %s job because it is not in progress", kind.String()),
	})
}

func ErrorJobHasAlreadyBeenStopped(kind userconfig.Kind) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrJobHasAlreadyBeenStopped,
		Message: fmt.Sprintf("%s job has already been stopped", kind.String()),
	})
}

func ErrorConflictingFields(key string, keys ...string) error {
	allKeys := append([]string{key}, keys...)

	return errors.WithStack(&errors.Error{
		Kind:    ErrConflictingFields,
		Message: fmt.Sprintf("please specify either the %s field (but not more than one at the same time)", s.StrsOr(allKeys)),
	})
}

func ErrorSpecifyExactlyOneKey(key string, keys ...string) error {
	allKeys := append([]string{key}, keys...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyExactlyOneKey,
		Message: fmt.Sprintf("specify exactly one of the following keys: %s", s.StrsOr(allKeys)),
	})
}
