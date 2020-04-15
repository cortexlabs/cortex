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

type Code int

const (
	Unknown Code = iota
	Stalled
	Error
	OOM
	Live
	Updating
	VersionMismatch
)

var _codes = []string{
	"status_unknown",
	"status_stalled",
	"status_error",
	"status_oom",
	"status_live",
	"status_updating",
	"status_version_mismatch", // local only
}

var _ = [1]int{}[int(VersionMismatch)-(len(_codes)-1)] // Ensure list length matches

var _codeMessages = []string{
	"unknown",                  // Unknown
	"compute unavailable",      // Stalled
	"error",                    // Error
	"error (out of memory)",    // OOM
	"live",                     // Live
	"updating",                 // Updating
	"error (version mismatch)", // cli version not equal to api spec version
}

var _ = [1]int{}[int(VersionMismatch)-(len(_codeMessages)-1)] // Ensure list length matches

func (code Code) String() string {
	if int(code) < 0 || int(code) >= len(_codes) {
		return _codes[Unknown]
	}
	return _codes[code]
}

func (code Code) Message() string {
	if int(code) < 0 || int(code) >= len(_codeMessages) {
		return _codeMessages[Unknown]
	}
	return _codeMessages[code]
}
