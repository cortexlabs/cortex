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
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type JobKey struct {
	ID      string `json:"job_id"`
	APIName string `json:"api_name"`
}

func (j JobKey) UserString() string {
	return fmt.Sprintf("%s (%s api)", j.ID, j.APIName)
}

// e.g. /jobs/<cortex version>/<api_name>/<job_id>/spec.json
func (j JobKey) SpecFilePath() string {
	return path.Join(j.Prefix(), "spec.json")
}

// e.g. /jobs/<cortex version>/<api_name>/<job_id>
func (j JobKey) Prefix() string {
	return s.EnsureSuffix(path.Join(BatchAPIJobPrefix(j.APIName), j.ID), "/")
}

func (j JobKey) K8sName() string {
	return fmt.Sprintf("%s-%s", j.APIName, j.ID)
}

type RuntimeJobConfig struct {
	Workers int                    `json:"workers"`
	Config  map[string]interface{} `json:"config"`
}

type Job struct {
	JobKey
	RuntimeJobConfig
	APIID           string    `json:"api_id"`
	SQSUrl          string    `json:"sqs_url"`
	TotalBatchCount int       `json:"total_batch_count"`
	StartTime       time.Time `json:"start_time"`
}

func BatchAPIJobPrefix(apiName string) string {
	return filepath.Join("jobs", consts.CortexVersion, apiName)
}
