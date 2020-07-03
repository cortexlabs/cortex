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
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type JobID struct {
	ID      string `json:"job_id"`
	APIName string `json:"api_name"`
}

func (j JobID) UserString() string {
	return fmt.Sprintf("%s (api %s)", j.ID, j.APIName)
}

func (j JobID) K8sName() string {
	return fmt.Sprintf("%s-%s", j.APIName, j.ID)
}

type Job struct {
	JobID
	userconfig.JobSpec
	APIID           string    `json:"api_id"`
	SQSUrl          string    `json:"sqs_url"`
	TotalPartitions int       `json:"total_partitions"`
	Created         time.Time `json:"created_time"`
}

func APIJobPrefix(apiName string) string {
	return filepath.Join("jobs", apiName, consts.CortexVersion)
}

func JobSpecPrefix(jobID JobID) string {
	return filepath.Join(APIJobPrefix(jobID.APIName), jobID.ID)
}

func JobSpecKey(jobID JobID) string {
	return filepath.Join(JobSpecPrefix(jobID), "spec.json")
}
