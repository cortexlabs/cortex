/*
Copyright 2020 Cortex Labs, Inc.

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

type JobMetrics struct {
	APIName  string    `json:"api_name"`
	JobID    string    `json:"job_id"`
	JobStats *JobStats `json:"job_stats"`
}

type JobStats struct {
	Succeeded               int      `json:"succeeded"`
	Failed                  int      `json:"failed"`
	AverageTimePerPartition *float64 `json:"average_time_per_partition"`
	TotalCompleted          int      `json:"total_completed"`
}

func (left JobMetrics) Merge(right JobMetrics) JobMetrics {
	var mergedJobStats *JobStats

	switch {
	case left.JobStats != nil && right.JobStats != nil:
		merged := (*left.JobStats).Merge(*right.JobStats)
		mergedJobStats = &merged
	case left.JobStats != nil:
		mergedJobStats = left.JobStats
	case right.JobStats != nil:
		mergedJobStats = right.JobStats
	}

	return JobMetrics{
		APIName:  left.APIName,
		JobID:    left.JobID,
		JobStats: mergedJobStats,
	}
}

func (left JobStats) Merge(right JobStats) JobStats {
	return JobStats{
		AverageTimePerPartition: mergeAvg(left.AverageTimePerPartition, left.TotalCompleted, right.AverageTimePerPartition, right.TotalCompleted),
		Succeeded:               left.Succeeded + right.Succeeded,
		Failed:                  left.Failed + right.Failed,
		TotalCompleted:          left.TotalCompleted + right.TotalCompleted,
	}
}
