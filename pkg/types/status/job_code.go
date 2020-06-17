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

package status

type JobCode int

const (
	JobUnknown JobCode = iota
	JobEnqueuing
	JobFailed
	JobRunning
	JobComplete
)

var _jobCodes = []string{
	"status_unknown",
	"status_enqueuing",
	"status_failed",
	"status_running",
	"status_complete",
}

var _ = [1]int{}[int(JobComplete)-(len(_jobCodes)-1)] // Ensure list length matches

var _jobCodeMessages = []string{
	"unknown",   // JobUnknown
	"enqueuing", // Job
	"failed",    // Failed
	"running",   // Running
	"complete",  // Complete
}

var _ = [1]int{}[int(JobComplete)-(len(_jobCodeMessages)-1)] // Ensure list length matches

func (code JobCode) String() string {
	if int(code) < 0 || int(code) >= len(_jobCodes) {
		return _jobCodes[Unknown]
	}
	return _jobCodes[code]
}

func (code JobCode) Message() string {
	if int(code) < 0 || int(code) >= len(_jobCodeMessages) {
		return _jobCodeMessages[Unknown]
	}
	return _jobCodeMessages[code]
}
