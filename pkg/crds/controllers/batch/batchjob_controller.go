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

package batchcontrollers

import (
	"context"
	"time"

	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/crds/controllers"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/go-logr/logr"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	kbatch "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	_sqsFinalizer                 = "sqs.finalizers.batch.cortex.dev"
	_s3Finalizer                  = "s3.finalizers.batch.cortex.dev"
	_completedTimestampAnnotation = "batch.cortex.dev/completed_timestamp"
)

// BatchJobReconciler reconciles a BatchJob object
type BatchJobReconciler struct {
	client.Client
	Config        BatchJobReconcilerConfig
	Log           logr.Logger
	AWS           *awslib.Client
	ClusterConfig *clusterconfig.Config
	Prometheus    promv1.API
	Scheme        *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch.cortex.dev,resources=batchjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.cortex.dev,resources=batchjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.cortex.dev,resources=batchjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BatchJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("cortex.labels",
		map[string]string{
			"batchjob": req.NamespacedName.String(),
			"apiKind":  userconfig.BatchAPIKind.String(),
		},
	)

	// Step 1: get resource from request
	batchJob := batch.BatchJob{}
	log.V(1).Info("retrieving resource")
	if err := r.Get(ctx, req.NamespacedName, &batchJob); err != nil {
		if !kerrors.IsNotFound(err) {
			log.Error(err, "failed to retrieve resource")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log = r.Log.WithValues("cortex.labels",
		map[string]string{
			"batchjob": req.NamespacedName.String(),
			"apiKind":  userconfig.BatchAPIKind.String(),
			"apiName":  batchJob.Spec.APIName,
			"apiID":    batchJob.Spec.APIID,
			"jobID":    batchJob.Name,
		},
	)
	// Step 2: create finalizer or handle deletion
	if batchJob.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so we add our finalizer if it does not exist yet,
		if !slices.HasString(batchJob.ObjectMeta.Finalizers, _sqsFinalizer) &&
			!slices.HasString(batchJob.ObjectMeta.Finalizers, _s3Finalizer) {
			log.V(1).Info("adding finalizers")
			batchJob.ObjectMeta.Finalizers = append(batchJob.ObjectMeta.Finalizers, _sqsFinalizer, _s3Finalizer)
			if err := r.Update(ctx, &batchJob); err != nil {
				log.Error(err, "failed to add finalizers to resource")
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if slices.HasString(batchJob.ObjectMeta.Finalizers, _sqsFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			log.V(1).Info("deleting SQS queue")
			if err := r.deleteSQSQueue(batchJob); err != nil {
				log.Error(err, "failed to delete SQS queue")
				return ctrl.Result{}, err
			}

			log.V(1).Info("removing sqs finalizer")
			// remove our finalizer from the list and update it.
			batchJob.ObjectMeta.Finalizers = slices.RemoveString(batchJob.ObjectMeta.Finalizers, _sqsFinalizer)
			if err := r.Update(ctx, &batchJob); err != nil {
				log.Error(err, "failed to remove sqs finalizer from resource")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil // return here because the status update will trigger another reconcile
		}

		if slices.HasString(batchJob.ObjectMeta.Finalizers, _s3Finalizer) {
			log.V(1).Info("persisting job to S3")
			if err := r.persistJobToS3(batchJob); err != nil {
				log.Error(err, "failed to persist job to S3")
				return ctrl.Result{}, err
			}

			log.V(1).Info("removing S3 finalizer")
			batchJob.ObjectMeta.Finalizers = slices.RemoveString(batchJob.ObjectMeta.Finalizers, _s3Finalizer)
			if err := r.Update(ctx, &batchJob); err != nil {
				log.Error(err, "failed to remove S3 finalizer from resource")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil // return here because the status update will trigger another reconcile
		}

		return ctrl.Result{}, nil
	}

	// Step 3: Update Status
	log.V(1).Info("checking if queue exists")
	queueExists, err := r.checkIfQueueExists(batchJob)
	if err != nil {
		log.Error(err, "failed to check if queue exists")
		return ctrl.Result{}, err
	}

	log.V(1).Info("getting configmap")
	configMap, err := r.getConfigMap(ctx, batchJob)
	if err != nil && !kerrors.IsNotFound(err) {
		log.Error(err, "failed to get configmap")
		return ctrl.Result{}, err
	}

	log.V(1).Info("getting worker job")
	workerJob, err := r.getWorkerJob(ctx, batchJob)
	if err != nil && !kerrors.IsNotFound(err) {
		log.Error(err, "failed to get worker job")
		return ctrl.Result{}, err
	}

	log.V(1).Info("checking enqueuing status")
	enqueuerJob, enqueuingStatus, err := r.checkEnqueuingStatus(ctx, batchJob)
	if err != nil {
		log.Error(err, "failed to check enqueuing status")
		return ctrl.Result{}, err
	}

	var totalBatchCount int
	if enqueuingStatus == batch.EnqueuingDone {
		totalBatchCount, err = r.Config.GetTotalBatchCount(r, batchJob)
	}

	workerJobExists := workerJob != nil
	configMapExists := configMap != nil
	statusInfo := batchJobStatusInfo{
		QueueExists:     queueExists,
		EnqueuingStatus: enqueuingStatus,
		EnqueuerJob:     enqueuerJob,
		WorkerJob:       workerJob,
		TotalBatchCount: totalBatchCount,
	}

	log.V(1).Info("status data successfully acquired",
		"queueExists", queueExists,
		"configMapExists", configMapExists,
		"enqueuingStatus", enqueuingStatus,
		"workerJobExists", workerJobExists,
	)

	log.V(1).Info("updating status")
	if err = r.updateStatus(ctx, &batchJob, statusInfo); err != nil {
		if controllers.IsOptimisticLockError(err) {
			log.Info("conflict during status update, retrying")
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	log.V(1).Info("current job status", "jobStatus", batchJob.Status.Status)

	// Step 4: Add a completion timestamp annotation if job is in a completed state
	var completedTimestamp *time.Time
	if batchJob.Status.Status.IsCompleted() {
		completedTimestampStr, completedTimestampExists := batchJob.Annotations[_completedTimestampAnnotation]
		if !completedTimestampExists {
			if err = r.updateCompletedTimestamp(ctx, &batchJob); err != nil {
				log.Error(err, "failed to update completed timestamp annotation")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		ts, err := time.Parse(time.RFC3339, completedTimestampStr)
		if err != nil {
			log.Error(err, "failed to parse completed timestamp string")
			return ctrl.Result{}, err
		}
		completedTimestamp = &ts
	}

	// Step 5: Create resources
	var queueURL string
	if !queueExists {
		log.Info("creating queue")
		queueURL, err = r.createQueue(batchJob)
		if err != nil {
			log.Error(err, "failed to create queue")
			return ctrl.Result{}, err
		}
	} else {
		queueURL = r.getQueueURL(batchJob)
	}

	switch enqueuingStatus {
	case batch.EnqueuingNotStarted:
		log.Info("enqueuing payload")
		if err = r.enqueuePayload(ctx, batchJob, queueURL); err != nil {
			log.Error(err, "failed to start enqueuing the payload")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	case batch.EnqueuingInProgress:
		// wait for enqueuing process to be reach a final state (done|failed)
		return ctrl.Result{}, nil
	case batch.EnqueuingFailed:
		log.Info("failed to enqueue payload")
	case batch.EnqueuingDone:
		if !configMapExists {
			log.V(1).Info("creating worker configmap")
			if err = r.createWorkerConfigMap(ctx, batchJob, queueURL); err != nil {
				log.Error(err, "failed to create worker configmap")
				return ctrl.Result{}, err
			}

		}
		if !workerJobExists {
			log.Info("creating worker job")
			if err = r.createWorkerJob(ctx, batchJob, queueURL); err != nil {
				log.Error(err, "failed to create worker job")
				return ctrl.Result{}, err
			}
		}
	}

	// Step 6: Delete self if TTL is enabled and reached a final state
	if batchJob.Spec.TTL != nil && completedTimestamp != nil {
		afterFinishedDuration := time.Since(*completedTimestamp)
		if afterFinishedDuration >= batchJob.Spec.TTL.Duration {
			if err = r.Delete(ctx, &batchJob); err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
			log.Info("TTL exceeded, deleting resource")
			return ctrl.Result{}, nil
		}
		log.V(1).Info("scheduling reconciliation requeue", "time", batchJob.Spec.TTL.Duration)
		return ctrl.Result{RequeueAfter: batchJob.Spec.TTL.Duration}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BatchJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batch.BatchJob{}).
		Owns(&kbatch.Job{}).
		Complete(r)
}
