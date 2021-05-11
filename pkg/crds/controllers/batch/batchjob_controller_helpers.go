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

package batchcontrollers

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/consts"
	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	"github.com/cortexlabs/yaml"
	cache "github.com/patrickmn/go-cache"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	_enqueuerContainerName  = "enqueuer"
	_deadlineExceededReason = "DeadlineExceeded"
	_cacheDuration          = 60 * time.Second
)

var totalBatchCountCache *cache.Cache

func init() {
	totalBatchCountCache = cache.New(_cacheDuration, _cacheDuration)
}

type batchJobStatusInfo struct {
	QueueExists     bool
	EnqueuingStatus batch.EnqueuingStatus
	EnqueuerJob     *kbatch.Job
	WorkerJob       *kbatch.Job
	TotalBatchCount int
}

func (r *BatchJobReconciler) checkIfQueueExists(batchJob batch.BatchJob) (bool, error) {
	queueName := r.getQueueName(batchJob)
	input := &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	}
	_, err := r.AWS.SQS().GetQueueUrl(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == sqs.ErrCodeQueueDoesNotExist {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

func (r *BatchJobReconciler) createQueue(batchJob batch.BatchJob) (string, error) {
	queueName := r.getQueueName(batchJob)

	tags := map[string]string{
		clusterconfig.ClusterNameTag: r.ClusterConfig.ClusterName,
		"apiName":                    batchJob.Spec.APIName,
		"apiID":                      batchJob.Spec.APIID,
		"jobID":                      batchJob.Name,
	}

	attributes := map[string]string{
		sqs.QueueAttributeNameFifoQueue:         "true",
		sqs.QueueAttributeNameVisibilityTimeout: "60",
	}

	if batchJob.Spec.DeadLetterQueue != nil {
		redrivePolicy := map[string]string{
			"deadLetterTargetArn": batchJob.Spec.DeadLetterQueue.ARN,
			"maxReceiveCount":     s.Int32(batchJob.Spec.DeadLetterQueue.MaxReceiveCount),
		}

		redrivePolicyJSONBytes, err := libjson.Marshal(redrivePolicy)
		if err != nil {
			return "", err
		}

		attributes[sqs.QueueAttributeNameRedrivePolicy] = string(redrivePolicyJSONBytes)
	}

	input := &sqs.CreateQueueInput{
		Attributes: aws.StringMap(attributes),
		QueueName:  aws.String(queueName),
		Tags:       aws.StringMap(tags),
	}
	output, err := r.AWS.SQS().CreateQueue(input)
	if err != nil {
		return "", err
	}

	return *output.QueueUrl, nil
}

func (r *BatchJobReconciler) getQueueURL(batchJob batch.BatchJob) string {
	// e.g. https://sqs.<region>.amazonaws.com/<account_id>/<queue_name>
	return fmt.Sprintf(
		"https://sqs.%s.amazonaws.com/%s/%s",
		r.ClusterConfig.Region, r.ClusterConfig.AccountID, r.getQueueName(batchJob),
	)
}

func (r *BatchJobReconciler) getQueueName(batchJob batch.BatchJob) string {
	// cx_<hash of cluster name>_b_<api_name>_<job_id>.fifo
	return clusterconfig.SQSNamePrefix(r.ClusterConfig.ClusterName) + "b" +
		clusterconfig.SQSQueueDelimiter + batchJob.Spec.APIName +
		clusterconfig.SQSQueueDelimiter + batchJob.Name + ".fifo"
}

func (r *BatchJobReconciler) checkEnqueuingStatus(ctx context.Context, batchJob batch.BatchJob) (*kbatch.Job, batch.EnqueuingStatus, error) {
	var enqueuerJob kbatch.Job
	if err := r.Get(ctx,
		client.ObjectKey{
			Namespace: batchJob.Namespace,
			Name:      batchJob.Spec.APIName + "-" + batchJob.Name + "-enqueuer",
		},
		&enqueuerJob,
	); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, batch.EnqueuingNotStarted, nil
		}
		return nil, "", err
	}

	enqueuerStatus := enqueuerJob.Status
	switch {
	case enqueuerStatus.Failed > 0:
		return &enqueuerJob, batch.EnqueuingFailed, nil
	case enqueuerStatus.Succeeded > 0:
		return &enqueuerJob, batch.EnqueuingDone, nil
	case enqueuerStatus.Active > 0:
		return &enqueuerJob, batch.EnqueuingInProgress, nil
	}

	return &enqueuerJob, batch.EnqueuingInProgress, nil
}

