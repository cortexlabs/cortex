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

package autoscaler

import (
	"context"
	"fmt"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
	kapps "k8s.io/api/apps/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	_waitForReadyReplicasTimeout = 10 * time.Minute
)

type realtimeScaler struct {
	k8s        *k8s.Client
	prometheus promv1.API
	logger     *zap.SugaredLogger
}

func NewRealtimeScaler(k8sClient *k8s.Client, promClient promv1.API, logger *zap.SugaredLogger) Scaler {
	return &realtimeScaler{
		k8s:        k8sClient,
		prometheus: promClient,
		logger:     logger,
	}
}

func (s *realtimeScaler) Scale(apiName string, request int32) error {
	deployment, err := s.k8s.GetDeployment(workloads.K8sName(apiName))
	if err != nil {
		return err
	}

	if deployment == nil {
		return errors.ErrorUnexpected("unable to find k8s deployment", apiName)
	}

	current := deployment.Status.Replicas

	if current == request {
		return nil
	}

	if request == 0 {
		if err := s.routeToActivator(deployment); err != nil {
			return err
		}
	}

	deployment.Spec.Replicas = pointer.Int32(request)

	if _, err := s.k8s.UpdateDeployment(deployment); err != nil {
		return err
	}

	if current == 0 && request > 0 {
		go func() {
			if err := s.routeToService(deployment); err != nil {
				s.logger.Errorw("failed to re-route traffic to API",
					zap.Error(err), zap.String("apiName", apiName),
				)
				// TODO: telemetry
			}
		}()

	}

	return nil
}

func (s *realtimeScaler) GetInFlightRequests(apiName string, window time.Duration) (*float64, error) {
	windowSeconds := int64(window.Seconds())

	// PromQL query:
	// 	sum(sum_over_time(cortex_in_flight_requests{api_name="<apiName>"}[60s])) /
	//	sum(count_over_time(cortex_in_flight_requests{api_name="<apiName>"}[60s]))
	query := fmt.Sprintf(
		"sum(sum_over_time(cortex_in_flight_requests{api_name=\"%s\"}[%ds])) / "+
			"max(count_over_time(cortex_in_flight_requests{api_name=\"%s\"}[%ds]))",
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

func (s *realtimeScaler) GetAutoscalingSpec(apiName string) (*userconfig.Autoscaling, error) {
	deployment, err := s.k8s.GetDeployment(workloads.K8sName(apiName))
	if err != nil {
		return nil, err
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

func (s *realtimeScaler) CurrentReplicas(apiName string) (int32, error) {
	deployment, err := s.k8s.GetDeployment(workloads.K8sName(apiName))
	if err != nil {
		return 0, err
	}

	if deployment == nil {
		return 0, errors.ErrorUnexpected("unable to find k8s deployment", apiName)
	}

	return deployment.Status.Replicas, nil
}

func (s *realtimeScaler) routeToService(deployment *kapps.Deployment) error {
	ctx := context.Background()
	vs, err := s.k8s.GetVirtualService(deployment.Name)
	if err != nil {
		return err // TODO: error handling
	}

	if len(vs.Spec.Http) < 1 {
		return errors.ErrorUnexpected("virtual service does not have any http entries")
	}

	if len(vs.Spec.Http[0].Route) != 2 {
		return errors.ErrorUnexpected("virtual service does not have the required minimum number of 2 http routes")
	}

	if err = s.waitForReadyReplicas(ctx, deployment); err != nil {
		return err // TODO: error handling
	}

	vs.Spec.Http[0].Route[0].Weight = 100 // service traffic
	vs.Spec.Http[0].Route[1].Weight = 0   // activator traffic

	vsClient := s.k8s.IstioClientSet().NetworkingV1beta1().VirtualServices(s.k8s.Namespace)
	if _, err = vsClient.Update(ctx, vs, kmeta.UpdateOptions{}); err != nil {
		return err // TODO: error handling
	}

	return nil
}

func (s *realtimeScaler) routeToActivator(deployment *kapps.Deployment) error {
	ctx := context.Background()
	vs, err := s.k8s.GetVirtualService(deployment.Name)
	if err != nil {
		return err // TODO: error handling
	}

	if len(vs.Spec.Http) < 1 {
		return errors.ErrorUnexpected("virtual service does not have any http entries")
	}

	if len(vs.Spec.Http[0].Route) != 2 {
		return errors.ErrorUnexpected("virtual service does not have the required minimum number of 2 http routes")
	}

	vs.Spec.Http[0].Route[0].Weight = 0   // service traffic
	vs.Spec.Http[0].Route[1].Weight = 100 // activator traffic

	vsClient := s.k8s.IstioClientSet().NetworkingV1beta1().VirtualServices(s.k8s.Namespace)
	if _, err = vsClient.Update(ctx, vs, kmeta.UpdateOptions{}); err != nil {
		return err // TODO: error handling
	}

	return nil
}

func (s *realtimeScaler) waitForReadyReplicas(ctx context.Context, deployment *kapps.Deployment) error {
	watcher, err := s.k8s.ClientSet().AppsV1().Deployments(s.k8s.Namespace).Watch(
		ctx,
		kmeta.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", deployment.Name),
			Watch:         true,
		},
	)
	if err != nil {
		return err
	}

	defer watcher.Stop()

	ctx, cancel := context.WithTimeout(ctx, _waitForReadyReplicasTimeout)
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
