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

package realtimeapi

import (
	"context"
	"fmt"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	_metricsWindowHours    = 24
	_metricsRequestTimeout = 10 // seconds
	_defaultSvcDomain      = "default.svc.cluster.local"
)

func GetMultipleMetrics(apis []spec.API) ([]metrics.Metrics, error) {
	allMetrics := make([]metrics.Metrics, len(apis))
	fns := make([]func() error, len(apis))

	for i := range apis {
		localIdx := i
		api := apis[i]
		fns[i] = func() error {
			apiMetrics, err := GetMetrics(&api)
			if err != nil {
				return err
			}
			allMetrics[localIdx] = *apiMetrics
			return nil
		}
	}

	if len(fns) > 0 {
		err := parallel.RunFirstErr(fns[0], fns[1:]...)
		if err != nil {
			return nil, err
		}
	}

	return allMetrics, nil
}

func GetMetrics(api *spec.API) (*metrics.Metrics, error) {
	client, err := promapi.NewClient(promapi.Config{
		Address: config.Cluster.PrometheusURL,
	})
	if err != nil {
		return nil, err
	}

	promAPIv1 := promv1.NewAPI(client)

	var (
		reqCount       float64
		avgLatency     *float64
		statusCodes2XX float64
		statusCodes4XX float64
		statusCodes5XX float64
	)

	err = parallel.RunFirstErr(
		func() error {
			var err error
			reqCount, err = getRequestCountMetric(promAPIv1, api.Name)
			return err
		},
		func() error {
			var err error
			avgLatency, err = getAvgLatencyMetric(promAPIv1, api.Name)
			return err
		},
		func() error {
			var err error
			statusCodes2XX, err = getStatusCode2XXMetric(promAPIv1, api.Name)
			return err
		},
		func() error {
			var err error
			statusCodes4XX, err = getStatusCode4XXMetric(promAPIv1, api.Name)
			return err
		},
		func() error {
			var err error
			statusCodes5XX, err = getStatusCode5XXMetric(promAPIv1, api.Name)
			return err
		},
	)

	if err != nil {
		return nil, err
	}

	return &metrics.Metrics{
		APIName: api.Name,
		NetworkStats: &metrics.NetworkStats{
			Latency: avgLatency,
			Code2XX: int(statusCodes2XX),
			Code4XX: int(statusCodes4XX),
			Code5XX: int(statusCodes5XX),
			Total:   int(reqCount),
		},
	}, nil
}

func getRequestCountMetric(promAPIv1 promv1.API, apiName string) (float64, error) {
	query := fmt.Sprintf(
		"sum(increase(istio_requests_total{destination_service_name=\"api-%s.%s\"}[%dh]) >= 0)",
		apiName, _defaultSvcDomain, _metricsWindowHours,
	)

	values, err := queryPrometheusVec(promAPIv1, query)
	if err != nil {
		return 0, err
	}

	if values.Len() == 0 {
		return 0, nil
	}

	requestCount := float64(values[0].Value)
	return requestCount, nil
}

func getAvgLatencyMetric(promAPIv1 promv1.API, apiName string) (*float64, error) {
	query := fmt.Sprintf(
		"rate(istio_request_duration_milliseconds_sum{destination_service_name=\"api-%s.%s\", reporter=\"source\", response_code=\"200\"}[%dh]) "+
			"/ rate(istio_request_duration_milliseconds_count{destination_service_name=\"api-%s.%s\", reporter=\"source\", response_code=\"200\"}[%dh]) >= 0",
		apiName, _defaultSvcDomain, _metricsWindowHours,
		apiName, _defaultSvcDomain, _metricsWindowHours,
	)

	values, err := queryPrometheusVec(promAPIv1, query)
	if err != nil {
		return nil, err
	}

	if values.Len() == 0 {
		return nil, nil
	}

	avgLatency := float64(values[0].Value)
	return &avgLatency, nil
}

func getStatusCode2XXMetric(promAPIv1 promv1.API, apiName string) (float64, error) {
	query := fmt.Sprintf(
		"sum(increase(istio_requests_total{destination_service_name=\"api-%s.%s\", response_code=~\"^2[0-9]{2}$\"}[%dh]) >= 0)",
		apiName, _defaultSvcDomain, _metricsWindowHours,
	)

	values, err := queryPrometheusVec(promAPIv1, query)
	if err != nil {
		return 0, err
	}

	if values.Len() == 0 {
		return 0, nil
	}

	statusCodes2XX := float64(values[0].Value)
	return statusCodes2XX, nil
}

func getStatusCode4XXMetric(promAPIv1 promv1.API, apiName string) (float64, error) {
	query := fmt.Sprintf(
		"sum(increase(istio_requests_total{destination_service_name=\"api-%s.%s\", response_code=~\"^4[0-9]{2}$\"}[%dh]) >= 0)",
		apiName, _defaultSvcDomain, _metricsWindowHours,
	)

	values, err := queryPrometheusVec(promAPIv1, query)
	if err != nil {
		return 0, err
	}

	if values.Len() == 0 {
		return 0, nil
	}

	statusCodes4XX := float64(values[0].Value)
	return statusCodes4XX, nil
}

func getStatusCode5XXMetric(promAPIv1 promv1.API, apiName string) (float64, error) {
	query := fmt.Sprintf(
		"sum(increase(istio_requests_total{destination_service_name=\"api-%s.%s\", response_code=~\"^5[0-9]{2}$\"}[%dh]) >= 0)",
		apiName, _defaultSvcDomain, _metricsWindowHours,
	)

	values, err := queryPrometheusVec(promAPIv1, query)
	if err != nil {
		return 0, err
	}

	if values.Len() == 0 {
		return 0, nil
	}

	statusCodes5XX := float64(values[0].Value)
	return statusCodes5XX, nil
}

func queryPrometheusVec(promAPIv1 promv1.API, query string) (model.Vector, error) {
	ctx, cancel := context.WithTimeout(context.Background(), _metricsRequestTimeout*time.Second)
	defer cancel()

	valuesQuery, err := promAPIv1.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	values, ok := valuesQuery.(model.Vector)
	if !ok {
		return nil, errors.ErrorUnexpected("failed to convert metric to vector")
	}

	return values, nil
}