func (r *BatchJobReconciler) enqueuePayload(ctx context.Context, batchJob batch.BatchJob, queueURL string) error {
	enqueuerJob, err := r.desiredEnqueuerJob(batchJob, queueURL)
	if err != nil {
		return err
	}

	if err = r.Create(ctx, enqueuerJob); err != nil {
		return err
	}

	return nil
}

func (r *BatchJobReconciler) createWorkerJob(ctx context.Context, batchJob batch.BatchJob, queueURL string) error {
	apiSpec, err := r.getAPISpec(batchJob) // TODO: should be cached
	if err != nil {
		return errors.Wrap(err, "failed to get API spec")
	}

	jobSpec, err := r.uploadJobSpec(batchJob, *apiSpec, queueURL)
	if err != nil {
		return errors.Wrap(err, "failed to upload job spec")
	}

	workerJob, err := r.desiredWorkerJob(batchJob, *apiSpec, *jobSpec)
	if err != nil {
		return errors.Wrap(err, "failed to get desired worker job")
	}

	if err = r.Create(ctx, workerJob); err != nil {
		return err
	}

	return nil
}

func (r *BatchJobReconciler) desiredEnqueuerJob(batchJob batch.BatchJob, queueURL string) (*kbatch.Job, error) {
	job := k8s.Job(
		&k8s.JobSpec{
			Name:        batchJob.Spec.APIName + "-" + batchJob.Name + "-enqueuer",
			Namespace:   batchJob.Namespace,
			Parallelism: 1,
			Labels: map[string]string{
				"apiKind":          userconfig.BatchAPIKind.String(),
				"apiName":          batchJob.Spec.APIName,
				"apiID":            batchJob.Spec.APIID,
				"jobID":            batchJob.Name,
				"cortex.dev/api":   "true",
				"cortex.dev/batch": "enqueuer",
			},
			PodSpec: k8s.PodSpec{
				Labels: map[string]string{
					"apiKind":          userconfig.BatchAPIKind.String(),
					"apiName":          batchJob.Spec.APIName,
					"apiID":            batchJob.Spec.APIID,
					"jobID":            batchJob.Name,
					"cortex.dev/api":   "true",
					"cortex.dev/batch": "enqueuer",
				},
				Annotations: map[string]string{
					"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
					"cluster-autoscaler.kubernetes.io/safe-to-evict":   "false",
				},
				K8sPodSpec: kcore.PodSpec{
					RestartPolicy: kcore.RestartPolicyNever,
					Containers: []kcore.Container{
						{
							Name:  _enqueuerContainerName,
							Image: r.ClusterConfig.ImageEnqueuer,
							Args: []string{
								"-cluster-uid", r.ClusterConfig.ClusterUID,
								"-region", r.ClusterConfig.Region,
								"-bucket", r.ClusterConfig.Bucket,
								"-queue", queueURL,
								"-apiName", batchJob.Spec.APIName,
								"-jobID", batchJob.Name,
							},
							ImagePullPolicy: kcore.PullAlways,
						},
					},
					NodeSelector:       workloads.NodeSelectors(),
					Tolerations:        workloads.GenerateResourceTolerations(),
					Affinity:           workloads.GenerateNodeAffinities(batchJob.Spec.NodeGroups),
					ServiceAccountName: workloads.ServiceAccountName,
				},
			},
		},
	)

	if err := ctrl.SetControllerReference(&batchJob, job, r.Scheme); err != nil {
		return nil, err
	}

	return job, nil
}

