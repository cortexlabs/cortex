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

package status

type JobCode int

const (
	JobUnknown JobCode = iota
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
)

var _jobCodes = []string{
	"status_unknown",
	"status_enqueuing",
	"status_running",
	"status_enqueue_failed",
	"status_completed_with_failures",
	"status_succeeded",
	"status_unexpected_error",
	"status_worker_error",
	"status_worker_oom",
	"status_timed_out",
	"status_stopped",
}

var _ = [1]int{}[int(JobStopped)-(len(_jobCodes)-1)] // Ensure list length matches

var _jobCodeMessages = []string{
	"unknown",
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
}

var _ = [1]int{}[int(JobStopped)-(len(_jobCodeMessages)-1)] // Ensure list length matches

func (code JobCode) IsInProgress() bool {
	return code == JobEnqueuing || code == JobRunning
}

func (code JobCode) IsCompleted() bool {
	return code == JobEnqueueFailed || code == JobCompletedWithFailures || code == JobSucceeded || code == JobUnexpectedError || code == JobWorkerError || code == JobWorkerOOM || code == JobStopped || code == JobTimedOut
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
