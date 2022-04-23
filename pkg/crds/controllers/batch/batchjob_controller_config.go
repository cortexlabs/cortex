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

package batchcontrollers

import (
	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
)

// BatchJobReconcilerConfig reconciler config for the BatchJob kind. Allows for mocking specific methods
type BatchJobReconcilerConfig struct {
	GetTotalBatchCount func(r *BatchJobReconciler, batchJob batch.BatchJob) (int, error)
	GetMetrics         func(r *BatchJobReconciler, batchJob batch.BatchJob) (metrics.BatchMetrics, error)
	SaveJobMetrics     func(r *BatchJobReconciler, batchJob batch.BatchJob) error
	SaveJobStatus      func(r *BatchJobReconciler, batchJob batch.BatchJob) error
}

// ApplyDefaults sets the defaults for BatchJobReconcilerConfig
func (c BatchJobReconcilerConfig) ApplyDefaults() BatchJobReconcilerConfig {
	if c.GetTotalBatchCount == nil {
		c.GetTotalBatchCount = getTotalBatchCount
	}

	if c.GetMetrics == nil {
		c.GetMetrics = getMetrics
	}

	if c.SaveJobMetrics == nil {
		c.SaveJobMetrics = saveJobMetrics
	}

	if c.SaveJobStatus == nil {
		c.SaveJobStatus = saveJobStatus
	}

	return c
}
