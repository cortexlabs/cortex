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
	"time"

	"github.com/PEAT-AI/yaml"
	"github.com/cortexlabs/cortex/pkg/config"
	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const _batchJobTTL = 40 * time.Second // Double the duration of the statsd pod monitor

func DryRun(submission *schema.BatchJobSubmission) ([]string, error) {
	err := validateJobSubmission(submission)
	if err != nil {
		return nil, err
	}

	if submission.FilePathLister != nil {
		s3Files, err := listFilesDryRun(&submission.FilePathLister.S3Lister)
		if err != nil {
			return nil, errors.Wrap(err, schema.FilePathListerKey)
		}

		return s3Files, nil
	}

	if submission.DelimitedFiles != nil {
		s3Files, err := listFilesDryRun(&submission.DelimitedFiles.S3Lister)
		if err != nil {
			return nil, errors.Wrap(err, schema.DelimitedFilesKey)
		}

		return s3Files, nil
	}

	return nil, nil
}

func SubmitJob(apiName string, submission *schema.BatchJobSubmission) (*spec.BatchJob, error) {
	err := validateJobSubmission(submission)
	if err != nil {
		return nil, err
	}

	virtualService, err := config.K8s.GetVirtualService(workloads.K8sName(apiName))
	if err != nil {
		return nil, err
	}

	apiID := virtualService.Labels["apiID"]
	jobID := spec.MonotonicallyDecreasingID()

	apiSpec, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	jobSpec := spec.BatchJob{
		RuntimeBatchJobConfig: submission.RuntimeBatchJobConfig,
		JobKey: spec.JobKey{
			APIName: apiName,
			ID:      jobID,
			Kind:    userconfig.BatchAPIKind,
		},
		APIID:     apiSpec.ID,
		StartTime: time.Now(),
	}

	err = uploadJobSpec(&jobSpec)
	if err != nil {
		return nil, err
	}

	// upload job payload for enqueuer
	payloadKey := spec.JobPayloadKey(config.ClusterConfig.ClusterUID, userconfig.BatchAPIKind, apiName, jobID)
	if err = config.AWS.UploadJSONToS3(submission, config.ClusterConfig.Bucket, payloadKey); err != nil {
		return nil, err
	}

	var jobConfig *string
	if submission.Config != nil {
		jobConfigBytes, err := yaml.Marshal(submission.Config)
		if err != nil {
			return nil, err
		}
		jobConfig = pointer.String(string(jobConfigBytes))
	}

	var timeout *kmeta.Duration
	if submission.Timeout != nil {
		timeout = &kmeta.Duration{Duration: time.Duration(*submission.Timeout) * time.Second}
	}

	var deadLetterQueue *batch.DeadLetterQueueSpec
	if submission.SQSDeadLetterQueue != nil {
		deadLetterQueue = &batch.DeadLetterQueueSpec{
			ARN:             submission.SQSDeadLetterQueue.ARN,
			MaxReceiveCount: int32(submission.SQSDeadLetterQueue.MaxReceiveCount),
		}
	}

	batchJob := batch.BatchJob{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      jobID,
			Namespace: config.K8s.Namespace,
			Labels: map[string]string{
				"apiName":        apiName,
				"apiID":          apiID,
				"specID":         virtualService.Labels["specID"],
				"apiKind":        userconfig.BatchAPIKind.String(),
				"cortex.dev/api": "true",
			},
		},
		Spec: batch.BatchJobSpec{
			APIName:         apiName,
			APIID:           apiID,
			Workers:         int32(submission.Workers),
			Config:          jobConfig,
			Timeout:         timeout,
			DeadLetterQueue: deadLetterQueue,
			TTL:             &kmeta.Duration{Duration: _batchJobTTL},
			NodeGroups:      apiSpec.NodeGroups,
			Probes:          workloads.GetReadinessProbesFromContainers(apiSpec.Pod.Containers),
		},
	}

	ctx := context.Background()
	if err = config.K8s.Create(ctx, &batchJob); err != nil {
		return nil, err
	}

	return &jobSpec, nil
}

func StopJob(jobKey spec.JobKey) error {
	return config.K8s.Delete(context.Background(), &batch.BatchJob{
		ObjectMeta: kmeta.ObjectMeta{Name: jobKey.ID, Namespace: config.K8s.Namespace},
	})
}

func uploadJobSpec(jobSpec *spec.BatchJob) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.ClusterConfig.Bucket, jobSpec.SpecFilePath(config.ClusterConfig.ClusterUID))
	if err != nil {
		return err
	}
	return nil
}
