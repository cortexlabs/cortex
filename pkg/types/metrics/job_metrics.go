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
	Succeeded           int      `json:"succeeded"`
	Failed              int      `json:"failed"`
	AverageTimePerBatch *float64 `json:"average_time_per_batch"`
	TotalCompleted      int      `json:"total_completed"`
}

func (left JobMetrics) Merge(right JobMetrics) JobMetrics {
	newJobMetrics := JobMetrics{}
	newJobMetrics.MergeInPlace(left)
	newJobMetrics.MergeInPlace(right)
	return newJobMetrics
}

func (left *JobMetrics) MergeInPlace(right JobMetrics) {
	left.AverageTimePerBatch = mergeAvg(left.AverageTimePerBatch, left.TotalCompleted, right.AverageTimePerBatch, right.TotalCompleted)
	left.Succeeded = left.Succeeded + right.Succeeded
	left.Failed = left.Failed + right.Failed
	left.TotalCompleted = left.TotalCompleted + right.TotalCompleted
}
