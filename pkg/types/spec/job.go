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

package spec

import (
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type JobKey struct {
	ID      string          `json:"job_id"`
	APIName string          `json:"api_name"`
	Kind    userconfig.Kind `json:"kind"`
}

func (j JobKey) UserString() string {
	return fmt.Sprintf("%s (%s api)", j.ID, j.APIName)
}

// e.g. /<cluster name>/jobs/<job_api_kind>/<cortex version>/<api_name>/<job_id>/spec.json
func (j JobKey) SpecFilePath(clusterName string) string {
	return path.Join(j.Prefix(clusterName), "spec.json")
}

// e.g. /<cluster name>/jobs/<job_api_kind>/<cortex version>/<api_name>/<job_id>
func (j JobKey) Prefix(clusterName string) string {
	return s.EnsureSuffix(path.Join(JobAPIPrefix(clusterName, j.Kind, j.APIName), j.ID), "/")
}

func (j JobKey) K8sName() string {
	return fmt.Sprintf("%s-%s", j.APIName, j.ID)
}

type SQSDeadLetterQueue struct {
	ARN             string `json:"arn"`
	MaxReceiveCount int    `json:"max_receive_count"`
}

type RuntimeBatchJobConfig struct {
	Workers            int                    `json:"workers"`
	SQSDeadLetterQueue *SQSDeadLetterQueue    `json:"sqs_dead_letter_queue"`
	Config             map[string]interface{} `json:"config"`
	Timeout            *int                   `json:"timeout"`
}

type RuntimeTaskJobConfig struct {
	Workers int                    `json:"workers"`
	Config  map[string]interface{} `json:"config"`
	Timeout *int                   `json:"timeout"`
}

type BatchJob struct {
	JobKey
	RuntimeBatchJobConfig
	APIID           string    `json:"api_id"`
	SpecID          string    `json:"spec_id"`
	PredictorID     string    `json:"predictor_id"`
	SQSUrl          string    `json:"sqs_url"`
	TotalBatchCount int       `json:"total_batch_count"`
	StartTime       time.Time `json:"start_time"`
}

type TaskJob struct {
	JobKey
	RuntimeTaskJobConfig
	APIID       string    `json:"api_id"`
	SpecID      string    `json:"spec_id"`
	PredictorID string    `json:"predictor_id"`
	StartTime   time.Time `json:"start_time"`
}

// e.g. /<cluster name>/jobs/<job_api_kind>/<cortex version>/<api_name>
func JobAPIPrefix(clusterName string, kind userconfig.Kind, apiName string) string {
	return filepath.Join(clusterName, "jobs", kind.String(), consts.CortexVersion, apiName)
}
