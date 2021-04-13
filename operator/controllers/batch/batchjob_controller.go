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

	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/go-logr/logr"
	kbatch "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batch "github.com/cortexlabs/cortex/operator/apis/batch/v1alpha1"
)

// BatchJobReconciler reconciles a BatchJob object
type BatchJobReconciler struct {
	client.Client
	Log           logr.Logger
	AWS           *awslib.Client
	ClusterConfig *clusterconfig.Config
	Scheme        *runtime.Scheme
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
	log.V(1).Info("checking if queue exists")
	queueExists, err := r.checkIfQueueExists(batchJob)
	if err != nil {
		log.Error(err, "failed to check if queue exists")
		return ctrl.Result{}, err
	}

	log.V(1).Info("getting worker job")
	workerJob, err := r.getWorkerJob(ctx, batchJob)
	if err != nil && !kerrors.IsNotFound(err) {
		log.Error(err, "failed to get worker job")
		return ctrl.Result{}, err
	}

	log.V(1).Info("checking enqueueing status")
	enqueueingStatus, err := r.checkEnqueueingStatus(ctx, batchJob, workerJob)
	if err != nil {
		log.Error(err, "failed to check enqueuing status")
		return ctrl.Result{}, err
	}

	workerJobExists := workerJob != nil
	statusInfo := batchJobStatusInfo{
		QueueExists:     queueExists,
		EnqueuingStatus: enqueueingStatus,
		WorkerJob:       workerJob,
	}

	log.V(1).Info("status data successfully acquired",
		"queueExists", queueExists,
		"enqueuingStatus", enqueueingStatus,
		"workerJobExists", workerJobExists,
	)

	log.V(1).Info("updating status")
	if err = r.updateStatus(ctx, &batchJob, statusInfo); err != nil {
		if kerrors.IsConflict(err) {
			log.Info("conflict during status update, retrying")
		} else {
			log.Error(err, "failed to update status")
		}
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

	switch enqueueingStatus {
	case EnqueuingNotStarted:
		log.V(1).Info("enqueing payload")
		if err = r.enqueuePayload(ctx, batchJob, queueURL); err != nil {
			log.Error(err, "failed to start enqueueing the payload")
			return ctrl.Result{}, err
		}
	case EnqueuingInProgress:
		// wait for enqueuing process to be reach a final state (done|failed)
		return ctrl.Result{}, nil
	case EnqueuingFailed:
		log.Info("failed to enqueue payload")
		return ctrl.Result{}, nil
	}

	if !workerJobExists {
		log.V(1).Info("creating worker job")
		if err = r.createWorkerJob(ctx, batchJob); err != nil {
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
