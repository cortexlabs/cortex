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

package resource

import (
	"time"
)

type DataStatus struct {
	DataSavedStatus
	Code StatusCode `json:"status_code"`
}

type APIStatus struct {
	APISavedStatus
	Path              string `json:"path"`
	RequestedReplicas int32  `json:"requested_replicas"`
	ReplicaCounts     `json:"replica_counts"`
	Code              StatusCode `json:"status_code"`
}

type ReplicaCounts struct {
	ReadyUpdated        int32 `json:"ready_updated"`
	ReadyStaleCompute   int32 `json:"ready_stale_compute"`
	ReadyStaleResource  int32 `json:"ready_stale_resource"`
	FailedUpdated       int32 `json:"failed_updated"`
	FailedStaleCompute  int32 `json:"failed_stale_compute"`
	FailedStaleResource int32 `json:"failed_stale_resource"`
}

type APIGroupStatus struct {
	APIName      string     `json:"api_name"`
	Start        *time.Time `json:"start"`
	ActiveStatus *APIStatus `json:"active_status"`
	Code         StatusCode `json:"status_code"`
}

type Status interface {
	Message() string
	GetCode() StatusCode
}

func (replicaCounts *ReplicaCounts) TotalReady() int32 {
	return replicaCounts.ReadyUpdated + replicaCounts.ReadyStaleCompute + replicaCounts.ReadyStaleResource
}

func (replicaCounts *ReplicaCounts) TotalStaleReady() int32 {
	return replicaCounts.ReadyStaleCompute + replicaCounts.ReadyStaleResource
}

func (replicaCounts *ReplicaCounts) TotalStale() int32 {
	return replicaCounts.ReadyStaleCompute + replicaCounts.ReadyStaleResource + replicaCounts.FailedStaleCompute + replicaCounts.FailedStaleResource
}

func (status *DataStatus) GetCode() StatusCode {
	return status.Code
}

func (status *APIStatus) GetCode() StatusCode {
	return status.Code
}

func (status *APIGroupStatus) GetCode() StatusCode {
	return status.Code
}

type StatusCode int

const (
	StatusUnknown StatusCode = iota

	// Shared statuses
	StatusPending // Resource is pending other non-ready resources
	StatusPendingCompute
	StatusWaiting // Resource can be created based on resource DAG, but hasn't started yet
	StatusSkipped
	StatusParentFailed
	StatusParentKilled

	// Data statuses
	StatusDataRunning
	StatusDataSucceeded
	StatusDataFailed
	StatusDataKilled

	// API statuses
	StatusAPIUpdating
	StatusAPIReady
	StatusAPIStopping
	StatusAPIStopped
	StatusAPIError

	// Additional API group statuses (i.e. aggregated API status)
	StatusAPIGroupPendingUpdate
	StatusAPIGroupParentFailed
	StatusAPIGroupParentKilled
	StatusAPIGroupUpdateSkipped
)

var statusCodes = []string{
	"status_unknown",

	"status_pending",
	"status_pending_compute",
	"status_waiting",
	"status_skipped",
	"status_parent_failed",
	"status_parent_killed",

	"status_data_running",
	"status_data_succeeded",
	"status_data_failed",
	"status_data_killed",

	"status_api_updating",
	"status_api_ready",
	"status_api_stopping",
	"status_api_stopped",
	"status_api_error",

	"status_api_group_pending_update",
	"status_api_group_parent_failed",
	"status_api_group_parent_killed",
	"status_api_group_update_skipped",
}

var _ = [1]int{}[int(StatusAPIGroupUpdateSkipped)-(len(statusCodes)-1)] // Ensure list length matches

var statusCodeMessages = []string{
	"unknown", // StatusUnknown

	"pending",              // StatusPending
	"compute unavailable",  // StatusPendingCompute
	"pending",              // StatusWaiting
	"skipped",              // StatusSkipped
	"upstream error",       // StatusParentFailed
	"upstream termination", // StatusParentKilled

	"running",    // StatusDataRunning
	"ready",      // StatusDataSucceeded
	"error",      // StatusDataFailed
	"terminated", // StatusDataKilled

	"updating", // 	StatusAPIUpdating
	"ready",    // StatusAPIReady
	"stopping", // StatusAPIStopping
	"stopped",  // StatusAPIStopped
	"error",    // StatusAPIError

	"update pending",       // StatusAPIGroupPendingUpdate
	"upstream error",       // StatusAPIGroupParentFailed
	"upstream termination", // StatusAPIGroupParentKilled
	"update skipped",       // StatusAPIGroupUpdateSkipped
}

var _ = [1]int{}[int(StatusAPIGroupUpdateSkipped)-(len(statusCodeMessages)-1)] // Ensure list length matches

// StatusDataRunning aliases
const (
	RawColumnRunningMessage       = "ingesting"
	AggregatorRunningMessage      = "aggregating"
	TransformerRunningMessage     = "transforming"
	TrainingDatasetRunningMessage = "generating"
	ModelRunningMessage           = "training"
)

var statusSortBuckets = []int{
	999, // StatusUnknown

	4, // StatusPending
	4, // StatusPendingCompute
	4, // StatusWaiting
	2, // StatusSkipped
	2, // StatusParentFailed
	2, // StatusParentKilled

	3, // StatusDataRunning
	0, // StatusDataSucceeded
	1, // StatusDataFailed
	1, // StatusDataKilled

	3, // 	StatusAPIUpdating
	0, // StatusAPIReady
	3, // StatusAPIStopping
	1, // StatusAPIStopped
	1, // StatusAPIError

	0, // StatusAPIGroupPendingUpdate
	2, // StatusAPIGroupParentFailed
	2, // StatusAPIGroupParentKilled
	2, // StatusAPIGroupUpdateSkipped
}

var _ = [1]int{}[int(StatusAPIGroupUpdateSkipped)-(len(statusSortBuckets)-1)] // Ensure list length matches

func (code StatusCode) String() string {
	if int(code) < 0 || int(code) >= len(statusCodes) {
		return statusCodes[StatusUnknown]
	}
	return statusCodes[code]
}

func (code StatusCode) Message() string {
	if int(code) < 0 || int(code) >= len(statusCodeMessages) {
		return statusCodeMessages[StatusUnknown]
	}
	return statusCodeMessages[code]
}

func (code StatusCode) SortBucket() int {
	if int(code) < 0 || int(code) >= len(statusSortBuckets) {
		return statusSortBuckets[StatusUnknown]
	}
	return statusSortBuckets[code]
}

func (status *DataStatus) Message() string {
	if status.Code == StatusDataRunning {
		switch status.ResourceType {
		case RawColumnType:
			return RawColumnRunningMessage
		case AggregateType:
			return AggregatorRunningMessage
		case TransformedColumnType:
			return TransformerRunningMessage
		case TrainingDatasetType:
			return TrainingDatasetRunningMessage
		case ModelType:
			return ModelRunningMessage
		}
	}
	return status.Code.Message()
}

func (status *APIStatus) Message() string {
	return status.Code.Message()
}

func (status *APIGroupStatus) Message() string {
	return status.Code.Message()
}

// MarshalText satisfies TextMarshaler
func (k StatusCode) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (k *StatusCode) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(statusCodes); i++ {
		if enum == statusCodes[i] {
			*k = StatusCode(i)
			return nil
		}
	}

	*k = StatusUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (k *StatusCode) UnmarshalBinary(data []byte) error {
	return k.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (k StatusCode) MarshalBinary() ([]byte, error) {
	return []byte(k.String()), nil
}
