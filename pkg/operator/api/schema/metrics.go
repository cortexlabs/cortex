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

package schema

import (
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

type RegressionStats struct {
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
	Avg         *float64 `json:"avg"`
	SampleCount int      `json:"sample_count"`
}

func (left RegressionStats) Merge(right RegressionStats) RegressionStats {
	totalSampleCount := left.SampleCount + right.SampleCount

	return RegressionStats{
		Min:         slices.Float64PtrMin(left.Min, right.Min),
		Max:         slices.Float64PtrMax(left.Max, right.Max),
		Avg:         mergeAvg(left.Avg, left.SampleCount, right.Avg, right.SampleCount),
		SampleCount: totalSampleCount,
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

type APIMetrics struct {
	NetworkStats      *NetworkStats    `json:"network_stats"`
	ClassDistribution map[string]int   `json:"class_distribution"`
	RegressionStats   *RegressionStats `json:"regression_stats"`
}

func (left APIMetrics) Merge(right APIMetrics) APIMetrics {
	mergedClassDistribution := left.ClassDistribution

	if right.ClassDistribution != nil {
		if left.ClassDistribution == nil {
			mergedClassDistribution = right.ClassDistribution
		} else {
			for className, count := range right.ClassDistribution {
				if _, ok := mergedClassDistribution[className]; ok {
					mergedClassDistribution[className] += count
				} else {
					mergedClassDistribution[className] = count
				}
			}
		}
	}

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

	var mergedRegressionStats *RegressionStats
	switch {
	case left.RegressionStats != nil && right.RegressionStats != nil:
		merged := (*left.RegressionStats).Merge(*right.RegressionStats)
		mergedRegressionStats = &merged
	case left.RegressionStats != nil:
		mergedRegressionStats = left.RegressionStats
	case right.RegressionStats != nil:
		mergedRegressionStats = right.RegressionStats
	}

	return APIMetrics{
		NetworkStats:      mergedNetworkStats,
		RegressionStats:   mergedRegressionStats,
		ClassDistribution: mergedClassDistribution,
	}
}

func mergeAvg(left *float64, leftCount int, right *float64, rightCount int) *float64 {
	leftCountFloat64Ptr := pointer.Float64(float64(leftCount))
	rightCountFloat64Ptr := pointer.Float64(float64(rightCount))

	avg, _ := slices.Float64PtrAvg([]*float64{left, right}, []*float64{leftCountFloat64Ptr, rightCountFloat64Ptr})
	return avg
}
