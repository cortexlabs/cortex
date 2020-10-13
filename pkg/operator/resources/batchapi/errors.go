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

package batchapi

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

const (
	ErrJobNotFound                = "batchapi.job_not_found"
	ErrJobIsNotInProgress         = "batchapi.job_is_not_in_progress"
	ErrJobHasAlreadyBeenStopped   = "batchapi.job_has_already_been_stopped"
	ErrNoS3FilesFound             = "batchapi.no_s3_files_found"
	ErrNoDataFoundInJobSubmission = "batchapi.no_data_found_in_job_submission"
	ErrFailedToEnqueueMessages    = "batchapi.failed_to_enqueue_messages"
	ErrMessageExceedsMaxSize      = "batchapi.message_exceeds_max_size"
	ErrConflictingFields          = "batchapi.conflicting_fields"
	ErrBatchItemSizeExceedsLimit  = "batchapi.item_size_exceeds_limit"
	ErrSpecifyExactlyOneKey       = "batchapi.specify_exactly_one_key"
)

func ErrorJobNotFound(jobKey spec.JobKey) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrJobNotFound,
		Message: fmt.Sprintf("unable to find batch job %s", jobKey.UserString()),
	})
}

func ErrorJobIsNotInProgress() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrJobIsNotInProgress,
		Message: "cannot stop batch job because it is not in progress",
	})
}

func ErrorJobHasAlreadyBeenStopped() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrJobHasAlreadyBeenStopped,
		Message: "batch job has already been stopped",
	})
}

func ErrorNoS3FilesFound() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoS3FilesFound,
		Message: "no s3 files match search criteria",
	})
}

func ErrorNoDataFoundInJobSubmission() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoDataFoundInJobSubmission,
		Message: "unable to enqueue batches because no data was found",
	})
}

func ErrorFailedToEnqueueMessages(message string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFailedToEnqueueMessages,
		Message: message,
	})
}

func ErrorMessageExceedsMaxSize(messageSize int, messageLimit int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMessageExceedsMaxSize,
		Message: fmt.Sprintf("cannot enqueue message because its size of %d bytes exceeds the %d bytes limit; use a smaller batch size or reduce the size of each of item in the batch", messageSize, messageLimit),
	})
}

func ErrorConflictingFields(key string, keys ...string) error {
	allKeys := append([]string{key}, keys...)

	return errors.WithStack(&errors.Error{
		Kind:    ErrConflictingFields,
		Message: fmt.Sprintf("please specify either the %s field (but not more than one at the same time)", s.StrsOr(allKeys)),
	})
}

func ErrorItemSizeExceedsLimit(index int, size int, limit int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrBatchItemSizeExceedsLimit,
		Message: fmt.Sprintf("item %d has size %d bytes which exceeds the limit (%d bytes)", index, size, limit),
	})
}

func ErrorSpecifyExactlyOneKey(key string, keys ...string) error {
	allKeys := append([]string{key}, keys...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyExactlyOneKey,
		Message: fmt.Sprintf("specify exactly one of the following keys: %s", s.StrsOr(allKeys)),
	})
}
