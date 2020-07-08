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
	"fmt"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
)

const _batchItemSizeLimit = 1024 * 256

type Job struct {
	Workers          *int        `json:"parallelism,omitifempty"`
	BatchesPerWorker *int        `json:"batches_per_worker,omitifempty"`
	JobConfig        interface{} `json:"config"`
}

type JobSubmission struct {
	Job
	Batches   []json.RawMessage `json:"batches"`
	BatchSize *int              `json:"batch_size"`
}

func (submission *JobSubmission) Validate() error {
	if len(submission.Batches) == 0 {
		return errors.Wrap(cr.ErrorTooFewElements(1), BatchesKey)
	}

	for i, batch := range submission.Batches {
		if len(batch) > _batchItemSizeLimit {
			return errors.Wrap(ErrorItemSizeExceedsLimit(len(batch), _batchItemSizeLimit), fmt.Sprintf("element %d", i))
		}
	}

	if submission.BatchSize == nil {
		submission.BatchSize = pointer.Int(1)
	} else if *submission.BatchSize < 1 {
		return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(*submission.BatchSize, 1), WorkersKey)
	}

	if (submission.Workers == nil) == (submission.BatchesPerWorker == nil) {
		return errors.Wrap(ErrorConflictingFields(WorkersKey, BatchesPerWorkerKey))
	}

	if submission.Workers != nil && *submission.Workers <= 0 {
		return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(*submission.Workers, 1), WorkersKey)
	}

	if submission.BatchesPerWorker != nil && *submission.BatchesPerWorker <= 0 {
		return errors.Wrap(cr.ErrorMustBeGreaterThanOrEqualTo(*submission.BatchesPerWorker, 1), BatchesPerWorkerKey)
	}

	return nil
}
