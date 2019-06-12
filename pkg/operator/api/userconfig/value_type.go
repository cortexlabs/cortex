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

package userconfig

import (
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type ValueType int
type ValueTypes []ValueType

const (
	UnknownValueType ValueType = iota
	IntegerValueType
	FloatValueType
	StringValueType
	BoolValueType
)

var valueTypes = []string{
	"unknown",
	"INT",
	"FLOAT",
	"STRING",
	"BOOL",
}

func ValueTypeFromString(s string) ValueType {
	for i := 0; i < len(valueTypes); i++ {
		if s == valueTypes[i] {
			return ValueType(i)
		}
	}
	return UnknownValueType
}

func ValueTypeStrings() []string {
	return valueTypes[1:]
}

func (t ValueType) String() string {
	return valueTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t ValueType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ValueType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(valueTypes); i++ {
		if enum == valueTypes[i] {
			*t = ValueType(i)
			return nil
		}
	}

	*t = UnknownValueType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ValueType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ValueType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

func (ts ValueTypes) StringList() []string {
	strs := make([]string, len(ts))
	for i, t := range ts {
		strs[i] = t.String()
	}
	return strs
}

func (ts ValueTypes) String() string {
	return strings.Join(ts.StringList(), ", ")
}

func (t *ValueType) CastValue(value interface{}) (interface{}, error) {
	switch *t {
	case IntegerValueType:
		valueInt, ok := cast.InterfaceToInt64(value)
		if !ok {
			return nil, ErrorUnsupportedLiteralType(value, t.String())
		}
		return valueInt, nil

	case FloatValueType:
		valueFloat, ok := cast.InterfaceToFloat64(value)
		if !ok {
			return nil, ErrorUnsupportedLiteralType(value, t.String())
		}
		return valueFloat, nil

	case StringValueType:
		valueStr, ok := value.(string)
		if !ok {
			return nil, ErrorUnsupportedLiteralType(value, t.String())
		}
		return valueStr, nil

	case BoolValueType:
		valueBool, ok := value.(bool)
		if !ok {
			return nil, ErrorUnsupportedLiteralType(value, t.String())
		}
		return valueBool, nil
	}

	return nil, errors.New(t.String(), "unimplemented ValueType") // unexpected
}
