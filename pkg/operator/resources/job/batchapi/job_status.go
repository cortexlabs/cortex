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
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/yaml"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetJobStatus(jobKey spec.JobKey) (*status.BatchJobStatus, error) {
	ctx := context.Background()
	var batchJob batch.BatchJob
	err := config.K8s.Get(ctx, client.ObjectKey{Name: jobKey.ID, Namespace: config.K8s.Namespace}, &batchJob)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		return getJobStatusFromBatchJob(batchJob)
	}

	jobState, err := job.GetJobState(jobKey)
	if err != nil {
		return nil, err
	}

	return getJobStatusFromJobState(jobState)
}

func getJobStatusFromBatchJob(batchJob batch.BatchJob) (*status.BatchJobStatus, error) {
	jobKey := spec.JobKey{
		ID:      batchJob.Name,
		APIName: batchJob.Spec.APIName,
		Kind:    userconfig.BatchAPIKind,
	}

	var deadLetterQueue *spec.SQSDeadLetterQueue
	if batchJob.Spec.DeadLetterQueue != nil {
		deadLetterQueue = &spec.SQSDeadLetterQueue{
			ARN:             batchJob.Spec.DeadLetterQueue.ARN,
			MaxReceiveCount: int(batchJob.Spec.DeadLetterQueue.MaxReceiveCount),
		}
	}

	var jobConfig map[string]interface{}
	if batchJob.Spec.Config != nil {
		if err := yaml.Unmarshal([]byte(*batchJob.Spec.Config), &jobConfig); err != nil {
			return nil, err
		}
	}

	var timeout *int
	if batchJob.Spec.Timeout != nil {
		timeout = pointer.Int(int(batchJob.Spec.Timeout.Seconds()))
	}

	jobStatus := status.BatchJobStatus{
		BatchJob: spec.BatchJob{
			JobKey: jobKey,
			RuntimeBatchJobConfig: spec.RuntimeBatchJobConfig{
				Workers:            int(batchJob.Spec.Workers),
				SQSDeadLetterQueue: deadLetterQueue,
				Config:             jobConfig,
				Timeout:            timeout,
			},
			APIID:           batchJob.Spec.APIID,
			StartTime:       batchJob.CreationTimestamp.Time,
			SQSUrl:          batchJob.Status.QueueURL,
			TotalBatchCount: batchJob.Status.TotalBatchCount,
		},
		WorkerCounts: batchJob.Status.WorkerCounts,
		Status:       batchJob.Status.Status,
	}

	if batchJob.Status.EndTime != nil {
		jobStatus.EndTime = &batchJob.Status.EndTime.Time
	}

	queueMetrics, err := getQueueMetrics(jobKey)
	if aws.IsNonExistentQueueErr(err) {
		jobStatus.BatchesInQueue = 0
		jobStatus.TotalBatchCount = 0
	} else if err != nil {
		return nil, err
	} else {
		jobStatus.BatchesInQueue = queueMetrics.TotalUserMessages()

		if batchJob.Status.Status == status.JobEnqueuing {
			jobStatus.TotalBatchCount = queueMetrics.TotalUserMessages()
		}
	}

	jobMetrics, err := batch.GetMetrics(config.Prometheus, jobKey, time.Now())
	if err != nil {
		return nil, err
	}

	jobStatus.BatchMetrics = &jobMetrics
	jobStatus.WorkerCounts = batchJob.Status.WorkerCounts

	return &jobStatus, nil
}

func getJobStatusFromJobState(jobState *job.State) (*status.BatchJobStatus, error) {
	jobKey := jobState.JobKey

	jobSpec, err := operator.DownloadBatchJobSpec(jobKey)
	if err != nil {
		return nil, err
	}

	jobStatus := status.BatchJobStatus{
		BatchJob: *jobSpec,
		EndTime:  jobState.EndTime,
		Status:   jobState.Status,
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

func readMetricsFromS3(jobKey spec.JobKey) (metrics.BatchMetrics, error) {
	s3Key := spec.JobMetricsKey(config.ClusterConfig.ClusterUID, userconfig.BatchAPIKind, jobKey.APIName, jobKey.ID)
	batchMetrics := metrics.BatchMetrics{}
	err := config.AWS.ReadJSONFromS3(&batchMetrics, config.ClusterConfig.Bucket, s3Key)
	if err != nil {
		return batchMetrics, err
	}
	return batchMetrics, nil
}
