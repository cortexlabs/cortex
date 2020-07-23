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
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

const (
	ErrBatchAPINotDeployed        = "batchapi.batch_api_not_deployed"
	ErrJobNotFound                = "batchapi.job_not_found"
	ErrJobIsNotInProgress         = "batchapi.job_is_not_in_progress"
	ErrJobHasAlreadyBeenStopped   = "batchapi.job_has_already_been_stopped"
	ErrNoS3FilesFound             = "batchapi.no_s3_files_found"
	ErrNoDataFoundInJobSubmission = "batchapi.no_data_found_in_job_submission"
)

func ErrorBatchAPINotDeployed(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrBatchAPINotDeployed,
		Message: fmt.Sprintf("BatchAPI api named '%s' is not deployed", apiName),
	})
}

func ErrorJobNotFound(jobKey spec.JobKey) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrJobNotFound,
		Message: fmt.Sprintf("unable to find job %s", jobKey.UserString()),
	})
}

func ErrorJobIsNotInProgress() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrJobIsNotInProgress,
		Message: "cannot stop job because it is not in progress",
	})
}

func ErrorJobHasAlreadyBeenStopped() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrJobHasAlreadyBeenStopped,
		Message: "job has already been stopped",
	})
}

func ErrorNoS3FilesFound() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoS3FilesFound,
		Message: "no s3 files found based on search criteria",
	})
}

func ErrorNoDataFoundInJobSubmission() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoDataFoundInJobSubmission,
		Message: "unable to enqueue batches because no data was found",
	})
}
