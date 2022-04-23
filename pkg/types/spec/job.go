/*
Copyright 2022 Cortex Labs, Inc.

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

const (
	MetricsFileKey = "metrics.json"
)

type JobKey struct {
	ID      string          `json:"job_id" yaml:"job_id"`
	APIName string          `json:"api_name" yaml:"api_name"`
	Kind    userconfig.Kind `json:"kind" yaml:"kind"`
}

func (j JobKey) UserString() string {
	return fmt.Sprintf("%s (%s api)", j.ID, j.APIName)
}

// e.g. /<cluster UID>/jobs/<job_api_kind>/<cortex version>/<api_name>/<job_id>/spec.json
func (j JobKey) SpecFilePath(clusterUID string) string {
	return path.Join(j.Prefix(clusterUID), "spec.json")
}

// e.g. /<cluster UID>/jobs/<job_api_kind>/<cortex version>/<api_name>/<job_id>
func (j JobKey) Prefix(clusterUID string) string {
	return s.EnsureSuffix(path.Join(JobAPIPrefix(clusterUID, j.Kind, j.APIName), j.ID), "/")
}

func (j JobKey) K8sName() string {
	return fmt.Sprintf("%s-%s", j.APIName, j.ID)
}

type SQSDeadLetterQueue struct {
	ARN             string `json:"arn" yaml:"arn"`
	MaxReceiveCount int    `json:"max_receive_count" yaml:"max_receive_count"`
}

type RuntimeBatchJobConfig struct {
	Workers            int                    `json:"workers" yaml:"workers"`
	SQSDeadLetterQueue *SQSDeadLetterQueue    `json:"sqs_dead_letter_queue" yaml:"sqs_dead_letter_queue"`
	Config             map[string]interface{} `json:"config" yaml:"config"`
	Timeout            *int                   `json:"timeout" yaml:"timeout"`
}

type RuntimeTaskJobConfig struct {
	Workers int                    `json:"workers" yaml:"workers"`
	Config  map[string]interface{} `json:"config" yaml:"config"`
	Timeout *int                   `json:"timeout" yaml:"timeout"`
}

type BatchJob struct {
	JobKey
	RuntimeBatchJobConfig
	APIID           string    `json:"api_id" yaml:"api_id"`
	SQSUrl          string    `json:"sqs_url" yaml:"sqs_url"`
	TotalBatchCount int       `json:"total_batch_count,omitempty" yaml:"total_batch_count,omitempty"`
	StartTime       time.Time `json:"start_time,omitempty" yaml:"start_time,omitempty"`
}

type TaskJob struct {
	JobKey
	RuntimeTaskJobConfig
	APIID     string    `json:"api_id" yaml:"api_id"`
	SpecID    string    `json:"spec_id" yaml:"spec_id"`
	PodID     string    `json:"pod_id" yaml:"pod_id"`
	StartTime time.Time `json:"start_time" yaml:"start_time"`
}

// e.g. /<cluster UID>/jobs/<job_api_kind>/<cortex version>/<api_name>
func JobAPIPrefix(clusterUID string, kind userconfig.Kind, apiName string) string {
	return filepath.Join(clusterUID, "jobs", kind.String(), consts.CortexVersion, apiName)
}

func JobPayloadKey(clusterUID string, kind userconfig.Kind, apiName string, jobID string) string {
	return filepath.Join(JobAPIPrefix(clusterUID, kind, apiName), jobID, "payload.json")
}

func JobBatchCountKey(clusterUID string, kind userconfig.Kind, apiName string, jobID string) string {
	return filepath.Join(JobAPIPrefix(clusterUID, kind, apiName), jobID, "max_batch_count")
}

func JobMetricsKey(clusterUID string, kind userconfig.Kind, apiName string, jobID string) string {
	return filepath.Join(JobAPIPrefix(clusterUID, kind, apiName), jobID, MetricsFileKey)
}
