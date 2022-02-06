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

package status

// JobCode is an enum to represent a job status
// +kubebuilder:validation:Type=string
type JobCode int

// Possible values for JobCode
const (
	JobPending JobCode = iota // pending should be the first status in this list
	JobEnqueuing
	JobRunning
	JobEnqueueFailed
	JobCompletedWithFailures
	JobSucceeded
	JobUnexpectedError
	JobWorkerError
	JobWorkerOOM
	JobTimedOut
	JobStopped
	JobUnknown
)

var _jobCodes = []string{
	"pending",
	"enqueuing",
	"running",
	"enqueue_failed",
	"completed_with_failures",
	"succeeded",
	"unexpected_error",
	"worker_error",
	"worker_oom",
	"timed_out",
	"stopped",
	"unknown",
}

var _ = [1]int{}[int(JobUnknown)-(len(_jobCodes)-1)] // Ensure list length matches

var _jobCodeMessages = []string{
	"pending",
	"enqueuing",
	"running",
	"failed while enqueuing",
	"completed with failures",
	"succeeded",
	"unexpected error",
	"worker error",
	"out of memory",
	"timed out",
	"stopped",
	"unknown",
}

var _ = [1]int{}[int(JobUnknown)-(len(_jobCodeMessages)-1)] // Ensure list length matches

func (code JobCode) IsNotStarted() bool {
	return code == JobPending || code == JobEnqueuing
}

func (code JobCode) IsInProgress() bool {
	return code == JobEnqueuing || code == JobRunning
}

func (code JobCode) IsCompleted() bool {
	return code == JobEnqueueFailed || code == JobCompletedWithFailures ||
		code == JobSucceeded || code == JobUnexpectedError ||
		code == JobWorkerError || code == JobWorkerOOM ||
		code == JobStopped || code == JobTimedOut
}

func (code JobCode) String() string {
	if int(code) < 0 || int(code) >= len(_jobCodes) {
		return _jobCodes[JobUnknown]
	}
	return _jobCodes[code]
}

func (code JobCode) Message() string {
	if int(code) < 0 || int(code) >= len(_jobCodeMessages) {
		return _jobCodeMessages[JobUnknown]
	}
	return _jobCodeMessages[code]
}

// MarshalText satisfies TextMarshaler
func (code JobCode) MarshalText() ([]byte, error) {
	return []byte(code.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (code *JobCode) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_jobCodes); i++ {
		if enum == _jobCodes[i] {
			*code = JobCode(i)
			return nil
		}
	}

	*code = JobUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (code *JobCode) UnmarshalBinary(data []byte) error {
	return code.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (code JobCode) MarshalBinary() ([]byte, error) {
	return []byte(code.String()), nil
}