func (r *BatchJobReconciler) desiredWorkerJob(batchJob batch.BatchJob, apiSpec spec.API, jobSpec spec.BatchJob) (*kbatch.Job, error) {
	var containers []kcore.Container
	var volumes []kcore.Volume

	switch apiSpec.Handler.Type {
	case userconfig.PythonHandlerType:
		containers, volumes = workloads.PythonHandlerJobContainers(&apiSpec)
	case userconfig.TensorFlowHandlerType:
		containers, volumes = workloads.TensorFlowHandlerJobContainers(&apiSpec)
	default:
		return nil, fmt.Errorf("unexpected handler type (%s)", apiSpec.Handler.Type)
	}

	for i, container := range containers {
		if container.Name == workloads.APIContainerName {
			containers[i].Env = append(container.Env, kcore.EnvVar{
				Name:  "CORTEX_JOB_SPEC",
				Value: workloads.BatchSpecPath,
			})
		}
	}

	job := k8s.Job(
		&k8s.JobSpec{
			Name:        batchJob.Spec.APIName + "-" + batchJob.Name,
			Namespace:   batchJob.Namespace,
			Parallelism: batchJob.Spec.Workers,
			Labels: map[string]string{
				"apiKind":          userconfig.BatchAPIKind.String(),
				"apiName":          batchJob.Spec.APIName,
				"apiID":            batchJob.Spec.APIID,
				"specID":           apiSpec.SpecID,
				"handlerID":        apiSpec.HandlerID,
				"jobID":            batchJob.Name,
				"cortex.dev/api":   "true",
				"cortex.dev/batch": "worker",
			},
			PodSpec: k8s.PodSpec{
				Labels: map[string]string{
					"apiKind":          userconfig.BatchAPIKind.String(),
					"apiName":          batchJob.Spec.APIName,
					"apiID":            batchJob.Spec.APIID,
					"specID":           apiSpec.SpecID,
					"handlerID":        apiSpec.HandlerID,
					"jobID":            batchJob.Name,
					"cortex.dev/api":   "true",
					"cortex.dev/batch": "worker",
				},
				Annotations: map[string]string{
					"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
					"cluster-autoscaler.kubernetes.io/safe-to-evict":   "false",
				},
				K8sPodSpec: kcore.PodSpec{
					InitContainers: []kcore.Container{
						workloads.KubexitInitContainer(),
						workloads.BatchInitContainer(&apiSpec, &jobSpec),
					},
					Containers:         containers,
					Volumes:            volumes,
					RestartPolicy:      kcore.RestartPolicyNever,
					NodeSelector:       workloads.NodeSelectors(),
					Affinity:           workloads.GenerateNodeAffinities(batchJob.Spec.NodeGroups),
					Tolerations:        workloads.GenerateResourceTolerations(),
					ServiceAccountName: workloads.ServiceAccountName,
				},
			},
		},
	)

	if batchJob.Spec.Timeout != nil {
		job.Spec.ActiveDeadlineSeconds = pointer.Int64(int64(batchJob.Spec.Timeout.Seconds()))
	}

	if err := ctrl.SetControllerReference(&batchJob, job, r.Scheme); err != nil {
		return nil, err
	}

	return job, nil
}

func (r *BatchJobReconciler) getAPISpec(batchJob batch.BatchJob) (*spec.API, error) {
	apiSpecKey := spec.Key(batchJob.Spec.APIName, batchJob.Spec.APIID, r.ClusterConfig.ClusterUID)

	apiSpecBytes, err := r.AWS.ReadBytesFromS3(r.ClusterConfig.Bucket, apiSpecKey)
	if err != nil {
		return nil, err
	}

	apiSpec := &spec.API{}
	if err := json.Unmarshal(apiSpecBytes, apiSpec); err != nil {
		return nil, err
	}

	return apiSpec, nil
}

func (r *BatchJobReconciler) getWorkerJob(ctx context.Context, batchJob batch.BatchJob) (*kbatch.Job, error) {
	workerName := batchJob.Spec.APIName + "-" + batchJob.Name

	var job kbatch.Job
	err := r.Get(ctx, client.ObjectKey{Namespace: batchJob.Namespace, Name: workerName}, &job)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &job, nil
}

