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
)

type ColumnType int
type ColumnTypes []ColumnType

const (
	UnknownColumnType ColumnType = iota
	IntegerColumnType
	FloatColumnType
	StringColumnType
	IntegerListColumnType
	FloatListColumnType
	StringListColumnType
	ValueColumnType
)

var columnTypes = []string{
	"unknown",
	"INT_COLUMN",
	"FLOAT_COLUMN",
	"STRING_COLUMN",
	"INT_LIST_COLUMN",
	"FLOAT_LIST_COLUMN",
	"STRING_LIST_COLUMN",
	"VALUE_COLUMN",
}

var columnJSONPlaceholders = []string{
	"_",
	"INT",
	"FLOAT",
	"\"STRING\"",
	"[INT]",
	"[FLOAT]",
	"[\"STRING\"]",
	"VALUE",
}

func ColumnTypeFromString(s string) ColumnType {
	for i := 0; i < len(columnTypes); i++ {
		if s == columnTypes[i] {
			return ColumnType(i)
		}
	}
	return UnknownColumnType
}

func ColumnTypeStrings() []string {
	return columnTypes[1:]
}

func (t ColumnType) String() string {
	return columnTypes[t]
}

func (t ColumnType) JSONPlaceholder() string {
	return columnJSONPlaceholders[t]
}

// MarshalText satisfies TextMarshaler
func (t ColumnType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ColumnType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(columnTypes); i++ {
		if enum == columnTypes[i] {
			*t = ColumnType(i)
			return nil
		}
	}

	*t = UnknownColumnType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ColumnType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ColumnType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

func (ts ColumnTypes) StringList() []string {
	strs := make([]string, len(ts))
	for i, t := range ts {
		strs[i] = t.String()
	}
	return strs
}

func (ts ColumnTypes) String() string {
	return strings.Join(ts.StringList(), ", ")
}
