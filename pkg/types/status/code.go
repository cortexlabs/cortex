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

type Code int

const (
	Unknown Code = iota
	Stalled
	Error
	ErrorImagePull
	OOM
	Live
	Updating
)

var _codes = []string{
	"status_unknown",
	"status_stalled",
	"status_error",
	"status_error_image_pull",
	"status_oom",
	"status_live",
	"status_updating",
}

var _ = [1]int{}[int(Updating)-(len(_codes)-1)] // Ensure list length matches

var _codeMessages = []string{
	"unknown",               // Unknown
	"compute unavailable",   // Stalled
	"error",                 // Error
	"error (image pull)",    // Live
	"error (out of memory)", // OOM
	"live",                  // Live
	"updating",              // Updating
}

var _ = [1]int{}[int(Updating)-(len(_codeMessages)-1)] // Ensure list length matches

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

// MarshalText satisfies TextMarshaler
func (code Code) MarshalText() ([]byte, error) {
	return []byte(code.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (code *Code) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_codes); i++ {
		if enum == _codes[i] {
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
