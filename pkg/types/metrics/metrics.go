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
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

type Metrics struct {
	APIName      string        `json:"api_name"`
	NetworkStats *NetworkStats `json:"network_stats"`
}

func (left Metrics) Merge(right Metrics) Metrics {
	var mergedNetworkStats *NetworkStats
	switch {
	case left.NetworkStats != nil && right.NetworkStats != nil:
		merged := (*left.NetworkStats).Merge(*right.NetworkStats)
		mergedNetworkStats = &merged
	case left.NetworkStats != nil:
		mergedNetworkStats = left.NetworkStats
	case right.NetworkStats != nil:
		mergedNetworkStats = right.NetworkStats
	}

	return Metrics{
		NetworkStats: mergedNetworkStats,
	}
}

type NetworkStats struct {
	Latency *float64 `json:"latency"`
	Code2XX int      `json:"code_2xx"`
	Code4XX int      `json:"code_4xx"`
	Code5XX int      `json:"code_5xx"`
	Total   int      `json:"total"`
}

func (left NetworkStats) Merge(right NetworkStats) NetworkStats {
	return NetworkStats{
		Latency: mergeAvg(left.Latency, left.Total, right.Latency, right.Total),
		Code2XX: left.Code2XX + right.Code2XX,
		Code4XX: left.Code4XX + right.Code4XX,
		Code5XX: left.Code5XX + right.Code5XX,
		Total:   left.Total + right.Total,
	}
}

func mergeAvg(left *float64, leftCount int, right *float64, rightCount int) *float64 {
	leftCountFloat64Ptr := pointer.Float64(float64(leftCount))
	rightCountFloat64Ptr := pointer.Float64(float64(rightCount))

	avg, _ := slices.Float64PtrAvg([]*float64{left, right}, []*float64{leftCountFloat64Ptr, rightCountFloat64Ptr})
	return avg
}
