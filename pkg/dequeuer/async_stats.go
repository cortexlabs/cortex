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

package dequeuer

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type AsyncStatsReporter struct {
	handler      http.Handler
	latencies    *prometheus.HistogramVec
	requestCount *prometheus.CounterVec
}

func NewAsyncPrometheusStatsReporter() *AsyncStatsReporter {
	latenciesHist := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "cortex_async_latency",
		Help: "Histogram of the latencies for an AsyncAPI kind in seconds",
	}, []string{"status_code"})

	requestCounter := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cortex_async_request_count",
		Help: "Request count for an AsyncAPI",
	}, []string{"status_code"})

	handler := promhttp.Handler()

	return &AsyncStatsReporter{
		handler:      handler,
		latencies:    latenciesHist,
		requestCount: requestCounter,
	}
}

func (r *AsyncStatsReporter) HandleEvent(event RequestEvent) {
	labels := map[string]string{
		"status_code": strconv.Itoa(event.StatusCode),
	}

	r.latencies.With(labels).Observe(event.Duration.Seconds())
	r.requestCount.With(labels).Add(1)
}

func (r *AsyncStatsReporter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}
