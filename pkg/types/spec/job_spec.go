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

package spec

import (
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type JobKey struct {
	ID      string `json:"job_id"`
	APIName string `json:"api_name"`
}

func (j JobKey) UserString() string {
	return fmt.Sprintf("%s (api %s)", j.ID, j.APIName)
}

// e.g. /jobs/<cortex version>/<api name>/<job id>/spec.json
func (j JobKey) FileSpecKey() string {
	return path.Join(j.PrefixKey(), "spec.json")
}

// e.g. /jobs/<cortex version>/<api_name>/<job_id>
func (j JobKey) PrefixKey() string {
	return path.Join(APIJobPrefix(j.APIName), j.ID)
}

func (j JobKey) K8sName() string {
	return fmt.Sprintf("%s-%s", j.APIName, j.ID)
}

type Job struct {
	JobKey
	userconfig.Job
	APIID           string    `json:"api_id"`
	SQSUrl          string    `json:"sqs_url"`
	TotalBatchCount int       `json:"total_batch_count"`
	Created         time.Time `json:"created_time"`
}

func (j Job) RequestedWorkers() (int, error) {
	if j.Workers != nil {
		return *j.Workers, nil
	}

	if j.BatchesPerWorker != nil && *j.BatchesPerWorker > 0 {
		return j.TotalBatchCount / (*j.BatchesPerWorker), nil
	}

	return 0, errors.ErrorUnexpected(fmt.Sprintf("%s and %s are both not specified", userconfig.WorkersKey, userconfig.BatchesPerWorkerKey))
}

func APIJobPrefix(apiName string) string {
	return filepath.Join("jobs", apiName, consts.CortexVersion)
}
