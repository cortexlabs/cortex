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

package autoscaler

import (
	"context"
	"fmt"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
	kapps "k8s.io/api/apps/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type RealtimeScaler struct {
	k8s        *k8s.Client
	prometheus promv1.API
	logger     *zap.SugaredLogger
}

func NewRealtimeScaler(k8sClient *k8s.Client, promClient promv1.API, logger *zap.SugaredLogger) *RealtimeScaler {
	return &RealtimeScaler{
		k8s:        k8sClient,
		prometheus: promClient,
		logger:     logger,
	}
}

func (s *RealtimeScaler) Scale(apiName string, request int32) error {
	ctx := context.Background()

	// we use the controller-runtime client to make use of the cache mechanism
	var deployment kapps.Deployment
	err := s.k8s.Get(ctx, ctrlclient.ObjectKey{
		Namespace: s.k8s.Namespace,
		Name:      workloads.K8sName(apiName),
	}, &deployment)
	if err != nil {
		return errors.Wrap(err, "failed to get deployment")
	}

	if deployment.Spec.Replicas == nil {
		return errors.Wrap(err, "k8s deployment doesn't have the replicas field set")
	}

	current := *deployment.Spec.Replicas
	if current == request {
		return nil
	}

	if request == 0 {
		if err = s.routeToActivator(&deployment); err != nil {
			return errors.Wrap(err, "failed to re-route traffic to activator")
		}
	}

	deployment.Spec.Replicas = pointer.Int32(request)

	if err = s.k8s.Update(ctx, &deployment); err != nil {
		return errors.Wrap(err, "failed to update deployment")
	}

	if current == 0 && request > 0 {
		go func() {
			if err := s.routeToService(&deployment); err != nil {
				s.logger.Errorw("failed to re-route traffic to API",
					zap.Error(err), zap.String("apiName", apiName),
				)
				telemetry.Error(err)
			}
		}()

	}

	return nil
}

func (s *RealtimeScaler) GetInFlightRequests(apiName string, window time.Duration) (*float64, error) {
	windowSeconds := int64(window.Seconds())

	// PromQL query:
	// 	sum(sum_over_time(cortex_in_flight_requests{api_name="<apiName>"}[60s])) /
	//	sum(count_over_time(cortex_in_flight_requests{api_name="<apiName>", container!="activator"}[60s]))
	query := fmt.Sprintf(
		"sum(sum_over_time(cortex_in_flight_requests{api_name=\"%s\"}[%ds])) / "+
			"max(count_over_time(cortex_in_flight_requests{api_name=\"%s\", container!=\"activator\"}[%ds]))",
		apiName, windowSeconds,
		apiName, windowSeconds,
	)

	ctx, cancel := context.WithTimeout(context.Background(), _prometheusQueryTimeoutSeconds*time.Second)
	defer cancel()

	valuesQuery, _, err := s.prometheus.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	values, ok := valuesQuery.(model.Vector)
	if !ok {
		return nil, errors.ErrorUnexpected("failed to convert prometheus metric to vector")
	}

	// no values available
	if values.Len() == 0 {
		return nil, nil
	}

	avgInflightRequests := float64(values[0].Value)

	return &avgInflightRequests, nil
}

func (s *RealtimeScaler) GetAutoscalingSpec(apiName string) (*userconfig.Autoscaling, error) {
	deployment, err := s.k8s.GetDeployment(workloads.K8sName(apiName))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deployment")
	}

	if deployment == nil {
		return nil, errors.ErrorUnexpected("unable to find k8s deployment", apiName)
	}

	autoscalingSpec, err := userconfig.AutoscalingFromAnnotations(deployment)
	if err != nil {
		return nil, err
	}

	return autoscalingSpec, nil
}

func (s *RealtimeScaler) CurrentRequestedReplicas(apiName string) (int32, error) {
	ctx := context.Background()

	// we use the controller-runtime client to make use of the cache mechanism
	var deployment kapps.Deployment
	err := s.k8s.Get(ctx, ctrlclient.ObjectKey{
		Namespace: s.k8s.Namespace,
		Name:      workloads.K8sName(apiName),
	}, &deployment)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get deployment")
	}

	if deployment.Spec.Replicas == nil {
		return 0, errors.Wrap(err, "k8s deployment doesn't have the replicas field set")
	}

	return *deployment.Spec.Replicas, nil
}

func (s *RealtimeScaler) routeToService(deployment *kapps.Deployment) error {
	ctx := context.Background()
	vs, err := s.k8s.GetVirtualService(deployment.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get virtual service")
	}

	if len(vs.Spec.Http) < 1 {
		return errors.ErrorUnexpected("virtual service does not have any http entries")
	}

	if err = s.waitForReadyReplicas(ctx, deployment); err != nil {
		return errors.Wrap(err, "no ready replicas available")
	}

	for i := range vs.Spec.Http {
		if len(vs.Spec.Http[i].Route) != 2 {
			return errors.ErrorUnexpected("virtual service does not have the required number of 2 http routes")
		}

		vs.Spec.Http[i].Route[0].Weight = 100 // service traffic
		vs.Spec.Http[i].Route[1].Weight = 0   // activator traffic
	}

	vsClient := s.k8s.IstioClientSet().NetworkingV1beta1().VirtualServices(s.k8s.Namespace)
	if _, err = vsClient.Update(ctx, vs, kmeta.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed to update virtual service")
	}

	return nil
}

func (s *RealtimeScaler) routeToActivator(deployment *kapps.Deployment) error {
	ctx := context.Background()
	vs, err := s.k8s.GetVirtualService(deployment.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get virtual service")
	}

	if len(vs.Spec.Http) < 1 {
		return errors.ErrorUnexpected("virtual service does not have any http entries")
	}

	for i := range vs.Spec.Http {
		if len(vs.Spec.Http[i].Route) != 2 {
			return errors.ErrorUnexpected("virtual service does not have the required number of 2 http routes")
		}

		vs.Spec.Http[i].Route[0].Weight = 0   // service traffic
		vs.Spec.Http[i].Route[1].Weight = 100 // activator traffic
	}

	vsClient := s.k8s.IstioClientSet().NetworkingV1beta1().VirtualServices(s.k8s.Namespace)
	if _, err = vsClient.Update(ctx, vs, kmeta.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed to update virtual service")
	}

	return nil
}

func (s *RealtimeScaler) waitForReadyReplicas(ctx context.Context, deployment *kapps.Deployment) error {
	watcher, err := s.k8s.ClientSet().AppsV1().Deployments(s.k8s.Namespace).Watch(
		ctx,
		kmeta.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", deployment.Name),
			Watch:         true,
		},
	)
	if err != nil {
		return errors.Wrap(err, "could not create deployment watcher")
	}

	defer watcher.Stop()

	ctx, cancel := context.WithTimeout(ctx, consts.WaitForReadyReplicasTimeout)
	defer cancel()

	for {
		select {
		case event := <-watcher.ResultChan():
			deploy, ok := event.Object.(*kapps.Deployment)
			if !ok {
				continue
			}

			if deploy.Status.ReadyReplicas > 0 {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
