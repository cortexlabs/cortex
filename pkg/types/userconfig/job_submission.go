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

package userconfig

import (
	"encoding/json"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/gobwas/glob"
)

type Job struct {
	Workers *int                   `json:"workers,omitifempty"`
	Config  map[string]interface{} `json:"config"`
}

type ItemList struct {
	Items     []json.RawMessage `json:"items"`
	BatchSize *int              `json:"batch_size"`
}

type FilePathLister struct {
	aws.S3Lister
	BatchSize *int `json:"batch_size"`
}

type DelimitedFiles struct {
	aws.S3Lister
	BatchSize *int `json:"batch_size"`
}

type JobSubmission struct {
	Job
	ItemList       *ItemList       `json:"item_list"`
	FilePathLister *FilePathLister `json:"file_path_lister"`
	DelimitedFiles *DelimitedFiles `json:"delimited_files"`
}

func (submission *JobSubmission) Validate() error {
	providedKeys := []string{}
	if submission.ItemList != nil {
		providedKeys = append(providedKeys, ItemListKey)
	}
	if submission.FilePathLister != nil {
		providedKeys = append(providedKeys, FilePathListerKey)
	}
	if submission.DelimitedFiles != nil {
		providedKeys = append(providedKeys, DelimitedFilesKey)
	}

	if len(providedKeys) == 0 {
		return ErrorSpecifyExactlyOneKey(ItemListKey, FilePathListerKey, DelimitedFilesKey)
	}

	if len(providedKeys) > 1 {
		return ErrorConflictingFields(providedKeys[0], providedKeys[1:]...)
	}

	if submission.ItemList != nil {
		if len(submission.ItemList.Items) == 0 {
			return errors.Wrap(cr.ErrorTooFewElements(1), ItemsKey)
		}

		for i, batch := range submission.ItemList.Items {
			if len(batch) > aws.MessageSizeLimit {
				return ErrorItemSizeExceedsLimit(i, len(batch), aws.MessageSizeLimit)
			}
		}

		if submission.ItemList.BatchSize == nil {
			submission.ItemList.BatchSize = pointer.Int(1)
		} else if *submission.ItemList.BatchSize < 1 {
			return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(*submission.ItemList.BatchSize, 1), BatcheSizeKey)
		}
	}

	if submission.FilePathLister != nil {
		err := validateS3Lister(&submission.DelimitedFiles.S3Lister)
		if err != nil {
			return errors.Wrap(err, FilePathListerKey)
		}
	}

	if submission.DelimitedFiles != nil {
		err := validateS3Lister(&submission.DelimitedFiles.S3Lister)
		if err != nil {
			return errors.Wrap(err, DelimitedFilesKey)
		}
	}

	if submission.Workers != nil && *submission.Workers <= 0 {
		return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(*submission.Workers, 1), WorkersKey)
	}

	return nil
}

func validateS3Lister(s3Lister *aws.S3Lister) error {
	if len(s3Lister.S3Paths) == 0 {
		return errors.Wrap(cr.ErrorTooFewElements(0), S3PathsKey)
	}

	for _, globPattern := range s3Lister.Includes {
		_, err := glob.Compile(globPattern, '/')
		if err != nil {
			return errors.Wrap(err, "invalid glob pattern", IncludesKey, globPattern)
		}
	}

	for _, globPattern := range s3Lister.Excludes {
		_, err := glob.Compile(globPattern, '/')
		if err != nil {
			return errors.Wrap(err, "invalid glob pattern", ExcludesKey, globPattern)
		}
	}

	return nil
}