func (r *BatchJobReconciler) updateStatus(ctx context.Context, batchJob *batch.BatchJob, statusInfo batchJobStatusInfo) error {
	batchJob.Status.ID = batchJob.Name

	if statusInfo.QueueExists {
		batchJob.Status.QueueURL = r.getQueueURL(*batchJob)
	}

	switch statusInfo.EnqueuingStatus {
	case batch.EnqueuingNotStarted:
		batchJob.Status.Status = status.JobPending
	case batch.EnqueuingInProgress:
		batchJob.Status.Status = status.JobEnqueuing
	case batch.EnqueuingFailed:
		batchJob.Status.Status = status.JobEnqueueFailed
		batchJob.Status.EndTime = statusInfo.EnqueuerJob.Status.CompletionTime
	case batch.EnqueuingDone:
		batchJob.Status.TotalBatchCount = statusInfo.TotalBatchCount
	}

	worker := statusInfo.WorkerJob
	if worker != nil {
		batchJob.Status.EndTime = worker.Status.CompletionTime // assign right away, because it's a pointer

		if worker.Status.Failed == batchJob.Spec.Workers {
			batchJobStatus := status.JobWorkerError
			for _, condition := range worker.Status.Conditions {
				if condition.Reason == _deadlineExceededReason {
					batchJobStatus = status.JobTimedOut
					break
				}
			}

			isWorkerOOM, err := r.checkWorkersOOM(ctx, batchJob)
			if err != nil {
				return err
			}

			if isWorkerOOM {
				batchJobStatus = status.JobWorkerOOM
			}

			batchJob.Status.Status = batchJobStatus
		} else if worker.Status.Succeeded == batchJob.Spec.Workers {
			batchJob.Status.Status = status.JobSucceeded

			jobMetrics, err := r.Config.GetMetrics(r, *batchJob)
			if err != nil {
				return err
			}

			if jobMetrics.Failed > 0 {
				batchJob.Status.Status = status.JobCompletedWithFailures
			}

		} else if worker.Status.Active > 0 {
			batchJob.Status.Status = status.JobRunning
		}

		batchJob.Status.WorkerCounts = &status.WorkerCounts{
			Running:   worker.Status.Active,
			Succeeded: worker.Status.Succeeded,
			Failed:    worker.Status.Failed,
		}
	}

	if err := r.Status().Update(ctx, batchJob); err != nil {
		return err
	}

	return nil
}

func (r *BatchJobReconciler) checkWorkersOOM(ctx context.Context, batchJob *batch.BatchJob) (bool, error) {
	workerJobPods := kcore.PodList{}
	if err := r.List(ctx, &workerJobPods,
		client.InNamespace(r.ClusterConfig.Namespace),
		client.MatchingLabels{
			"jobID":   batchJob.Name,
			"apiName": batchJob.Spec.APIName,
			"apiID":   batchJob.Spec.APIID,
		},
	); err != nil {
		return false, err
	}

	for i := range workerJobPods.Items {
		if k8s.WasPodOOMKilled(&workerJobPods.Items[i]) {
			return true, nil
		}
	}
	return false, nil
}

func (r *BatchJobReconciler) deleteSQSQueue(batchJob batch.BatchJob) error {
	queueURL := r.getQueueURL(batchJob)
	input := sqs.DeleteQueueInput{QueueUrl: aws.String(queueURL)}
	if _, err := r.AWS.SQS().DeleteQueue(&input); err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == sqs.ErrCodeQueueDoesNotExist {
				return nil
			}
		}
		return err
	}
	return nil
}

func (r *BatchJobReconciler) uploadJobSpec(batchJob batch.BatchJob, api spec.API, queueURL string) (*spec.BatchJob, error) {
	var deadLetterQueue *spec.SQSDeadLetterQueue
	if batchJob.Spec.DeadLetterQueue != nil {
		deadLetterQueue = &spec.SQSDeadLetterQueue{
			ARN:             batchJob.Spec.DeadLetterQueue.ARN,
			MaxReceiveCount: int(batchJob.Spec.DeadLetterQueue.MaxReceiveCount),
		}
	}

	var config map[string]interface{}
	if batchJob.Spec.Config != nil {
		if err := yaml.Unmarshal([]byte(*batchJob.Spec.Config), &config); err != nil {
			return nil, err
		}
	}

	var timeout *int
	if batchJob.Spec.Timeout != nil {
		timeout = pointer.Int(int(batchJob.Spec.Timeout.Seconds()))
	}

	totalBatchCount, err := r.Config.GetTotalBatchCount(r, batchJob)
	if err != nil {
		return nil, err
	}

	jobSpec := spec.BatchJob{
		JobKey: spec.JobKey{
			ID:      batchJob.Name,
			APIName: batchJob.Spec.APIName,
			Kind:    userconfig.BatchAPIKind,
		},
		RuntimeBatchJobConfig: spec.RuntimeBatchJobConfig{
			Workers:            int(batchJob.Spec.Workers),
			SQSDeadLetterQueue: deadLetterQueue,
			Timeout:            timeout,
			Config:             config,
		},
		APIID:           api.ID,
		SQSUrl:          queueURL,
		StartTime:       batchJob.CreationTimestamp.Time,
		TotalBatchCount: totalBatchCount,
	}

	if err = r.AWS.UploadJSONToS3(&jobSpec, r.ClusterConfig.Bucket, r.jobSpecKey(batchJob)); err != nil {
		return nil, err
	}

	return &jobSpec, nil
}

