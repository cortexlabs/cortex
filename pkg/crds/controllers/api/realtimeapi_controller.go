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

package api

import (
	"context"
	"fmt"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/crds/controllers"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/workloads"
	"github.com/go-logr/logr"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1alpha1 "github.com/cortexlabs/cortex/pkg/crds/apis/api/v1alpha1"
)

// RealtimeAPIReconciler reconciles a RealtimeAPI object
type RealtimeAPIReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=api.cortex.dev,resources=realtimeapis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.cortex.dev,resources=realtimeapis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=api.cortex.dev,resources=realtimeapis/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RealtimeAPIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("realtimeapi", req.NamespacedName)

	// Step 1: get resource from request
	api := apiv1alpha1.RealtimeAPI{}
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
		For(&apiv1alpha1.RealtimeAPI{}).
		Owns(&kapps.Deployment{}).
		Owns(&kcore.Service{}).
		Owns(&istionetworking.VirtualService{}).
		Complete(r)
}

func (r *RealtimeAPIReconciler) getDeployment(ctx context.Context, api apiv1alpha1.RealtimeAPI) (*kapps.Deployment, error) {
	req := client.ObjectKey{Namespace: api.Namespace, Name: workloads.K8sName(api.Name)}
	deployment := kapps.Deployment{}
	if err := r.Get(ctx, req, &deployment); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &deployment, nil
}

func (r *RealtimeAPIReconciler) updateStatus(ctx context.Context, api *apiv1alpha1.RealtimeAPI, deployment *kapps.Deployment) error {
	apiStatus := status.Pending
	api.Status.Status = apiStatus // FIXME: handle other status

	endpoint, err := r.getEndpoint(ctx, api)
	if err != nil {
		return errors.Wrap(err, "failed to get api endpoint")
	}

	api.Status.Endpoint = endpoint
	if deployment != nil {
		api.Status.DesiredReplicas = *deployment.Spec.Replicas
		api.Status.CurrentReplicas = deployment.Status.Replicas
		api.Status.ReadyReplicas = deployment.Status.ReadyReplicas
	}

	if err = r.Status().Update(ctx, api); err != nil {
		return err
	}

	return nil
}

func (r *RealtimeAPIReconciler) createOrUpdateDeployment(ctx context.Context, api apiv1alpha1.RealtimeAPI) (controllerutil.OperationResult, error) {
	deployment := kapps.Deployment{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      workloads.K8sName(api.Name),
			Namespace: api.Namespace},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, r, &deployment, func() error {
		deployment.Spec = r.desiredDeployment(api).Spec
		return nil
	})
	if err != nil {
		return op, err
	}
	return op, nil
}

func (r *RealtimeAPIReconciler) createOrUpdateService(ctx context.Context, api apiv1alpha1.RealtimeAPI) (controllerutil.OperationResult, error) {
	service := kcore.Service{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      workloads.K8sName(api.Name),
			Namespace: api.Namespace},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, r, &service, func() error {
		service.Spec = r.desiredService(api).Spec
		return nil
	})
	if err != nil {
		return op, err
	}
	return op, nil
}

func (r *RealtimeAPIReconciler) createOrUpdateVirtualService(ctx context.Context, api apiv1alpha1.RealtimeAPI) (controllerutil.OperationResult, error) {
	vs := istionetworking.VirtualService{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      workloads.K8sName(api.Name),
			Namespace: api.Namespace},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, r, &vs, func() error {
		vs.Spec = r.desiredVirtualService(api).Spec
		return nil
	})
	if err != nil {
		return op, err
	}
	return op, nil
}

func (r *RealtimeAPIReconciler) getEndpoint(ctx context.Context, api *apiv1alpha1.RealtimeAPI) (string, error) {
	req := client.ObjectKey{Namespace: consts.IstioNamespace, Name: "ingressgateway-apis"}
	svc := kcore.Service{}
	if err := r.Get(ctx, req, &svc); err != nil {
		return "", err
	}

	ingress := svc.Status.LoadBalancer.Ingress
	if ingress == nil || len(ingress) == 0 {
		return "", nil
	}

	endpoint := fmt.Sprintf("http://%s/%s",
		svc.Status.LoadBalancer.Ingress[0].Hostname, api.Spec.Networking.Endpoint,
	)

	return endpoint, nil
}

func (r *RealtimeAPIReconciler) desiredDeployment(api apiv1alpha1.RealtimeAPI) kapps.Deployment {
	panic("implement me!")
}

func (r *RealtimeAPIReconciler) desiredService(api apiv1alpha1.RealtimeAPI) kcore.Service {
	panic("implement me!")
}

func (r *RealtimeAPIReconciler) desiredVirtualService(api apiv1alpha1.RealtimeAPI) istionetworking.VirtualService {
	panic("implement me!")
}
