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

package status

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
)

// There is one Status per API resource ID (including stale/removed models)
type Status struct {
	APIName              string     `json:"api_name"`
	ResourceID           string     `json:"resource_id"`
	WorkloadID           string     `json:"workload_id"`
	Start                *time.Time `json:"start"`
	MinReplicas          int32      `json:"min_replicas"`
	MaxReplicas          int32      `json:"max_replicas"`
	InitReplicas         int32      `json:"init_replicas"`
	TargetCPUUtilization int32      `json:"target_cpu_utilization"`
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

// There is one GroupStatus per API name/endpoint
type GroupStatus struct {
	APIName              string     `json:"api_name"`
	ActiveStatus         *Status    `json:"active_status"` // The most recently ready API status, or the ctx API status if it's ready
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

func (rc *ReplicaCounts) TotalReady() int32 {
	return rc.ReadyUpdatedCompute + rc.ReadyStaleCompute
}

func (rc *ReplicaCounts) TotalFailed() int32 {
	return rc.FailedUpdatedCompute + rc.FailedStaleCompute
}

func (grc *GroupedReplicaCounts) Available() int32 {
	return grc.ReadyUpdated + grc.ReadyStaleModel + grc.ReadyStaleCompute
}

func (grc *GroupedReplicaCounts) ReadyStale() int32 {
	return grc.ReadyStaleModel + grc.ReadyStaleCompute
}

type StatusCode int

const (
	StatusUnknown StatusCode = iota
	StatusPendingCompute
	StatusError
	StatusKilledOOM
	StatusLive
	StatusUpdating
	StatusStopping
	StatusStopped
)

var statusCodes = []string{
	"status_unknown",
	"status_pending_compute",
	"status_error",
	"status_killed_oom",
	"status_live",
	"status_updating",
	"status_stopping",
	"status_stopped",
}

var _ = [1]int{}[int(StatusStopped)-(len(statusCodes)-1)] // Ensure list length matches

var statusCodeMessages = []string{
	"unknown",               // StatusUnknown
	"compute unavailable",   // StatusPendingCompute
	"error",                 // StatusError
	"error (out of memory)", // StatusKilledOOM
	"live",                  // StatusLive
	"updating",              // StatusUpdating
	"stopping",              // StatusStopping
	"stopped",               // StatusStopped
}

var _ = [1]int{}[int(StatusStopped)-(len(statusCodeMessages)-1)] // Ensure list length matches

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

func (status *Status) Message() string {
	return status.Code.Message()
}

func (status *GroupStatus) Message() string {
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
