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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type AsyncScaler struct {
	k8s        *k8s.Client
	prometheus promv1.API
}

func NewAsyncScaler(k8sClient *k8s.Client, promClient promv1.API) *AsyncScaler {
	return &AsyncScaler{
		k8s:        k8sClient,
		prometheus: promClient,
	}
}

func (s *AsyncScaler) Scale(apiName string, request int32) error {
	deployment, err := s.k8s.GetDeployment(workloads.K8sName(apiName))
	if err != nil {
		return err
	}

	if deployment == nil {
		return errors.ErrorUnexpected("unable to find k8s deployment", apiName)
	}

	if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == request {
		return nil
	}

	deployment.Spec.Replicas = pointer.Int32(request)

	if _, err = s.k8s.UpdateDeployment(deployment); err != nil {
		return err
	}

	return nil
}

func (s *AsyncScaler) GetInFlightRequests(apiName string, window time.Duration) (*float64, error) {
	windowSeconds := int64(window.Seconds())

	// PromQL query:
	// 	sum(sum_over_time(cortex_async_in_flight{api_name="<apiName>"}[60s])) /
	//	sum(count_over_time(cortex_async_in_flight{api_name="<apiName>"}[60s]))
	query := fmt.Sprintf(
		"sum(sum_over_time(cortex_async_in_flight{api_name=\"%s\"}[%ds])) / "+
			"max(count_over_time(cortex_async_in_flight{api_name=\"%s\"}[%ds]))",
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

	if values.Len() != 0 {
		return pointer.Float64(float64(values[0].Value)), nil
	}
	return nil, nil
}

func (s *AsyncScaler) GetAutoscalingSpec(apiName string) (*userconfig.Autoscaling, error) {
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

func (s *AsyncScaler) CurrentRequestedReplicas(apiName string) (int32, error) {
	deployment, err := s.k8s.GetDeployment(workloads.K8sName(apiName))
	if err != nil {
		return 0, err
	}

	if deployment == nil {
		return 0, errors.ErrorUnexpected("unable to find k8s deployment", apiName)
	}

	if deployment.Spec.Replicas == nil {
		return 0, errors.ErrorUnexpected("k8s deployment doesn't have the replicas field set", apiName)
	}

	return *deployment.Spec.Replicas, nil
}
