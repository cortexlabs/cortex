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

package gcp

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"google.golang.org/api/googleapi"
)

const (
	ErrCredentialsFileEnvVarNotSet = "gcp.credentials_file_env_var_not_set"
	ErrProjectIDMismatch           = "gcp.project_id_mismatch"

	YouAlreadyOwnThisBucketErrorMessage = "You already own this bucket. Please select another name."
	InvalidBucketNameErrorMessage       = "Sorry, that name is not available. Please try a different one."
)

func IsGCPError(err error) bool {
	_, ok := errors.CauseOrSelf(err).(*googleapi.Error)
	return ok
}

func IsErrCode(err error, errorCode int, errorMessage *string) bool {
	gcpError, ok := errors.CauseOrSelf(err).(*googleapi.Error)
	if !ok {
		return false
	}
	if gcpError.Code == errorCode {
		if errorMessage != nil && gcpError.Message == *errorMessage {
			return true
		}
		if errorMessage != nil && gcpError.Message != *errorMessage {
			return false
		}
		return true
	}
	return false
}

func DoesBucketAlreadyExistError(err error) bool {
	return IsErrCode(err, 409, pointer.String(YouAlreadyOwnThisBucketErrorMessage))
}

func IsInvalidBucketNameError(err error) bool {
	return IsErrCode(err, 409, pointer.String(InvalidBucketNameErrorMessage))
}

func ErrorCredentialsFileEnvVarNotSet() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCredentialsFileEnvVarNotSet,
		Message: "please set the path to your credentials file via `export GOOGLE_APPLICATION_CREDENTIALS=<path>`",
	})
}

func ErrorProjectIDMismatch(credsFileProject string, providedProject string, credsFilePath string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrProjectIDMismatch,
		Message: fmt.Sprintf("the \"%s\" project was specified in your configuration, but the credentials file at %s (specified via $GOOGLE_APPLICATION_CREDENTIALS) is connected the the project named \"%s\"", providedProject, credsFilePath, credsFileProject),
	})
}
