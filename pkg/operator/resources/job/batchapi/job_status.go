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

package batchapi

import (
	"context"
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetJobStatus(jobKey spec.JobKey) (*status.BatchJobStatus, error) {
	jobState, err := job.GetJobState(jobKey)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var batchJob batch.BatchJob
	if err = config.K8s.Get(ctx, client.ObjectKey{Name: jobKey.ID, Namespace: config.K8s.Namespace}, &batchJob); err != nil {
		return nil, err
	}

	return getJobStatusFromJobState(jobState, &batchJob)
}

func getJobStatusFromJobState(jobState *job.State, batchJob *batch.BatchJob) (*status.BatchJobStatus, error) {
	jobKey := jobState.JobKey

	jobSpec, err := operator.DownloadBatchJobSpec(jobKey)
	if err != nil {
		return nil, err
	}

	jobCode := jobState.Status
	if batchJob != nil {
		jobCode = batchJob.Status.Status
	}

	jobStatus := status.BatchJobStatus{
		BatchJob: *jobSpec,
		EndTime:  jobState.EndTime,
		Status:   jobCode,
	}

	if batchJob != nil && jobCode.IsInProgress() {
		queueMetrics, err := getQueueMetrics(jobKey)
		if err != nil {
			return nil, err
		}

		jobStatus.BatchesInQueue = queueMetrics.TotalUserMessages()

		if batchJob.Status.Status == status.JobEnqueuing {
			jobStatus.TotalBatchCount = queueMetrics.TotalUserMessages()
		}

		if batchJob.Status.Status == status.JobRunning {
			jobMetrics, err := batch.GetMetrics(config.Prometheus, jobKey, time.Now())
			if err != nil {
				return nil, err
			}
			jobStatus.BatchMetrics = &jobMetrics
			jobStatus.WorkerCounts = batchJob.Status.WorkerCounts
		}
	}

	if _, ok := jobState.LastUpdatedMap[spec.MetricsFileKey]; ok && jobState.Status.IsCompleted() {
		jobMetrics, err := readMetricsFromS3(jobKey)
		if err != nil {
			return nil, err
		}
		jobStatus.BatchMetrics = &jobMetrics
	}

	return &jobStatus, nil
}

func getJobStatusFromK8sBatchJob(batchJob batch.BatchJob) (*status.BatchJobStatus, error) {
	jobState, err := job.GetJobState(spec.JobKey{
		ID:      batchJob.Name,
		APIName: batchJob.Spec.APIName,
		Kind:    userconfig.BatchAPIKind,
	})
	if err != nil {
		return nil, err
	}

	return getJobStatusFromJobState(jobState, &batchJob)
}

func readMetricsFromS3(jobKey spec.JobKey) (metrics.BatchMetrics, error) {
	s3Key := spec.JobMetricsKey(config.ClusterConfig.ClusterName, userconfig.BatchAPIKind, jobKey.APIName, jobKey.ID)
	batchMetrics := metrics.BatchMetrics{}
	err := config.AWS.ReadJSONFromS3(&batchMetrics, config.ClusterConfig.Bucket, s3Key)
	if err != nil {
		return batchMetrics, err
	}
	return batchMetrics, nil
}
