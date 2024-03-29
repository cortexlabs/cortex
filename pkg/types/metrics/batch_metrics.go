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

package metrics

import (
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

type BatchMetrics struct {
	Succeeded           int      `json:"succeeded"`
	Failed              int      `json:"failed"`
	AverageTimePerBatch *float64 `json:"average_time_per_batch"`
}

func (batchMetrics BatchMetrics) Merge(right BatchMetrics) BatchMetrics {
	newBatchMetrics := BatchMetrics{}
	newBatchMetrics.MergeInPlace(batchMetrics)
	newBatchMetrics.MergeInPlace(right)
	return newBatchMetrics
}

func (batchMetrics *BatchMetrics) MergeInPlace(right BatchMetrics) {
	batchMetrics.AverageTimePerBatch = mergeAvg(batchMetrics.AverageTimePerBatch, batchMetrics.Succeeded, right.AverageTimePerBatch, right.Succeeded)
	batchMetrics.Succeeded = batchMetrics.Succeeded + right.Succeeded
	batchMetrics.Failed = batchMetrics.Failed + right.Failed
}

func mergeAvg(left *float64, leftCount int, right *float64, rightCount int) *float64 {
	leftCountFloat64Ptr := pointer.Float64(float64(leftCount))
	rightCountFloat64Ptr := pointer.Float64(float64(rightCount))

	avg, _ := slices.Float64PtrAvg([]*float64{left, right}, []*float64{leftCountFloat64Ptr, rightCountFloat64Ptr})
	return avg
}
