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

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/go-logr/logr"
	kbatch "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batch "github.com/cortexlabs/cortex/operator/apis/batch/v1alpha1"
)

type batchJobStatusInfo struct {
	QueueExists     bool
	EnqueuingStatus string
	WorkerJob       *kbatch.Job
}

// BatchJobReconciler reconciles a BatchJob object
type BatchJobReconciler struct {
	client.Client
	Log    logr.Logger
	SQS    *sqs.SQS
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch.cortex.dev,resources=batchjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.cortex.dev,resources=batchjobs/status,verbs=get;update;patch

// Reconcile runs a reconciliation iteration for BatchJobs.batch.cortex.dev
func (r *BatchJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("batchjob", req.NamespacedName)

	// Step 1: get resource from request
	batchJob := batch.BatchJob{}
	log.V(1).Info("retrieving batchjob resource")
	if err := r.Get(ctx, req.NamespacedName, &batchJob); err != nil {
		if !kerrors.IsNotFound(err) {
			log.Error(err, "failed to retrieve batchjob")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Step 2: Update Status
	queueExists, err := r.checkIfQueueExists(batchJob)
	if err != nil {
		log.Error(err, "failed to check if queue exists")
		return ctrl.Result{}, err
	}

	enqueueingStatus, err := r.checkEnqueueingStatus(batchJob)
	if err != nil {
		log.Error(err, "failed to check enqueuing status")
		return ctrl.Result{}, err
	}

	workerJob, err := r.getWorkerJob(batchJob)
	if err != nil && !kerrors.IsNotFound(err) {
		log.Error(err, "failed to get worker job")
		return ctrl.Result{}, err
	}

	statusInfo := batchJobStatusInfo{
		QueueExists:     queueExists,
		EnqueuingStatus: enqueueingStatus,
		WorkerJob:       workerJob,
	}

	if err = r.updateStatus(ctx, &batchJob, statusInfo); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	// TODO: finalizer

	// Step 3: Create resources
	var queueURL string
	if !queueExists {
		log.V(1).Info("creating queue")
		queueURL, err = r.createQueue(batchJob)
		if err != nil {
			log.Error(err, "failed to create queue")
			return ctrl.Result{}, err
		}
	} else {
		queueURL = r.getQueueURL(batchJob)
	}

	// FIXME: replace status strings with enum
	if enqueueingStatus != "done" { // FIXME
		log.V(1).Info("enqueing payload")
		if err = r.enqueuePayload(batchJob, queueURL); err != nil {
			log.Error(err, "failed to start enqueueing the payload")
			return ctrl.Result{}, err
		}
	} else if enqueueingStatus == "in_progress" {
		// wait for enqueing process to be reach a final state (done|failed)
		return ctrl.Result{}, nil
	} else if enqueueingStatus == "failed" {
		log.Info("failed to enqueue payload")
		return ctrl.Result{}, nil
	}

	// TODO: create only if there are no workers created
	workerJobExists := workerJob != nil
	if !workerJobExists {
		log.V(1).Info("creating workers")
		if err = r.createWorkerJob(batchJob); err != nil {
			log.Error(err, "failed to create worker job")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the BatchJob controller with the controller manager
func (r *BatchJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batch.BatchJob{}).
		Owns(&kbatch.Job{}).
		Complete(r)
}
