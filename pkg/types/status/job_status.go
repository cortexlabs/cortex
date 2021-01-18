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

package status

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

type BatchJobStatus struct {
	spec.BatchJob
	EndTime        *time.Time            `json:"end_time"`
	Status         JobCode               `json:"status"`
	BatchesInQueue int                   `json:"batches_in_queue"`
	BatchMetrics   *metrics.BatchMetrics `json:"batch_metrics"`
	WorkerCounts   *WorkerCounts         `json:"worker_counts"`
}

type TaskJobStatus struct {
	spec.TaskJob
	EndTime      *time.Time    `json:"end_time"`
	Status       JobCode       `json:"status"`
	WorkerCounts *WorkerCounts `json:"worker_counts"`
}
