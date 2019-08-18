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

type RegressionStats struct {
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
	Avg         *float64 `json:"avg"`
	SampleCount int      `json:"sample_count"`
}

func (metricsLeft RegressionStats) Merge(metricsRight RegressionStats) RegressionStats {
	var mergedMin *float64
	switch {
	case metricsLeft.Min != nil && metricsRight.Min != nil:
		mergedMin = metricsLeft.Min
		if *metricsRight.Min < *metricsLeft.Min {
			mergedMin = metricsRight.Min
		}
	case metricsLeft.Min != nil:
		mergedMin = metricsLeft.Min
	case metricsRight.Min != nil:
		mergedMin = metricsRight.Min
	}

	var mergedMax *float64
	switch {
	case metricsLeft.Max != nil && metricsRight.Max != nil:
		mergedMax = metricsLeft.Max
		if *metricsRight.Max > *metricsLeft.Max {
			mergedMax = metricsRight.Max
		}
	case metricsLeft.Max != nil:
		mergedMax = metricsLeft.Max
	case metricsRight.Max != nil:
		mergedMax = metricsRight.Max
	}

	totalSampleCount := metricsLeft.SampleCount + metricsRight.SampleCount

	return RegressionStats{
		Min:         mergedMin,
		Max:         mergedMax,
		Avg:         mergeAvg(metricsLeft.Avg, metricsRight.Avg, metricsLeft.SampleCount, metricsRight.SampleCount),
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

func (metricsLeft NetworkStats) Merge(metricsRight NetworkStats) NetworkStats {
	return NetworkStats{
		Latency: mergeAvg(metricsLeft.Latency, metricsRight.Latency, metricsLeft.Total, metricsRight.Total),
		Code2XX: metricsLeft.Code2XX + metricsRight.Code2XX,
		Code4XX: metricsLeft.Code4XX + metricsRight.Code4XX,
		Code5XX: metricsLeft.Code5XX + metricsRight.Code5XX,
		Total:   metricsLeft.Total + metricsRight.Total,
	}
}

type APIMetrics struct {
	NetworkStats      *NetworkStats    `json:"network_stats"`
	ClassDistribution map[string]int   `json:"class_distribution"`
	RegressionStats   *RegressionStats `json:"regression_stats"`
}

func (metricsLeft APIMetrics) Merge(metricsRight APIMetrics) APIMetrics {
	mergedClassDistribution := metricsLeft.ClassDistribution

	if mergedClassDistribution == nil && metricsRight.ClassDistribution != nil {
		mergedClassDistribution = make(map[string]int, len(metricsRight.ClassDistribution))
	}

	for className, count := range metricsRight.ClassDistribution {
		if _, ok := mergedClassDistribution[className]; ok {
			mergedClassDistribution[className] += count
		} else {
			mergedClassDistribution[className] = count
		}
	}

	var mergedNetworkStats *NetworkStats
	switch {
	case metricsLeft.NetworkStats != nil && metricsRight.NetworkStats != nil:
		merged := (*metricsLeft.NetworkStats).Merge(*metricsRight.NetworkStats)
		mergedNetworkStats = &merged
	case metricsLeft.NetworkStats != nil:
		mergedNetworkStats = metricsLeft.NetworkStats
	case metricsRight.NetworkStats != nil:
		mergedNetworkStats = metricsRight.NetworkStats
	}

	var mergedRegressionStats *RegressionStats
	switch {
	case metricsLeft.RegressionStats != nil && metricsRight.RegressionStats != nil:
		merged := (*metricsLeft.RegressionStats).Merge(*metricsRight.RegressionStats)
		mergedRegressionStats = &merged
	case metricsLeft.RegressionStats != nil:
		mergedRegressionStats = metricsLeft.RegressionStats
	case metricsRight.RegressionStats != nil:
		mergedRegressionStats = metricsRight.RegressionStats
	}

	return APIMetrics{
		NetworkStats:      mergedNetworkStats,
		RegressionStats:   mergedRegressionStats,
		ClassDistribution: mergedClassDistribution,
	}
}

func mergeAvg(left *float64, right *float64, leftCount int, rightCount int) *float64 {
	total := 0

	if left != nil {
		total += leftCount
	}

	if right != nil {
		total += rightCount
	}

	if total == 0 {
		return nil
	}

	var mergedAvg float64

	if left != nil {
		mergedAvg += (*left) * float64(leftCount) / float64(total)
	}

	if right != nil {
		mergedAvg += (*right) * float64(rightCount) / float64(total)
	}

	return &mergedAvg
}
