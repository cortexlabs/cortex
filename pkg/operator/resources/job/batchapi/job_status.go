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

package batchapi

import (
	"context"
	"net/url"
	"time"

	"github.com/PEAT-AI/yaml"
	"github.com/cortexlabs/cortex/pkg/config"
	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetJob(jobKey spec.JobKey) (*schema.BatchJobResponse, error) {
	ctx := context.Background()
	var batchJob batch.BatchJob

	err := config.K8s.Get(ctx, client.ObjectKey{Name: jobKey.ID, Namespace: config.K8s.Namespace}, &batchJob)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, err
	}

	if kerrors.IsNotFound(err) {
		return getJobFromS3(jobKey)
	}

	return getJobFromCluster(batchJob)
}

func getJobFromS3(jobKey spec.JobKey) (*schema.BatchJobResponse, error) {
	jobState, err := job.GetJobState(jobKey)
	if err != nil {
		return nil, err
	}

	jobStatus, err := getJobStatusFromJobState(jobState)
	if err != nil {
		return nil, err
	}

	var jobMetrics *metrics.BatchMetrics
	if _, ok := jobState.LastUpdatedMap[spec.MetricsFileKey]; ok && jobState.Status.IsCompleted() {
		jobMetrics, err = readMetricsFromS3(jobState.JobKey)
		if err != nil {
			telemetry.Error(err)
		}
	}

	if jobMetrics == nil {
		// try to get metrics from prometheus if they aren't available in S3 because there might be a delay
		jobMetrics, err = batch.GetMetrics(config.Prometheus, jobStatus.JobKey, time.Now())
		if err != nil {
			telemetry.Error(err)
		}
	}

	apiSpec, err := operator.DownloadAPISpec(jobStatus.APIName, jobStatus.APIID)
	if err != nil {
		return nil, err
	}

	endpoint, err := getJobEndpoint(apiSpec, jobKey)
	if err != nil {
		return nil, err
	}

	return &schema.BatchJobResponse{
		APISpec:   *apiSpec,
		JobStatus: *jobStatus,
		Metrics:   jobMetrics,
		Endpoint:  endpoint,
	}, nil
}

func getJobFromCluster(batchJob batch.BatchJob) (*schema.BatchJobResponse, error) {
	jobStatus, err := getJobStatusFromBatchJob(batchJob)
	if err != nil {
		return nil, err
	}

	jobMetrics, err := batch.GetMetrics(config.Prometheus, jobStatus.JobKey, time.Now())
	if err != nil {
		telemetry.Error(err)
	}

	apiSpec, err := operator.DownloadAPISpec(jobStatus.APIName, jobStatus.APIID)
	if err != nil {
		return nil, err
	}

	endpoint, err := getJobEndpoint(apiSpec, jobStatus.JobKey)
	if err != nil {
		return nil, err
	}

	return &schema.BatchJobResponse{
		APISpec:   *apiSpec,
		JobStatus: *jobStatus,
		Metrics:   jobMetrics,
		Endpoint:  endpoint,
	}, nil
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

	return &jobStatus, nil
}

func readMetricsFromS3(jobKey spec.JobKey) (*metrics.BatchMetrics, error) {
	s3Key := spec.JobMetricsKey(config.ClusterConfig.ClusterUID, userconfig.BatchAPIKind, jobKey.APIName, jobKey.ID)
	batchMetrics := metrics.BatchMetrics{}
	err := config.AWS.ReadJSONFromS3(&batchMetrics, config.ClusterConfig.Bucket, s3Key)
	if err != nil {
		return nil, err
	}
	return &batchMetrics, nil
}

func getJobEndpoint(apiSpec *spec.API, jobKey spec.JobKey) (string, error) {
	endpoint, err := operator.APIEndpoint(apiSpec)
	if err != nil {
		return "", err
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	q := parsedURL.Query()
	q.Add("jobID", jobKey.ID)
	parsedURL.RawQuery = q.Encode()

	return parsedURL.String(), nil
}
