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
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewAsyncPrometheusStatsReporter(t *testing.T) {
	t.Parallel()

	statsReporter := NewAsyncPrometheusStatsReporter()

	statsReporter.HandleEvent(
		RequestEvent{
			StatusCode: 200,
			Duration:   100 * time.Millisecond,
		},
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/metrics", nil)
	statsReporter.ServeHTTP(w, r)

	result := w.Body.String()
	require.Contains(t, result, "cortex_async_latency")
	require.Contains(t, result, "cortex_async_request_count")
}

func TestAsyncStatsReporter_HandleEvent(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()

	latenciesHist := promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
		Name: "cortex_async_latency",
		Help: "Histogram of the latencies for an AsyncAPI kind in seconds",
	}, []string{"status_code"})

	requestCounter := promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Name: "cortex_async_request_count",
		Help: "Request count for an AsyncAPI",
	}, []string{"status_code"})

	statsReporter := AsyncStatsReporter{
		latencies:    latenciesHist,
		requestCount: requestCounter,
	}

	statsReporter.HandleEvent(
		RequestEvent{
			StatusCode: 200,
			Duration:   100 * time.Millisecond,
		},
	)

	expectedHist := `
# HELP cortex_async_latency Histogram of the latencies for an AsyncAPI kind in seconds
# TYPE cortex_async_latency histogram
cortex_async_latency_bucket{status_code="200",le="0.005"} 0
cortex_async_latency_bucket{status_code="200",le="0.01"} 0
cortex_async_latency_bucket{status_code="200",le="0.025"} 0
cortex_async_latency_bucket{status_code="200",le="0.05"} 0
cortex_async_latency_bucket{status_code="200",le="0.1"} 1
cortex_async_latency_bucket{status_code="200",le="0.25"} 1
cortex_async_latency_bucket{status_code="200",le="0.5"} 1
cortex_async_latency_bucket{status_code="200",le="1"} 1
cortex_async_latency_bucket{status_code="200",le="2.5"} 1
cortex_async_latency_bucket{status_code="200",le="5"} 1
cortex_async_latency_bucket{status_code="200",le="10"} 1
cortex_async_latency_bucket{status_code="200",le="+Inf"} 1
cortex_async_latency_sum{status_code="200"} 0.1
cortex_async_latency_count{status_code="200"} 1
`

	require.Equal(t, float64(1), testutil.ToFloat64(statsReporter.requestCount))
	require.NoError(t, testutil.CollectAndCompare(statsReporter.latencies, strings.NewReader(expectedHist)))
}
