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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/consts"
	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/crds/controllers"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/yaml"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	_enqueuerContainerName  = "enqueuer"
	_deadlineExceededReason = "DeadlineExceeded"
)

type batchJobStatusInfo struct {
	QueueExists     bool
	EnqueuingStatus batch.EnqueuingStatus
	EnqueuerJob     *kbatch.Job
	WorkerJob       *kbatch.Job
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
			return false, err
		}
	}
	return true, nil
}

func (r *BatchJobReconciler) createQueue(batchJob batch.BatchJob) (string, error) {
	queueName := r.getQueueName(batchJob)

	tags := map[string]string{
		clusterconfig.ClusterNameTag: r.ClusterConfig.ClusterName,
		"apiName":                    batchJob.Spec.APIName,
		"apiID":                      batchJob.Spec.APIId,
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
	apiSpec, err := r.getAPISpec(batchJob)
	if err != nil {
		return errors.Wrap(err, "failed to get API spec")
	}

	if err = r.uploadJobSpec(batchJob, *apiSpec, queueURL); err != nil {
		return errors.Wrap(err, "failed to upload job spec")
	}

	workerJob, err := r.desiredWorkerJob(batchJob, *apiSpec)
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
			PodSpec: k8s.PodSpec{
				Labels: map[string]string{
					"apiKind":        userconfig.BatchAPIKind.String(),
					"apiName":        batchJob.Spec.APIName,
					"apiID":          batchJob.Spec.APIId,
					"jobID":          batchJob.Name,
					"cortex.dev/api": "true",
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
								"-cluster", r.ClusterConfig.ClusterName,
								"-region", r.ClusterConfig.Region,
								"-bucket", r.ClusterConfig.Bucket,
								"-queue", queueURL,
								"-apiName", batchJob.Spec.APIName,
								"-jobID", batchJob.Name,
							},
							EnvFrom:         controllers.BaseEnvVars(),
							ImagePullPolicy: kcore.PullAlways,
						},
					},
					NodeSelector: controllers.NodeSelectors(),
					Tolerations:  controllers.GenerateResourceTolerations(),
					Affinity: &kcore.Affinity{
						NodeAffinity: &kcore.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: controllers.GeneratePreferredNodeAffinities(),
						},
					},
					ServiceAccountName: controllers.ServiceAccountName,
				},
			},
		},
	)

	if err := ctrl.SetControllerReference(&batchJob, job, r.Scheme); err != nil {
		return nil, err
	}

	return job, nil
}

