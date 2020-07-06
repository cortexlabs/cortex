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

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type JobSpec struct {
	Parallelism      *int        `json:"parallelism,omitifempty"`
	BatchesPerWorker *int        `json:"batches_per_worker,omitifempty"`
	JobConfig        interface{} `json:"config"`
}

type JobSubmission struct {
	JobSpec
	Items []json.RawMessage `json:"items"`
}

func (submission *JobSubmission) Validate() error {
	if len(submission.Items) == 0 {
		return errors.Wrap(cr.ErrorTooFewElements(0), ItemsKey)
	}

	if submission.Parallelism != nil && *submission.Parallelism <= 0 {
		return errors.Wrap(cr.ErrorMustBeGreaterThan(*submission.Parallelism, 0), ParallelismKey)
	}

	if submission.BatchesPerWorker != nil && *submission.BatchesPerWorker <= 0 {
		return errors.Wrap(cr.ErrorMustBeGreaterThan(*submission.BatchesPerWorker, 0), BatchesPerWorkerKey)
	}

	if (submission.Parallelism == nil) == (submission.BatchesPerWorker == nil) {
		return errors.Wrap(ErrorConflictingFields(ParallelismKey, BatchesPerWorkerKey))
	}

	return nil
}
