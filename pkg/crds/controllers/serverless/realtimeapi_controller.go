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

package serverlesscontroller

import (
	"context"
	"fmt"

	serverless "github.com/cortexlabs/cortex/pkg/crds/apis/serverless/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/crds/controllers"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/go-logr/logr"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const _terminationGracePeriodSeconds int64 = 60 // seconds

// RealtimeAPIReconciler reconciles a RealtimeAPI object
type RealtimeAPIReconciler struct {
	client.Client
	ClusterConfig *clusterconfig.Config
	Log           logr.Logger
	Scheme        *runtime.Scheme
}

// +kubebuilder:rbac:groups=serverless.cortex.dev,resources=realtimeapis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=serverless.cortex.dev,resources=realtimeapis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=serverless.cortex.dev,resources=realtimeapis/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RealtimeAPIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("realtimeapi", req.NamespacedName)

	// Step 1: get resource from request
	api := serverless.RealtimeAPI{}
	log.V(1).Info("retrieving resource")
	if err := r.Get(ctx, req.NamespacedName, &api); err != nil {
		if !kerrors.IsNotFound(err) {
			log.Error(err, "failed to retrieve resource")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Step 2: Update status
	log.V(1).Info("getting deployment")
	deployment, err := r.getDeployment(ctx, api)
	if err != nil {
		log.Error(err, "failed to get deployment")
		return ctrl.Result{}, err
	}

	log.V(1).Info("updating status")
	if err = r.updateStatus(ctx, &api, deployment); err != nil {
		if controllers.IsOptimisticLockError(err) {
			log.Info("conflict during status update, retrying")
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	// Step 3: Create or Update Resources
	deployOp, err := r.createOrUpdateDeployment(ctx, api)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.V(1).Info(fmt.Sprintf("deployment %s", deployOp))

	svcOp, err := r.createOrUpdateService(ctx, api)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.V(1).Info(fmt.Sprintf("service %s", svcOp))

	vsOp, err := r.createOrUpdateVirtualService(ctx, api)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.V(1).Info(fmt.Sprintf("virtual service %s", vsOp))

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RealtimeAPIReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&serverless.RealtimeAPI{}).
		Owns(&kapps.Deployment{}).
		Owns(&kcore.Service{}).
		Owns(&istioclientnetworking.VirtualService{}).
		Complete(r)
}
