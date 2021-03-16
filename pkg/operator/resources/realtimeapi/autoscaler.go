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
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/prometheus/common/model"
)

const (
	_prometheusQueryTimeoutSeconds = 10
)

func getInflightRequests(apiName string, window time.Duration) (*float64, error) {
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

	valuesQuery, err := config.Prometheus.Query(ctx, query, time.Now())
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
