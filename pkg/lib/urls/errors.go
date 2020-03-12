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

package urls

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrInvalidURL
	ErrDNS1035
	ErrDNS1123
	ErrEndpoint
	ErrEndpointEmptyPath
	ErrEndpointDoubleSlash
)

var _errorKinds = []string{
	"err_unknown",
	"err_invalid_url",
	"err_dns1035",
	"err_dns1123",
	"err_endpoint",
	"err_endpoint_empty_path",
	"err_endpoint_double_slash",
}

var _ = [1]int{}[int(ErrEndpointDoubleSlash)-(len(_errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return _errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_errorKinds); i++ {
		if enum == _errorKinds[i] {
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

func ErrorInvalidURL(provided string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidURL,
		Message: fmt.Sprintf("%s is not a valid URL", s.UserStr(provided)),
	})
}

func ErrorDNS1035(provided string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrDNS1035,
		Message: fmt.Sprintf("%s must contain only lower case letters, numbers, and dashes, start with a letter, and cannot end with a dash", s.UserStr(provided)),
	})
}

func ErrorDNS1123(provided string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrDNS1123,
		Message: fmt.Sprintf("%s must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character", s.UserStr(provided)),
	})
}

func ErrorEndpoint(provided string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrEndpoint,
		Message: fmt.Sprintf("%s must consist of lower case alphanumeric characters, '/', '-', '_', or '.'", s.UserStr(provided)),
	})
}

func ErrorEndpointEmptyPath() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrEndpointEmptyPath,
		Message: fmt.Sprintf("%s is not allowed (a path must be specified)", s.UserStr("/")),
	})
}

func ErrorEndpointDoubleSlash(provided string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrEndpointDoubleSlash,
		Message: fmt.Sprintf("%s cannot contain adjacent slashes", s.UserStr(provided)),
	})
}
