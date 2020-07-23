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
	"io"

	"github.com/aws/aws-sdk-go/service/s3"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func validateS3ListerDryRun(s3Lister *awslib.S3Lister, response io.Writer) error {
	filesFound := 0
	for _, s3Path := range s3Lister.S3Paths {
		if !awslib.IsValidS3Path(s3Path) {
			return awslib.ErrorInvalidS3Path(s3Path)
		}

		err := config.AWS.S3IteratorFromLister(*s3Lister, func(bucket string, s3Obj *s3.Object) (bool, error) {
			filesFound++
			filePath := awslib.S3Path(bucket, *s3Obj.Key)
			_, err := io.WriteString(response, fmt.Sprintf("(dryrun) found: %s\n", filePath))
			if err != nil {
				return false, err
			}
			return true, nil
		})

		if err != nil {
			return errors.Wrap(err, s3Path)
		}
	}

	if filesFound == 0 {
		return ErrorNoS3FilesFound()
	}

	return nil
}

func validateS3Lister(s3Lister *awslib.S3Lister) error {
	filesFound := 0
	for _, s3Path := range s3Lister.S3Paths {
		if !awslib.IsValidS3Path(s3Path) {
			return awslib.ErrorInvalidS3Path(s3Path)
		}

		err := config.AWS.S3IteratorFromLister(*s3Lister, func(objPath string, s3Obj *s3.Object) (bool, error) {
			filesFound++
			return false, nil
		})

		if err != nil {
			return errors.Wrap(err, s3Path)
		}
	}

	if filesFound == 0 {
		return ErrorNoS3FilesFound()
	}

	return nil
}

func validateSubmission(submission *userconfig.JobSubmission) error {
	err := submission.Validate()
	if err != nil {
		return err
	}

	if submission.DelimitedFiles != nil {
		err := validateS3Lister(&submission.DelimitedFiles.S3Lister)
		if err != nil {
			if errors.GetKind(err) == ErrNoS3FilesFound {
				return errors.Wrap(errors.Append(err, "; you can append `dryRun=true` query param to experiment with s3_file_list criteria without initializing jobs"), userconfig.DelimitedFilesKey)
			}
			return errors.Wrap(err, userconfig.DelimitedFilesKey)
		}
	}

	if submission.FilePathLister != nil {
		err := validateS3Lister(&submission.FilePathLister.S3Lister)
		if err != nil {
			if errors.GetKind(err) == ErrNoS3FilesFound {
				return errors.Wrap(errors.Append(err, "; you can append `dryRun=true` query param to experiment with s3_file_list criteria without initializing jobs"), userconfig.FilePathListerKey)
			}
			return errors.Wrap(err, userconfig.FilePathListerKey)
		}
	}

	return nil
}
