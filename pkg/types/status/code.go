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

var _ = [1]int{}[int(Stopping)-(len(codes)-1)] // Ensure list length matches

var codeMessages = []string{
	"unknown",               // Unknown
	"compute unavailable",   // Stalled
	"error",                 // Error
	"error (out of memory)", // OOM
	"live",                  // Live
	"updating",              // Updating
	"stopping",              // Stopping
}

var _ = [1]int{}[int(Stopping)-(len(codeMessages)-1)] // Ensure list length matches

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
