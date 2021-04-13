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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	batch "github.com/cortexlabs/cortex/operator/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/operator/controllers"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	_enqueuerContainerName = "enqueuer"
)

type batchJobStatusInfo struct {
	QueueExists     bool
	EnqueuingStatus EnqueuingStatus
	WorkerJob       *kbatch.Job
}

// EnqueuingStatus is an enum for the different possible enqueuing status
type EnqueuingStatus string

// Possible EnqueuingStatus states
const (
	EnqueuingNotStarted EnqueuingStatus = "not_started"
	EnqueuingInProgress EnqueuingStatus = "in_progress"
	EnqueuingDone       EnqueuingStatus = "done"
	EnqueuingFailed     EnqueuingStatus = "failed"
)

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

func (r *BatchJobReconciler) checkEnqueueingStatus(ctx context.Context, batchJob batch.BatchJob, workerJob *kbatch.Job) (EnqueuingStatus, error) {
	if workerJob != nil {
		return EnqueuingDone, nil
	}

	var enqueuerJob kbatch.Job
	if err := r.Get(ctx,
		client.ObjectKey{
			Namespace: batchJob.Namespace,
			Name:      batchJob.Spec.APIName + "-" + batchJob.Name + "-enqueuer",
		},
		&enqueuerJob,
	); err != nil {
		if kerrors.IsNotFound(err) {
			return EnqueuingNotStarted, nil
		}
		return "", err
	}

	enqueuerStatus := enqueuerJob.Status
	switch {
	case enqueuerStatus.Failed > 0:
		return EnqueuingFailed, nil
	case enqueuerStatus.Succeeded > 0:
		return EnqueuingDone, nil
	case enqueuerStatus.Active > 0:
		return EnqueuingInProgress, nil
	}

	return EnqueuingInProgress, nil
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

func (r *BatchJobReconciler) createWorkerJob(ctx context.Context, batchJob batch.BatchJob) error {
	workerJob, err := r.desiredWorkerJob(batchJob)
	if err != nil {
		return err
	}

	if err = r.Create(ctx, workerJob); err != nil {
		return err
	}

	return nil
}

func (r *BatchJobReconciler) desiredEnqueuerJob(batchJob batch.BatchJob, queueURL string) (*kbatch.Job, error) {
	job := k8s.Job(
		&k8s.JobSpec{
			Name:      batchJob.Spec.APIName + "-" + batchJob.Name + "-enqueuer",
			Namespace: batchJob.Namespace,
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
							Name:            _enqueuerContainerName,
							Image:           r.ClusterConfig.ImageEnqueuer,
							Args:            []string{"-queue", queueURL, "-apiName", batchJob.Spec.APIName, "-jobID", batchJob.Name},
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

func (r *BatchJobReconciler) desiredWorkerJob(batchJob batch.BatchJob) (*kbatch.Job, error) {
	apiSpec, err := r.getAPISpec(batchJob)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get API spec")
	}

	var containers []kcore.Container
	var volumes []kcore.Volume

	switch apiSpec.Predictor.Type {
	case userconfig.PythonPredictorType:
		containers, volumes = controllers.PythonPredictorContainers(apiSpec)
	case userconfig.TensorFlowPredictorType:
		panic("not implemented!") // FIXME: implement
	default:
		return nil, fmt.Errorf("unexpected predictor type (%s)", apiSpec.Predictor.Type)
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
						controllers.DownloaderContainer(apiSpec),
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

	if err = ctrl.SetControllerReference(&batchJob, job, r.Scheme); err != nil {
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

	// TODO replace strings with enum
	switch statusInfo.EnqueuingStatus {
	case EnqueuingInProgress:
		batchJob.Status.Status = "enqueing"
	case EnqueuingFailed:
		batchJob.Status.Status = "enqueing_failed"
	}

	worker := statusInfo.WorkerJob
	if worker != nil {
		batchJob.Status.StartTime = worker.Status.StartTime    // assign right away, because it's a pointer
		batchJob.Status.EndTime = worker.Status.CompletionTime // assign right away, because it's a pointer

		if worker.Status.Failed == batchJob.Spec.Workers {
			batchJob.Status.Status = "failed"
		} else if worker.Status.Succeeded == batchJob.Spec.Workers {
			batchJob.Status.Status = "completed"
		} else if worker.Status.Active > 0 {
			batchJob.Status.Status = "in_progress"
		}
		// TODO: update worker counts
	}

	if err := r.Status().Update(ctx, batchJob); err != nil {
		return err
	}

	return nil
}
