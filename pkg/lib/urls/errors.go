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

package urls

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrInvalidURL          = "urls.invalid_url"
	ErrDNS1035             = "urls.dns1035"
	ErrDNS1123             = "urls.dns1123"
	ErrEndpoint            = "urls.endpoint"
	ErrEndpointEmptyPath   = "urls.endpoint_empty_path"
	ErrEndpointDoubleSlash = "urls.endpoint_double_slash"
)

func ErrorInvalidURL(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidURL,
		Message: fmt.Sprintf("%s is not a valid URL", s.UserStr(provided)),
	})
}

func ErrorDNS1035(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDNS1035,
		Message: fmt.Sprintf("%s must contain only lower case letters, numbers, and dashes, start with a letter, and cannot end with a dash", s.UserStr(provided)),
	})
}

func ErrorDNS1123(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDNS1123,
		Message: fmt.Sprintf("%s must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character", s.UserStr(provided)),
	})
}

func ErrorEndpoint(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEndpoint,
		Message: fmt.Sprintf("%s must consist of lower case alphanumeric characters, '/', '-', '_', or '.'", s.UserStr(provided)),
	})
}

func ErrorEndpointEmptyPath() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEndpointEmptyPath,
		Message: fmt.Sprintf("%s is not allowed (a path must be specified)", s.UserStr("/")),
	})
}

func ErrorEndpointDoubleSlash(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEndpointDoubleSlash,
		Message: fmt.Sprintf("%s cannot contain adjacent slashes", s.UserStr(provided)),
	})
}
