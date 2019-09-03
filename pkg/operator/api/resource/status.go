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
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
)

type DataStatus struct {
	DataSavedStatus
	Code StatusCode `json:"status_code"`
}

// There is one APIStatus per API resource ID (including stale/removed models). There is always an APIStatus for APIs currently in the context.
type APIStatus struct {
	APISavedStatus
	Path                 string `json:"path"`
	MinReplicas          int32  `json:"min_replicas"`
	MaxReplicas          int32  `json:"max_replicas"`
	InitReplicas         int32  `json:"init_replicas"`
	TargetCPUUtilization int32  `json:"target_cpu_utilization"`
	ReplicaCounts        `json:"replica_counts"`
	PodStatuses          []k8s.PodStatus `json:"pod_statuses"`
	Code                 StatusCode      `json:"status_code"`
}

type ReplicaCounts struct {
	ReadyUpdatedCompute  int32 `json:"ready_updated_compute"`
	ReadyStaleCompute    int32 `json:"ready_stale_compute"`
	FailedUpdatedCompute int32 `json:"failed_updated_compute"`
	FailedStaleCompute   int32 `json:"failed_stale_compute"`
	K8sRequested         int32 `json:"k8s_requested"` // Number of requested replicas in an active k8s.deployment for this resource ID
}

// There is one APIGroupStatus per API name/endpoint
type APIGroupStatus struct {
	APIName              string     `json:"api_name"`
	ActiveStatus         *APIStatus `json:"active_status"` // The most recently ready API status, or the ctx API status if it's ready
	Code                 StatusCode `json:"status_code"`
	GroupedReplicaCounts `json:"grouped_replica_counts"`
}

type GroupedReplicaCounts struct {
	ReadyUpdated       int32 `json:"ready_updated"`       // Updated means the replica is fully up-to-date (compute and model match the API's current resource ID in the context)
	ReadyStaleModel    int32 `json:"ready_stale_model"`   // Stale model means the replica is serving a model which not currently in the context (either it was updated or removed)
	ReadyStaleCompute  int32 `json:"ready_stale_compute"` // Stale compute means the replica is serving the correct model, but the compute request has changed
	FailedUpdated      int32 `json:"failed_updated"`
	FailedStaleModel   int32 `json:"failed_stale_model"`
	FailedStaleCompute int32 `json:"failed_stale_compute"`
	Requested          int32 `json:"requested"`
}

type Status interface {
	Message() string
	GetCode() StatusCode
}

func (rc *ReplicaCounts) TotalReady() int32 {
	return rc.ReadyUpdatedCompute + rc.ReadyStaleCompute
}

func (rc *ReplicaCounts) TotalFailed() int32 {
	return rc.FailedUpdatedCompute + rc.FailedStaleCompute
}

func (grc *GroupedReplicaCounts) Available() int32 {
	return grc.ReadyUpdated + grc.ReadyStaleModel + grc.ReadyStaleCompute
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
	StatusError
	StatusParentFailed
	StatusParentKilled
	StatusKilledOOM

	// Data statuses
	StatusRunning
	StatusSucceeded
	StatusKilled

	// API statuses
	StatusCreating
	StatusLive
	StatusStopping
	StatusStopped
)

var statusCodes = []string{
	"status_unknown",

	"status_pending",
	"status_pending_compute",
	"status_waiting",
	"status_skipped",
	"status_error",
	"status_parent_failed",
	"status_parent_killed",
	"status_killed_oom",

	"status_running",
	"status_succeeded",
	"status_killed",

	"status_creating",
	"status_live",
	"status_stopping",
	"status_stopped",
}

var _ = [1]int{}[int(StatusStopped)-(len(statusCodes)-1)] // Ensure list length matches

var statusCodeMessages = []string{
	"unknown", // StatusUnknown

	"pending",                    // StatusPending
	"compute unavailable",        // StatusPendingCompute
	"pending",                    // StatusWaiting
	"skipped",                    // StatusSkipped
	"error",                      // StatusError
	"upstream error",             // StatusParentFailed
	"upstream termination",       // StatusParentKilled
	"terminated (out of memory)", // StatusDataOOM

	"running",    // StatusRunning
	"ready",      // StatusSucceeded
	"terminated", // StatusKilled

	"creating", // StatusCreating
	"live",     // StatusLive
	"stopping", // StatusStopping
	"stopped",  // StatusStopped
}

var _ = [1]int{}[int(StatusStopped)-(len(statusCodeMessages)-1)] // Ensure list length matches

var statusSortBuckets = []int{
	999, // StatusUnknown

	4, // StatusPending
	4, // StatusPendingCompute
	4, // StatusWaiting
	2, // StatusSkipped
	1, // StatusError
	2, // StatusParentFailed
	2, // StatusParentKilled
	1, // StatusKilledOOM

	3, // StatusRunning
	0, // StatusSucceeded
	1, // StatusKilled

	3, // StatusCreating
	0, // StatusLive
	3, // StatusStopping
	1, // StatusStopped
}

var _ = [1]int{}[int(StatusStopped)-(len(statusSortBuckets)-1)] // Ensure list length matches

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
	return status.Code.Message()
}

func (status *APIStatus) Message() string {
	return status.Code.Message()
}

func (status *APIGroupStatus) Message() string {
	return status.Code.Message()
}

// MarshalText satisfies TextMarshaler
func (code StatusCode) MarshalText() ([]byte, error) {
	return []byte(code.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (code *StatusCode) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(statusCodes); i++ {
		if enum == statusCodes[i] {
			*code = StatusCode(i)
			return nil
		}
	}

	*code = StatusUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (code *StatusCode) UnmarshalBinary(data []byte) error {
	return code.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (code StatusCode) MarshalBinary() ([]byte, error) {
	return []byte(code.String()), nil
}
