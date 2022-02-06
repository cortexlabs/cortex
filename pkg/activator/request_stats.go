/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Modifications Copyright 2022 Cortex Labs, Inc.
*/

package activator

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusStatsReporter struct {
	handler          http.Handler
	inFlightRequests *prometheus.GaugeVec
}

func NewPrometheusStatsReporter() *PrometheusStatsReporter {
	inFlightRequestsGauge := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cortex_in_flight_requests",
		Help: "The number of in-flight requests for a cortex API",
	}, []string{"api_name"})

	return &PrometheusStatsReporter{
		handler:          promhttp.Handler(),
		inFlightRequests: inFlightRequestsGauge,
	}
}

func (r *PrometheusStatsReporter) AddAPI(apiName string) {
	r.inFlightRequests.WithLabelValues(apiName).Set(0)
}

func (r *PrometheusStatsReporter) RemoveAPI(apiName string) {
	r.inFlightRequests.DeleteLabelValues(apiName)
}

func (r *PrometheusStatsReporter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}
