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

	batch "github.com/cortexlabs/cortex/operator/apis/batch/v1alpha1"
	"k8s.io/api/batch/v1"
)

func (r *BatchJobReconciler) checkIfQueueExists(batchJob batch.BatchJob) (bool, error) {
	panic("implement me!")
}

func (r *BatchJobReconciler) createQueue(batchJob batch.BatchJob) (string, error) {
	panic("implement me!")
}

func (r *BatchJobReconciler) getQueueURL(batchJob batch.BatchJob) string {
	panic("implement me!")
}

func (r *BatchJobReconciler) checkEnqueueingStatus(batchJob batch.BatchJob) (string, error) {
	panic("implement me!")
}

func (r *BatchJobReconciler) enqueuePayload(batchJob batch.BatchJob, queueURL string) error {
	panic("implement me!")
}

func (r *BatchJobReconciler) createWorkerJob(job batch.BatchJob) error {
	panic("implement me!")
}

func (r *BatchJobReconciler) getWorkerJob(batchJob batch.BatchJob) (*v1.Job, error) {
	panic("implement me!")
}

func (r *BatchJobReconciler) updateStatus(ctx context.Context, batchJob *batch.BatchJob, statusInfo batchJobStatusInfo) error {
	batchJob.Status.ID = batchJob.Name

	if statusInfo.QueueExists {
		batchJob.Status.QueueURL = r.getQueueURL(*batchJob)
	}

	// TODO replace strings with enum
	switch statusInfo.EnqueuingStatus {
	case "in_progress":
		batchJob.Status.Status = "enqueing"
	case "failed":
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
