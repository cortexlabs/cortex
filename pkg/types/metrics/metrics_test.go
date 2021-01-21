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

	networkStats := NetworkStats{
		Code2XX: 3,
		Code4XX: 4,
		Code5XX: 5,
		Latency: pointer.Float64(30),
		Total:   12,
	}

	mergedNetworkStats := networkStats.Merge(networkStats)

	mergedAPIMetrics := Metrics{
		NetworkStats: &mergedNetworkStats,
	}

	apiMetrics := Metrics{
		NetworkStats: &networkStats,
	}

	require.Equal(t, mergedAPIMetrics, apiMetrics.Merge(apiMetrics))
}
