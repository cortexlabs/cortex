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
	"fmt"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrUnknownKind
	ErrNotFound
	ErrInvalidType
)

var (
	errorKinds = []string{
		"err_unknown",
		"err_unknown_kind",
		"err_not_found",
		"err_invalid_type",
	}
)

func (t ErrorKind) String() string {
	return errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(errorKinds); i++ {
		if enum == errorKinds[i] {
			*t = ErrorKind(i)
			return nil
		}
	}

	*t = ErrUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ErrorKind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ErrorKind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

type ResourceError struct {
	Kind ErrorKind

	message string
}

func ErrorNotFound(name string, resourceType Type) error {
	return ResourceError{
		Kind:    ErrNotFound,
		message: fmt.Sprintf("%s %s not found", resourceType, s.UserStr(name)),
	}
}

func ErrorInvalidType(invalid string) error {
	return ResourceError{
		Kind:    ErrInvalidType,
		message: fmt.Sprintf("invalid resource type %s", s.UserStr(invalid)),
	}
}

func ErrorUnknownKind(name string) error {
	return ResourceError{
		Kind:    ErrUnknownKind,
		message: fmt.Sprintf("unknown kind %s", s.UserStr(name)),
	}
}

func (e ResourceError) Error() string {
	return e.message
}
