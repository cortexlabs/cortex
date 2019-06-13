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
)

type CompoundType string

type compoundTypeParsed struct {
	Original    string
	columnTypes map[ColumnType]bool
	valueTypes  map[ValueType]bool
}

func CompoundTypeFromString(val interface{}) (CompoundType, error) {
	parsed, err := parseCompoundType(val)
	if err != nil {
		return "", err
	}
	return CompoundType(parsed.Original), nil
}

func parseCompoundType(val interface{}) (*compoundTypeParsed, error) {
	compoundStr, ok := val.(string)
	if !ok {
		return nil, ErrorInvalidCompoundType(val)
	}

	parsed := &compoundTypeParsed{
		Original:    compoundStr,
		columnTypes: map[ColumnType]bool{},
		valueTypes:  map[ValueType]bool{},
	}

	for _, str := range strings.Split(compoundStr, "|") {
		columnType := ColumnTypeFromString(str)
		if columnType != UnknownColumnType && columnType != InferredColumnType {
			if parsed.columnTypes[columnType] == true {
				return nil, ErrorDuplicateTypeInTypeString(columnType.String(), compoundStr)
			}
			parsed.columnTypes[columnType] = true
			continue
		}

		valueType := ValueTypeFromString(str)
		if valueType != UnknownValueType {
			if parsed.valueTypes[valueType] == true {
				return nil, ErrorDuplicateTypeInTypeString(valueType.String(), compoundStr)
			}
			parsed.valueTypes[valueType] = true
			continue
		}

		return nil, ErrorInvalidCompoundType(compoundStr)
	}

	if len(parsed.columnTypes) == 0 && len(parsed.valueTypes) == 0 {
		return nil, ErrorInvalidCompoundType("")
	}

	if len(parsed.columnTypes) > 0 && len(parsed.valueTypes) > 0 {
		return nil, ErrorCannotMixValueAndColumnTypes(compoundStr)
	}

	return parsed, nil
}

func (compoundType *CompoundType) String() string {
	return string(*compoundType)
}

func (compoundType *CompoundType) IsColumns() bool {
	parsed, _ := parseCompoundType(string(*compoundType))
	return len(parsed.columnTypes) > 0
}

func (compoundType *CompoundType) IsValues() bool {
	parsed, _ := parseCompoundType(string(*compoundType))
	return len(parsed.valueTypes) > 0
}

func (compoundType *CompoundType) SupportsType(t interface{}) bool {
	parsed, _ := parseCompoundType(string(*compoundType))

	if columnType, ok := t.(ColumnType); ok {
		return parsed.columnTypes[columnType] || columnType == InferredColumnType
	}

	if valueType, ok := t.(ValueType); ok {
		if valueType == IntegerValueType {
			return parsed.valueTypes[IntegerValueType] || parsed.valueTypes[FloatValueType]
		}
		return parsed.valueTypes[valueType]
	}

	if typeStr, ok := t.(string); ok {
		columnType := ColumnTypeFromString(typeStr)
		if columnType != UnknownColumnType {
			return compoundType.SupportsType(columnType)
		}
		valueType := ValueTypeFromString(typeStr)
		if valueType != UnknownValueType {
			return compoundType.SupportsType(valueType)
		}
	}

	return false
}

func (compoundType *CompoundType) CastValue(value interface{}) (interface{}, error) {
	parsed, _ := parseCompoundType(string(*compoundType))
	if len(parsed.columnTypes) > 0 {
		return nil, ErrorColumnTypeLiteral(value)
	}

	if parsed.valueTypes[IntegerValueType] {
		valueInt, ok := cast.InterfaceToInt64(value)
		if ok {
			return valueInt, nil
		}
	}

	if parsed.valueTypes[FloatValueType] {
		valueFloat, ok := cast.InterfaceToFloat64(value)
		if ok {
			return valueFloat, nil
		}
	}

	if parsed.valueTypes[StringValueType] {
		if valueStr, ok := value.(string); ok {
			return valueStr, nil
		}
	}

	if parsed.valueTypes[BoolValueType] {
		if valueBool, ok := value.(bool); ok {
			return valueBool, nil
		}
	}

	return nil, ErrorUnsupportedLiteralType(value, compoundType.String())
}
