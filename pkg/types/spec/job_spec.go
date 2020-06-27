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
	"time"

	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/status"
)

type JobSpec struct {
	ID              string                `json:"job_id"`
	APIID           string                `json:"api_id"`
	APIName         string                `json:"api_name"`
	SQSUrl          string                `json:"sqs_url"`
	Config          interface{}           `json:"config"`
	Parallelism     int                   `json:"parallelism"`
	Status          status.JobCode        `json:"status"`
	TotalPartitions int                   `json:"total_partitions"`
	StartTime       time.Time             `json:"start_time"`
	EndTime         *time.Time            `json:"end_time"`
	LastUpdated     time.Time             `json:"last_updated"`
	Metrics         *metrics.JobMetrics   `json:"metrics"`
	QueueMetrics    *metrics.QueueMetrics `json:"queue_metrics"`
	WorkerStats     *status.WorkerStats   `json:"worker_stats"`
}
