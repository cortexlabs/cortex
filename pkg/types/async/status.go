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

package async

// Status is an enum type for workload status
type Status string

// Different possible workload status
const (
	StatusNotFound   Status = "not_found"
	StatusFailed     Status = "failed"
	StatusInProgress Status = "in_progress"
	StatusInQueue    Status = "in_queue"
	StatusCompleted  Status = "completed"
)

func (status Status) String() string {
	return string(status)
}

func (status Status) Valid() bool {
	switch status {
	case StatusNotFound, StatusFailed, StatusInProgress, StatusInQueue, StatusCompleted:
		return true
	default:
		return false
	}
}
