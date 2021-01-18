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

package batchapi

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cortexlabs/cortex/pkg/consts"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/gobwas/glob"
)

func validateJobSubmissionSchema(submission *schema.BatchJobSubmission) error {
	providedKeys := []string{}
	if submission.ItemList != nil {
		providedKeys = append(providedKeys, schema.ItemListKey)
	}
	if submission.FilePathLister != nil {
		providedKeys = append(providedKeys, schema.FilePathListerKey)
	}
	if submission.DelimitedFiles != nil {
		providedKeys = append(providedKeys, schema.DelimitedFilesKey)
	}

	if len(providedKeys) == 0 {
		return job.ErrorSpecifyExactlyOneKey(schema.ItemListKey, schema.FilePathListerKey, schema.DelimitedFilesKey)
	}

	if len(providedKeys) > 1 {
		return job.ErrorConflictingFields(providedKeys[0], providedKeys[1:]...)
	}

	if submission.ItemList != nil {
		if len(submission.ItemList.Items) == 0 {
			return errors.Wrap(cr.ErrorTooFewElements(1), schema.ItemsKey)
		}

		for i, batch := range submission.ItemList.Items {
			if len(batch) > _messageSizeLimit {
				return ErrorItemSizeExceedsLimit(i, len(batch), _messageSizeLimit)
			}
		}

		if submission.ItemList.BatchSize < 1 {
			return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(submission.ItemList.BatchSize, 1), schema.ItemListKey, schema.BatchSizeKey)
		}
	}

	if submission.FilePathLister != nil {
		if submission.FilePathLister.BatchSize < 1 {
			return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(submission.FilePathLister.BatchSize, 1), schema.FilePathListerKey, schema.BatchSizeKey)
		}
	}

	if submission.DelimitedFiles != nil {
		if submission.DelimitedFiles.BatchSize < 1 {
			return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(submission.DelimitedFiles.BatchSize, 1), schema.DelimitedFilesKey, schema.BatchSizeKey)
		}
	}

	if submission.Workers <= 0 {
		return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(submission.Workers, 1), schema.WorkersKey)
	}

	if submission.Timeout != nil && *submission.Timeout <= 0 {
		return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(submission.Timeout, 1), schema.TimeoutKey)
	}

	if submission.SQSDeadLetterQueue != nil {
		if len(submission.SQSDeadLetterQueue.ARN) == 0 {
			return errors.Wrap(cr.ErrorCannotBeEmpty(), schema.SQSDeadLetterQueueKey, schema.ARNKey)
		}
		if submission.SQSDeadLetterQueue.MaxReceiveCount < 1 {
			return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(submission.SQSDeadLetterQueue.MaxReceiveCount, 1), schema.SQSDeadLetterQueueKey, schema.MaxReceiveCountKey)
		}
	}

	return nil
}

func validateJobSubmission(submission *schema.BatchJobSubmission) error {
	err := validateJobSubmissionSchema(submission)
	if err != nil {
		return errors.Append(err, fmt.Sprintf("\n\njob submission schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
	}

	if submission.FilePathLister != nil {
		err := validateS3Lister(&submission.FilePathLister.S3Lister)
		if err != nil {
			return errors.Wrap(err, schema.FilePathListerKey)
		}
	}

	if submission.DelimitedFiles != nil {
		err := validateS3Lister(&submission.DelimitedFiles.S3Lister)
		if err != nil {
			return errors.Wrap(err, schema.DelimitedFilesKey)
		}
	}

	return nil
}

func validateS3Lister(s3Lister *schema.S3Lister) error {
	if len(s3Lister.S3Paths) == 0 {
		return errors.Wrap(cr.ErrorTooFewElements(1), schema.S3PathsKey)
	}

	for _, globPattern := range s3Lister.Includes {
		_, err := glob.Compile(globPattern, '/')
		if err != nil {
			return errors.Wrap(err, schema.IncludesKey, globPattern)
		}
	}

	for _, globPattern := range s3Lister.Excludes {
		_, err := glob.Compile(globPattern, '/')
		if err != nil {
			return errors.Wrap(err, schema.ExcludesKey, globPattern)
		}
	}

	filesFound := 0
	for _, s3Path := range s3Lister.S3Paths {
		if !awslib.IsValidS3Path(s3Path) {
			return awslib.ErrorInvalidS3Path(s3Path)
		}

		err := s3IteratorFromLister(*s3Lister, func(objPath string, s3Obj *s3.Object) (bool, error) {
			filesFound++
			return false, nil
		})
		if err != nil {
			return errors.Wrap(err, s3Path)
		}

		if filesFound > 0 {
			return nil
		}
	}

	return ErrorNoS3FilesFound()
}

func listFilesDryRun(s3Lister *schema.S3Lister) ([]string, error) {
	var s3Files []string
	for _, s3Path := range s3Lister.S3Paths {
		if !awslib.IsValidS3Path(s3Path) {
			return nil, awslib.ErrorInvalidS3Path(s3Path)
		}

		err := s3IteratorFromLister(*s3Lister, func(bucket string, s3Obj *s3.Object) (bool, error) {
			s3Files = append(s3Files, awslib.S3Path(bucket, *s3Obj.Key))
			return true, nil
		})

		if err != nil {
			return nil, errors.Wrap(err, s3Path)
		}
	}

	if len(s3Files) == 0 {
		return nil, ErrorNoS3FilesFound()
	}

	return s3Files, nil
}