func (r *BatchJobReconciler) jobSpecKey(batchJob batch.BatchJob) string {
	// e.g. <cluster uid>/jobs/<job_api_kind>/<cortex version>/<api_name>/<job_id>/spec.json
	return filepath.Join(
		r.ClusterConfig.ClusterUID,
		"jobs",
		userconfig.BatchAPIKind.String(),
		consts.CortexVersion,
		batchJob.Spec.APIName,
		batchJob.Name,
		"spec.json",
	)
}

func (r *BatchJobReconciler) updateCompletedTimestamp(ctx context.Context, batchJob *batch.BatchJob) error {
	ts := time.Now().Format(time.RFC3339)
	if batchJob.Annotations != nil {
		batchJob.Annotations[_completedTimestampAnnotation] = ts
	} else {
		batchJob.Annotations = map[string]string{
			_completedTimestampAnnotation: ts,
		}
	}
	if err := r.Update(ctx, batchJob); err != nil {
		return err
	}
	return nil
}

func (r *BatchJobReconciler) persistJobToS3(batchJob batch.BatchJob) error {
	return parallel.RunFirstErr(
		func() error {
			if batchJob.Status.Status != status.JobSucceeded {
				return nil
			}
			return r.Config.SaveJobMetrics(r, batchJob)
		},
		func() error {
			return r.Config.SaveJobStatus(r, batchJob)
		},
	)
}

func getTotalBatchCount(r *BatchJobReconciler, batchJob batch.BatchJob) (int, error) {
	key := spec.JobBatchCountKey(r.ClusterConfig.ClusterUID, userconfig.BatchAPIKind, batchJob.Spec.APIName, batchJob.Name)
	cachedTotalBatchCount, found := totalBatchCountCache.Get(key)
	var totalBatchCount int
	if !found {
		totalBatchCountBytes, err := r.AWS.ReadBytesFromS3(r.ClusterConfig.Bucket, key)
		if err != nil {
			return 0, err
		}

		totalBatchCount, err = strconv.Atoi(string(totalBatchCountBytes))
		if err != nil {
			return 0, err
		}
	} else {
		totalBatchCount = cachedTotalBatchCount.(int)
	}

	totalBatchCountCache.Set(key, totalBatchCount, _cacheDuration)

	return totalBatchCount, nil
}

func getMetrics(r *BatchJobReconciler, batchJob batch.BatchJob) (metrics.BatchMetrics, error) {
	jobMetrics, err := batch.GetMetrics(r.Prometheus, spec.JobKey{
		ID:      batchJob.Name,
		APIName: batchJob.Spec.APIName,
		Kind:    userconfig.BatchAPIKind,
	}, batchJob.Status.EndTime.Time)
	if err != nil {
		return metrics.BatchMetrics{}, err
	}
	return jobMetrics, nil
}

func saveJobMetrics(r *BatchJobReconciler, batchJob batch.BatchJob) error {
	jobMetrics, err := r.Config.GetMetrics(r, batchJob)
	if err != nil {
		return err
	}

	key := spec.JobMetricsKey(r.ClusterConfig.ClusterUID, userconfig.BatchAPIKind, batchJob.Spec.APIName, batchJob.Name)
	if err = r.AWS.UploadJSONToS3(&jobMetrics, r.ClusterConfig.Bucket, key); err != nil {
		return err
	}

	return nil
}

func saveJobStatus(r *BatchJobReconciler, batchJob batch.BatchJob) error {
	return parallel.RunFirstErr(
		func() error {
			stoppedStatusKey := filepath.Join(
				spec.JobAPIPrefix(r.ClusterConfig.ClusterUID, userconfig.BatchAPIKind, batchJob.Spec.APIName),
				batchJob.Name,
				status.JobStopped.String(),
			)
			return r.AWS.UploadStringToS3("", r.ClusterConfig.Bucket, stoppedStatusKey)

		},
		func() error {
			jobStatus := batchJob.Status.Status.String()
			key := filepath.Join(
				spec.JobAPIPrefix(r.ClusterConfig.ClusterUID, userconfig.BatchAPIKind, batchJob.Spec.APIName),
				batchJob.Name,
				jobStatus,
			)
			return r.AWS.UploadStringToS3("", r.ClusterConfig.Bucket, key)
		},
	)
}
