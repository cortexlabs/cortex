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
	ErrNameNotFound
	ErrNameOrTypeNotFound
	ErrInvalidType
	ErrTemplateInTemplate
	ErrEmbedInTemplate
	ErrBeMoreSpecific
)

var (
	errorKinds = []string{
		"err_unknown",
		"err_unknown_kind",
		"err_not_found",
		"err_name_not_found",
		"err_name_or_type_not_found",
		"err_invalid_type",
		"err_template_in_template",
		"err_embed_in_template",
		"err_be_more_specific",
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

type Error struct {
	Kind    ErrorKind
	message string
}

func (e Error) Error() string {
	return e.message
}

func ErrorNotFound(name string, resourceType Type) error {
	return Error{
		Kind:    ErrNotFound,
		message: fmt.Sprintf("%s %s not found", resourceType, s.UserStr(name)),
	}
}

func ErrorNameNotFound(name string) error {
	return Error{
		Kind:    ErrNameNotFound,
		message: fmt.Sprintf("resource name %s not found", s.UserStr(name)),
	}
}

func ErrorNameOrTypeNotFound(nameOrType string) error {
	return Error{
		Kind:    ErrNameOrTypeNotFound,
		message: fmt.Sprintf("resource name or type %s not found", s.UserStr(nameOrType)),
	}
}

func ErrorInvalidType(invalid string) error {
	return Error{
		Kind:    ErrInvalidType,
		message: fmt.Sprintf("invalid resource type %s", s.UserStr(invalid)),
	}
}

func ErrorUnknownKind(name string) error {
	return Error{
		Kind:    ErrUnknownKind,
		message: fmt.Sprintf("unknown kind %s", s.UserStr(name)),
	}
}

func ErrorTemplateInTemplate() error {
	return Error{
		Kind:    ErrTemplateInTemplate,
		message: "templates cannot be defined inside of templates",
	}
}

func ErrorEmbedInTemplate() error {
	return Error{
		Kind:    ErrEmbedInTemplate,
		message: "embeds cannot be defined inside of templates",
	}
}

func ErrorBeMoreSpecific(vals ...string) error {
	return Error{
		Kind:    ErrBeMoreSpecific,
		message: fmt.Sprintf("please specify %s", s.UserStrsOr(vals)),
	}
}
