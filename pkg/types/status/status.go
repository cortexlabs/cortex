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

type Status struct {
	APIName       string `json:"api_name"`
	Code          Code   `json:"status_code"`
	ReplicaCounts `json:"replica_counts"`
}

type ReplicaCounts struct {
	Updated   SubReplicaCounts `json:"updated"` // fully up-to-date (compute and model)
	Stale     SubReplicaCounts `json:"stale"`
	Requested int32            `json:"requested"`
}

type SubReplicaCounts struct {
	Pending      int32 `json:"pending"`
	Initializing int32 `json:"initializing"`
	Ready        int32 `json:"ready"`
	Terminating  int32 `json:"terminating"`
	Failed       int32 `json:"failed"`
	Killed       int32 `json:"killed"`
	KilledOOM    int32 `json:"killed_oom"`
	Stalled      int32 `json:"stalled"` // pending for a long time
	Unknown      int32 `json:"unknown"`
}

// func (rc *ReplicaCounts) TotalAvailable() int32 {
// 	return rc.ReadyUpdated + rc.ReadyStaleModel + rc.ReadyStaleCompute
// }

// func (rc *ReplicaCounts) TotalFailed() int32 {
// 	return rc.FailedUpdated + rc.FailedStaleModel + rc.FailedStaleCompute +
// 	rc.KilledUpdated + rc.KilledStaleModel + rc.KilledStaleCompute +
// 	rc.KilledOOMUpdated + rc.KilledOOMStaleModel + rc.KilledOOMStaleCompute +
// }

// func (rc *ReplicaCounts) TotalAvailableStale() int32 {
// 	return rc.ReadyStaleModel + rc.ReadyStaleCompute
// }

type Code int

const (
	Unknown Code = iota
	Stalled
	Error
	OOM
	Live
	Updating
	Stopping
)

var codes = []string{
	"status_unknown",
	"status_stalled",
	"status_error",
	"status_oom",
	"status_live",
	"status_updating",
	"status_stopping",
}

var _ = [1]int{}[int(Stopped)-(len(codes)-1)] // Ensure list length matches

var codeMessages = []string{
	"unknown",               // Unknown
	"compute unavailable",   // Stalled
	"error",                 // Error
	"error (out of memory)", // OOM
	"live",                  // Live
	"updating",              // Updating
	"stopping",              // Stopping
}

var _ = [1]int{}[int(StatusStopped)-(len(codeMessages)-1)] // Ensure list length matches

func (code Code) String() string {
	if int(code) < 0 || int(code) >= len(codes) {
		return codes[Unknown]
	}
	return codes[code]
}

func (code Code) Message() string {
	if int(code) < 0 || int(code) >= len(codeMessages) {
		return codeMessages[Unknown]
	}
	return codeMessages[code]
}

func (status *Status) Message() string {
	return status.Code.Message()
}

// MarshalText satisfies TextMarshaler
func (code Code) MarshalText() ([]byte, error) {
	return []byte(code.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (code *Code) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(codes); i++ {
		if enum == codes[i] {
			*code = Code(i)
			return nil
		}
	}

	*code = Unknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (code *Code) UnmarshalBinary(data []byte) error {
	return code.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (code Code) MarshalBinary() ([]byte, error) {
	return []byte(code.String()), nil
}
