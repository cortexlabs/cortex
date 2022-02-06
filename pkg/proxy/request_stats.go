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

package proxy

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RequestStats struct {
	sync.Mutex
	counts []int64
}

func (s *RequestStats) Append(val int64) {
	s.Lock()
	defer s.Unlock()
	s.counts = append(s.counts, val)
}

func (s *RequestStats) GetAllAndDelete() []int64 {
	var output []int64
	s.Lock()
	defer s.Unlock()
	output = s.counts
	s.counts = []int64{}
	return output
}

func (s *RequestStats) Report() RequestStatsReport {
	requestCounts := s.GetAllAndDelete()

	total := 0.0
	if len(requestCounts) > 0 {
		for _, val := range requestCounts {
			total += float64(val)
		}

		total /= float64(len(requestCounts))
	}

	return RequestStatsReport{AvgInFlight: total}
}

type RequestStatsReport struct {
	AvgInFlight float64
}

type PrometheusStatsReporter struct {
	handler          http.Handler
	inFlightRequests prometheus.Gauge
}

func NewPrometheusStatsReporter() *PrometheusStatsReporter {
	inFlightRequestsGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cortex_in_flight_requests",
		Help: "The number of in-flight requests for a cortex API",
	})

	return &PrometheusStatsReporter{
		handler:          promhttp.Handler(),
		inFlightRequests: inFlightRequestsGauge,
	}
}

func (r *PrometheusStatsReporter) Report(stats RequestStatsReport) {
	r.inFlightRequests.Set(stats.AvgInFlight)
}

func (r *PrometheusStatsReporter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}
