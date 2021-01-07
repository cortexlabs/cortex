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

// There could be Cortex specific messages in queue
type QueueMetrics struct {
	Visible    int `json:"visible"`
	NotVisible int `json:"not_visible"`
}

func (q QueueMetrics) IsEmpty() bool {
	return q.TotalInQueue() == 0
}

func (q QueueMetrics) TotalInQueue() int {
	return q.Visible + q.NotVisible
}

func (q QueueMetrics) TotalUserMessages() int {
	total := q.TotalInQueue()
	if total == 0 {
		return 0
	}

	return total - 1 // An extra message is added
}
