/*
Copyright 2019 Cortex Labs, Inc.

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

package metrics

import (
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/stretchr/testify/require"
)

func TestMergeAvg(t *testing.T) {
	floatNilPtr := (*float64)(nil)

	require.Equal(t, floatNilPtr, mergeAvg(nil, 0, nil, 0))
	require.Equal(t, floatNilPtr, mergeAvg(nil, 1, nil, 0))

	require.Equal(t, float64(1), *mergeAvg(pointer.Float64(1), 1, nil, 1))
	require.Equal(t, pointer.Float64(1), mergeAvg(nil, 1, pointer.Float64(1), 1))

	require.Equal(t, floatNilPtr, mergeAvg(pointer.Float64(1), 0, nil, 1))
	require.Equal(t, floatNilPtr, mergeAvg(nil, 1, pointer.Float64(1), 0))

	require.Equal(t, float64(1.25), *mergeAvg(pointer.Float64(1.25), 5, nil, 0))
	require.Equal(t, float64(1.25), *mergeAvg(nil, 0, pointer.Float64(1.25), 5))

	require.Equal(t, float64(1.25), *mergeAvg(pointer.Float64(1), 3, pointer.Float64(2), 1))
	require.Equal(t, float64(1.25), *mergeAvg(pointer.Float64(2), 1, pointer.Float64(1), 3))
}

func TestRegressionStatsMerge(t *testing.T) {
	require.Equal(t, RegressionStats{}, RegressionStats{}.Merge(RegressionStats{}))
	require.Equal(t, RegressionStats{Min: pointer.Float64(1)}, RegressionStats{Min: pointer.Float64(1)}.Merge(RegressionStats{}))
	require.Equal(t, RegressionStats{Min: pointer.Float64(1)}, RegressionStats{}.Merge(RegressionStats{Min: pointer.Float64(1)}))
	require.Equal(t, RegressionStats{Min: pointer.Float64(1)}, RegressionStats{Min: pointer.Float64(2)}.Merge(RegressionStats{Min: pointer.Float64(1)}))
	require.Equal(t, RegressionStats{Min: pointer.Float64(1)}, RegressionStats{Min: pointer.Float64(1)}.Merge(RegressionStats{Min: pointer.Float64(2)}))

	require.Equal(t, RegressionStats{Max: pointer.Float64(1)}, RegressionStats{Max: pointer.Float64(1)}.Merge(RegressionStats{}))
	require.Equal(t, RegressionStats{Max: pointer.Float64(1)}, RegressionStats{}.Merge(RegressionStats{Max: pointer.Float64(1)}))
	require.Equal(t, RegressionStats{Max: pointer.Float64(2)}, RegressionStats{Max: pointer.Float64(2)}.Merge(RegressionStats{Max: pointer.Float64(1)}))
	require.Equal(t, RegressionStats{Max: pointer.Float64(2)}, RegressionStats{Max: pointer.Float64(1)}.Merge(RegressionStats{Max: pointer.Float64(2)}))

	left := RegressionStats{
		Max:         pointer.Float64(5),
		Min:         pointer.Float64(2),
		Avg:         pointer.Float64(3.5),
		SampleCount: 4,
	}

	right := RegressionStats{
		Max:         pointer.Float64(6),
		Min:         pointer.Float64(1),
		Avg:         pointer.Float64(3.5),
		SampleCount: 6,
	}

	merged := RegressionStats{
		Max:         pointer.Float64(6),
		Min:         pointer.Float64(1),
		Avg:         pointer.Float64(3.5),
		SampleCount: 10,
	}

	require.Equal(t, merged, left.Merge(right))
	require.Equal(t, merged, right.Merge(left))
}

func TestNetworkStatsMerge(t *testing.T) {
	require.Equal(t, NetworkStats{}, NetworkStats{}.Merge(NetworkStats{}))

	right := NetworkStats{
		Code2XX: 3,
		Code4XX: 4,
		Code5XX: 5,
		Latency: pointer.Float64(30),
		Total:   12,
	}

	left := NetworkStats{
		Code2XX: 1,
		Code4XX: 3,
		Code5XX: 4,
		Latency: pointer.Float64(5),
		Total:   8,
	}

	merged := NetworkStats{
		Code2XX: 4,
		Code4XX: 7,
		Code5XX: 9,
		Latency: pointer.Float64(20),
		Total:   20,
	}

	require.Equal(t, merged, left.Merge(right))
	require.Equal(t, merged, right.Merge(left))
}

func TestAPIMetricsMerge(t *testing.T) {
	require.Equal(t, Metrics{}, Metrics{}.Merge(Metrics{}))

	classDistribution := map[string]int{
		"class_a": 1,
		"class_b": 2,
		"class_c": 4,
	}

	networkStats := NetworkStats{
		Code2XX: 3,
		Code4XX: 4,
		Code5XX: 5,
		Latency: pointer.Float64(30),
		Total:   12,
	}

	regressionStats := RegressionStats{
		Max:         pointer.Float64(6),
		Min:         pointer.Float64(1),
		Avg:         pointer.Float64(3.5),
		SampleCount: 6,
	}

	mergedNetworkStats := networkStats.Merge(networkStats)
	mergedRegressionStats := regressionStats.Merge(regressionStats)

	mergedClassDistribution := map[string]int{
		"class_a": 2,
		"class_b": 4,
		"class_c": 8,
	}

	require.Equal(t, Metrics{ClassDistribution: classDistribution}, Metrics{ClassDistribution: classDistribution}.Merge(Metrics{}))
	require.Equal(t, Metrics{ClassDistribution: classDistribution}, Metrics{}.Merge(Metrics{ClassDistribution: classDistribution}))

	mergedAPIMetrics := Metrics{
		ClassDistribution: mergedClassDistribution,
		NetworkStats:      &mergedNetworkStats,
		RegressionStats:   &mergedRegressionStats,
	}

	apiMetrics := Metrics{
		ClassDistribution: classDistribution,
		NetworkStats:      &networkStats,
		RegressionStats:   &regressionStats,
	}

	require.Equal(t, mergedAPIMetrics, apiMetrics.Merge(apiMetrics))
}