func (r *BatchJobReconciler) desiredWorkerJob(batchJob batch.BatchJob, apiSpec spec.API) (*kbatch.Job, error) {
	var containers []kcore.Container
	var volumes []kcore.Volume

	switch apiSpec.Predictor.Type {
	case userconfig.PythonPredictorType:
		containers, volumes = controllers.PythonPredictorContainers(&apiSpec)
	case userconfig.TensorFlowPredictorType:
		panic("not implemented!") // FIXME: implement
	default:
		return nil, fmt.Errorf("unexpected predictor type (%s)", apiSpec.Predictor.Type)
	}

	for i, container := range containers {
		if container.Name == controllers.APIContainerName {
			containers[i].Env = append(container.Env, kcore.EnvVar{
				Name:  "CORTEX_JOB_SPEC",
				Value: "s3://" + r.ClusterConfig.Bucket + "/" + r.jobSpecKey(batchJob),
			})
		}
	}

	job := k8s.Job(
		&k8s.JobSpec{
			Name:        batchJob.Spec.APIName + "-" + batchJob.Name,
			Namespace:   batchJob.Namespace,
			Parallelism: batchJob.Spec.Workers,
			PodSpec: k8s.PodSpec{
				Labels: map[string]string{
					"apiKind":        userconfig.BatchAPIKind.String(),
					"apiName":        batchJob.Spec.APIName,
					"apiID":          batchJob.Spec.APIId,
					"specID":         apiSpec.SpecID,
					"predictorID":    apiSpec.PredictorID,
					"jobID":          batchJob.Name,
					"cortex.dev/api": "true",
				},
				Annotations: map[string]string{
					"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
					"cluster-autoscaler.kubernetes.io/safe-to-evict":   "false",
				},
				K8sPodSpec: kcore.PodSpec{
					InitContainers: []kcore.Container{
						controllers.DownloaderContainer(&apiSpec),
					},
					Containers:    containers,
					Volumes:       volumes,
					RestartPolicy: kcore.RestartPolicyNever,
					NodeSelector:  controllers.NodeSelectors(),
					Affinity: &kcore.Affinity{
						NodeAffinity: &kcore.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: controllers.GeneratePreferredNodeAffinities(),
						},
					},
					Tolerations:        controllers.GenerateResourceTolerations(),
					ServiceAccountName: controllers.ServiceAccountName,
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
	apiSpecKey := spec.Key(batchJob.Spec.APIName, batchJob.Spec.APIId, r.ClusterConfig.ClusterName)

	input := s3.GetObjectInput{
		Bucket: aws.String(r.ClusterConfig.Bucket),
		Key:    aws.String(apiSpecKey),
	}
	buff := aws.WriteAtBuffer{}
	if _, err := r.AWS.S3Downloader().Download(&buff, &input); err != nil {
		return nil, err
	}

	apiSpec := &spec.API{}
	if err := json.Unmarshal(buff.Bytes(), apiSpec); err != nil {
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
	}

	worker := statusInfo.WorkerJob
	if worker != nil {
		batchJob.Status.StartTime = worker.Status.StartTime    // assign right away, because it's a pointer
		batchJob.Status.EndTime = worker.Status.CompletionTime // assign right away, because it's a pointer

		if worker.Status.Failed == batchJob.Spec.Workers {
			var batchJobStatus status.JobCode
			for _, condition := range worker.Status.Conditions {
				if condition.Reason == _deadlineExceededReason {
					batchJobStatus = status.JobTimedOut
					break
				}
				// TODO: handle known failure conditions (OOM / image pull failure / etc)
			}
			if batchJobStatus != status.JobTimedOut {
				batchJobStatus = status.JobWorkerError
			}
			batchJob.Status.Status = batchJobStatus
		} else if worker.Status.Succeeded == batchJob.Spec.Workers {
			batchJob.Status.Status = status.JobSucceeded
		} else if worker.Status.Active > 0 {
			batchJob.Status.Status = status.JobRunning
		}
		// TODO: update worker counts
	}

	if err := r.Status().Update(ctx, batchJob); err != nil {
		return err
	}

	return nil
}

func (r *BatchJobReconciler) deleteSQSQueue(batchJob batch.BatchJob) error {
	queueURL := r.getQueueURL(batchJob)
	input := sqs.DeleteQueueInput{QueueUrl: aws.String(queueURL)}
	if _, err := r.AWS.SQS().DeleteQueue(&input); err != nil {
		return err
	}

	return nil
}

func (r *BatchJobReconciler) uploadJobSpec(batchJob batch.BatchJob, api spec.API, queueURL string) error {
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
			return err
		}
	}

	maxBatchCount, err := r.getMaxBatchCount(batchJob)
	if err != nil {
		return err
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
			Config:             config,
		},
		APIID:           api.ID,
		SpecID:          api.SpecID,
		PredictorID:     api.PredictorID,
		SQSUrl:          queueURL,
		StartTime:       batchJob.CreationTimestamp.Time,
		TotalBatchCount: maxBatchCount,
	}

	buffer := &bytes.Buffer{}
	if err := json.NewEncoder(buffer).Encode(jobSpec); err != nil {
		return err
	}

	input := s3manager.UploadInput{
		Body:   buffer,
		Bucket: aws.String(r.ClusterConfig.Bucket),
		Key:    aws.String(r.jobSpecKey(batchJob)),
	}
	if _, err := r.AWS.S3Uploader().Upload(&input); err != nil {
		return err
	}
	return nil
}

func (r *BatchJobReconciler) getMaxBatchCount(batchJob batch.BatchJob) (int, error) {
	key := spec.JobBatchCountKey(r.ClusterConfig.ClusterName, userconfig.BatchAPIKind, batchJob.Spec.APIName, batchJob.Name)
	maxBatchCountBytes, err := r.AWS.ReadBytesFromS3(r.ClusterConfig.Bucket, key)
	if err != nil {
		return 0, err
	}

	maxBatchCount, err := strconv.Atoi(string(maxBatchCountBytes))
	if err != nil {
		return 0, err
	}

	return maxBatchCount, nil
}

func (r *BatchJobReconciler) jobSpecKey(batchJob batch.BatchJob) string {
	// e.g. <cluster name>/jobs/<job_api_kind>/<cortex version>/<api_name>/<job_id>/spec.json
	return filepath.Join(
		r.ClusterConfig.ClusterName,
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
