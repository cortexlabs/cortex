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

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

func isValidColumnOutputType(columnTypeStr string) bool {
	return slices.HasString(columnTypeStr, ColumnTypeStrings())
}

func isValidColumnInputType(columnTypeStr string) bool {
	for _, columnTypeStrItem := range strings.Split(columnTypeStr, "|") {
		if !slices.HasString(columnTypeStrItem, ColumnTypeStrings()) {
			return false
		}
	}
	return true
}

func isValidValueType(valueTypeStr string) bool {
	for _, valueTypeStrItem := range strings.Split(valueTypeStr, "|") {
		if !slices.HasString(valueTypeStrItem, ValueTypeStrings()) {
			return false
		}
	}
	return true
}

func ValidateColumnInputTypes(columnTypes map[string]interface{}) error {
	for columnInputName, columnType := range columnTypes {
		if columnTypeStr, ok := columnType.(string); ok {
			if !isValidColumnInputType(columnTypeStr) {
				return errors.Wrap(ErrorInvalidColumnInputType(columnTypeStr), columnInputName)
			}
			continue
		}

		if columnTypeStrs, ok := cast.InterfaceToStrSlice(columnType); ok {
			if len(columnTypeStrs) != 1 {
				return errors.Wrap(ErrorTypeListLength(columnTypeStrs), columnInputName)
			}
			if !isValidColumnInputType(columnTypeStrs[0]) {
				return errors.Wrap(ErrorInvalidColumnInputType(columnTypeStrs), columnInputName)
			}
			continue
		}

		return errors.Wrap(ErrorInvalidColumnInputType(columnType), columnInputName)
	}

	return nil
}

func ValidateColumnInputValues(columnInputValues map[string]interface{}) error {
	for columnInputName, columnInputValue := range columnInputValues {
		if _, ok := columnInputValue.(string); ok {
			continue
		}
		if columnNames, ok := cast.InterfaceToStrSlice(columnInputValue); ok {
			if columnNames == nil {
				return errors.New(columnInputName, s.ErrCannotBeNull)
			}
			continue
		}
		return errors.New(columnInputName, s.ErrInvalidPrimitiveType(columnInputValue, s.PrimTypeString, s.PrimTypeStringList))
	}

	return nil
}

func ValidateColumnRuntimeTypes(columnRuntimeTypes map[string]interface{}) error {
	for columnInputName, columnType := range columnRuntimeTypes {
		if columnTypeStr, ok := columnType.(string); ok {
			if !isValidColumnOutputType(columnTypeStr) {
				return errors.Wrap(ErrorInvalidColumnRuntimeType(columnTypeStr), columnInputName)
			}
			continue
		}
		if columnTypeStrs, ok := cast.InterfaceToStrSlice(columnType); ok {
			for i, columnTypeStr := range columnTypeStrs {
				if !isValidColumnOutputType(columnTypeStr) {
					return errors.Wrap(ErrorInvalidColumnRuntimeType(columnTypeStr), columnInputName, s.Index(i))
				}
			}
			continue
		}
		return errors.Wrap(ErrorInvalidColumnRuntimeType(columnType), columnInputName)
	}

	return nil
}

func CheckColumnRuntimeTypesMatch(columnRuntimeTypes map[string]interface{}, columnSchemaTypes map[string]interface{}) error {
	err := ValidateColumnInputTypes(columnSchemaTypes)
	if err != nil {
		return err
	}
	err = ValidateColumnRuntimeTypes(columnRuntimeTypes)
	if err != nil {
		return err
	}

	for columnInputName, columnSchemaType := range columnSchemaTypes {
		if len(columnRuntimeTypes) == 0 {
			return errors.New(s.MapMustBeDefined(maps.InterfaceMapKeys(columnSchemaTypes)...))
		}

		columnRuntimeType, ok := columnRuntimeTypes[columnInputName]
		if !ok {
			return errors.New(columnInputName, s.ErrMustBeDefined)
		}

		if columnSchemaTypeStr, ok := columnSchemaType.(string); ok {
			validTypes := strings.Split(columnSchemaTypeStr, "|")
			columnRuntimeTypeStr, ok := columnRuntimeType.(string)
			if !ok {
				return errors.Wrap(ErrorUnsupportedColumnType(columnRuntimeType, validTypes), columnInputName)
			}
			if !slices.HasString(columnRuntimeTypeStr, validTypes) {
				return errors.Wrap(ErrorUnsupportedColumnType(columnRuntimeTypeStr, validTypes), columnInputName)
			}
			continue
		}

		if columnSchemaTypeStrs, ok := cast.InterfaceToStrSlice(columnSchemaType); ok {
			validTypes := strings.Split(columnSchemaTypeStrs[0], "|")
			columnRuntimeTypeStrs, ok := cast.InterfaceToStrSlice(columnRuntimeType)
			if !ok {
				return errors.Wrap(ErrorUnsupportedColumnType(columnRuntimeType, columnSchemaTypeStrs), columnInputName)
			}
			for i, columnRuntimeTypeStr := range columnRuntimeTypeStrs {
				if !slices.HasString(columnRuntimeTypeStr, validTypes) {
					return errors.Wrap(ErrorUnsupportedColumnType(columnRuntimeTypeStr, validTypes), columnInputName, s.Index(i))
				}
			}
			continue
		}

		return errors.Wrap(ErrorInvalidColumnInputType(columnSchemaType), columnInputName) // unexpected
	}

	for columnInputName := range columnRuntimeTypes {
		if _, ok := columnSchemaTypes[columnInputName]; !ok {
			return errors.New(s.ErrUnsupportedKey(columnInputName))
		}
	}

	return nil
}

func ValidateArgTypes(argTypes map[string]interface{}) error {
	for argName, valueType := range argTypes {
		if isValidValueType(argName) {
			return ErrorArgNameCannotBeType(argName)
		}
		err := ValidateValueType(valueType)
		if err != nil {
			return errors.Wrap(err, argName)
		}
	}
	return nil
}

func ValidateValueType(valueType interface{}) error {
	if valueTypeStr, ok := valueType.(string); ok {
		if !isValidValueType(valueTypeStr) {
			return ErrorInvalidValueDataType(valueTypeStr)
		}
		return nil
	}

	if valueTypeStrs, ok := cast.InterfaceToStrSlice(valueType); ok {
		if len(valueTypeStrs) != 1 {
			return errors.Wrap(ErrorTypeListLength(valueTypeStrs))
		}
		if !isValidValueType(valueTypeStrs[0]) {
			return ErrorInvalidValueDataType(valueTypeStrs[0])
		}
		return nil
	}

	if valueTypeMap, ok := cast.InterfaceToInterfaceInterfaceMap(valueType); ok {
		foundGenericKey := false
		for key := range valueTypeMap {
			if strKey, ok := key.(string); ok {
				if isValidValueType(strKey) {
					foundGenericKey = true
					break
				}
			}
		}
		if foundGenericKey && len(valueTypeMap) != 1 {
			return ErrorGenericTypeMapLength(valueTypeMap)
		}

		for key, val := range valueTypeMap {
			if foundGenericKey {
				err := ValidateValueType(key)
				if err != nil {
					return err
				}
			}
			err := ValidateValueType(val)
			if err != nil {
				return errors.Wrap(err, s.UserStrStripped(key))
			}
		}
		return nil
	}

	return ErrorInvalidValueDataType(valueType)
}

func ValidateArgValues(argValues map[string]interface{}) error {
	for argName, value := range argValues {
		err := ValidateValue(value)
		if err != nil {
			return errors.Wrap(err, argName)
		}
	}
	return nil
}

func ValidateValue(value interface{}) error {
	return nil
}

func CastValue(value interface{}, valueType interface{}) (interface{}, error) {
	err := ValidateValueType(valueType)
	if err != nil {
		return nil, err
	}
	err = ValidateValue(value)
	if err != nil {
		return nil, err
	}

	if value == nil {
		return nil, nil
	}

	if valueTypeStr, ok := valueType.(string); ok {
		validTypes := strings.Split(valueTypeStr, "|")
		var validTypeNames []s.PrimitiveType

		if slices.HasString(IntegerValueType.String(), validTypes) {
			validTypeNames = append(validTypeNames, s.PrimTypeInt)
			valueInt, ok := cast.InterfaceToInt64(value)
			if ok {
				return valueInt, nil
			}
		}
		if slices.HasString(FloatValueType.String(), validTypes) {
			validTypeNames = append(validTypeNames, s.PrimTypeFloat)
			valueFloat, ok := cast.InterfaceToFloat64(value)
			if ok {
				return valueFloat, nil
			}
		}
		if slices.HasString(StringValueType.String(), validTypes) {
			validTypeNames = append(validTypeNames, s.PrimTypeString)
			if valueStr, ok := value.(string); ok {
				return valueStr, nil
			}
		}
		if slices.HasString(BoolValueType.String(), validTypes) {
			validTypeNames = append(validTypeNames, s.PrimTypeBool)
			if valueBool, ok := value.(bool); ok {
				return valueBool, nil
			}
		}
		return nil, errors.New(s.ErrInvalidPrimitiveType(value, validTypeNames...))
	}

	if valueTypeMap, ok := cast.InterfaceToInterfaceInterfaceMap(valueType); ok {
		valueMap, ok := cast.InterfaceToInterfaceInterfaceMap(value)
		if !ok {
			return nil, errors.New(s.ErrInvalidPrimitiveType(value, s.PrimTypeMap))
		}

		if len(valueTypeMap) == 0 {
			if len(valueMap) == 0 {
				return make(map[interface{}]interface{}), nil
			}
			return nil, errors.New(s.UserStr(valueMap), s.ErrMustBeEmpty)
		}

		isGenericMap := false
		var genericMapKeyType string
		var genericMapValueType interface{}
		if len(valueTypeMap) == 1 {
			for valueTypeKey, valueTypeVal := range valueTypeMap { // Will only be length one
				if valueTypeKeyStr, ok := valueTypeKey.(string); ok {
					if isValidValueType(valueTypeKeyStr) {
						isGenericMap = true
						genericMapKeyType = valueTypeKeyStr
						genericMapValueType = valueTypeVal
					}
				}
			}
		}

		if isGenericMap {
			valueMapCasted := make(map[interface{}]interface{}, len(valueMap))
			for valueKey, valueVal := range valueMap {
				valueKeyCasted, err := CastValue(valueKey, genericMapKeyType)
				if err != nil {
					return nil, err
				}
				valueValCasted, err := CastValue(valueVal, genericMapValueType)
				if err != nil {
					return nil, errors.Wrap(err, s.UserStrStripped(valueKey))
				}
				valueMapCasted[valueKeyCasted] = valueValCasted
			}
			return valueMapCasted, nil
		}

		// Non-generic map
		valueMapCasted := make(map[interface{}]interface{}, len(valueMap))
		for valueKey, valueType := range valueTypeMap {
			valueVal, ok := valueMap[valueKey]
			if !ok {
				return nil, errors.New(s.UserStrStripped(valueKey), s.ErrMustBeDefined)
			}
			valueValCasted, err := CastValue(valueVal, valueType)
			if err != nil {
				return nil, errors.Wrap(err, s.UserStrStripped(valueKey))
			}
			valueMapCasted[valueKey] = valueValCasted
		}
		for valueKey := range valueMap {
			if _, ok := valueTypeMap[valueKey]; !ok {
				return nil, errors.New(s.ErrUnsupportedKey(valueKey))
			}
		}
		return valueMapCasted, nil
	}

	if valueTypeStrs, ok := cast.InterfaceToStrSlice(valueType); ok {
		valueTypeStr := valueTypeStrs[0]
		valueSlice, ok := cast.InterfaceToInterfaceSlice(value)
		if !ok {
			return nil, errors.New(s.ErrInvalidPrimitiveType(value, s.PrimTypeList))
		}
		valueSliceCasted := make([]interface{}, len(valueSlice))
		for i, valueItem := range valueSlice {
			valueItemCasted, err := CastValue(valueItem, valueTypeStr)
			if err != nil {
				return nil, errors.Wrap(err, s.Index(i))
			}
			valueSliceCasted[i] = valueItemCasted
		}
		return valueSliceCasted, nil
	}

	return nil, ErrorInvalidValueDataType(valueType) // unexpected
}

func CheckArgRuntimeTypesMatch(argRuntimeTypes map[string]interface{}, argSchemaTypes map[string]interface{}) error {
	err := ValidateArgTypes(argSchemaTypes)
	if err != nil {
		return err
	}
	err = ValidateArgTypes(argRuntimeTypes)
	if err != nil {
		return err
	}

	for argName, argSchemaType := range argSchemaTypes {
		if len(argRuntimeTypes) == 0 {
			return errors.New(s.MapMustBeDefined(maps.InterfaceMapKeys(argSchemaTypes)...))
		}

		argRuntimeType, ok := argRuntimeTypes[argName]
		if !ok {
			return errors.New(argName, s.ErrMustBeDefined)
		}
		err := CheckValueRuntimeTypesMatch(argRuntimeType, argSchemaType)
		if err != nil {
			return errors.Wrap(err, argName)
		}
	}

	for argName := range argRuntimeTypes {
		if _, ok := argSchemaTypes[argName]; !ok {
			return errors.New(s.ErrUnsupportedKey(argName))
		}
	}

	return nil
}

func CheckValueRuntimeTypesMatch(runtimeType interface{}, schemaType interface{}) error {
	if schemaTypeStr, ok := schemaType.(string); ok {
		validTypes := strings.Split(schemaTypeStr, "|")
		runtimeTypeStr, ok := runtimeType.(string)
		if !ok {
			return ErrorUnsupportedDataType(runtimeType, schemaTypeStr)
		}
		for _, runtimeTypeOption := range strings.Split(runtimeTypeStr, "|") {
			if !slices.HasString(runtimeTypeOption, validTypes) {
				return ErrorUnsupportedDataType(runtimeTypeStr, schemaTypeStr)
			}
		}
		return nil
	}

	if schemaTypeMap, ok := cast.InterfaceToInterfaceInterfaceMap(schemaType); ok {
		runtimeTypeMap, ok := cast.InterfaceToInterfaceInterfaceMap(runtimeType)
		if !ok {
			return ErrorUnsupportedDataType(runtimeType, schemaTypeMap)
		}

		isGenericMap := false
		var genericMapKeyType string
		var genericMapValueType interface{}
		if len(schemaTypeMap) == 1 {
			for schemaTypeKey, schemaTypeValue := range schemaTypeMap { // Will only be length one
				if schemaTypeMapStr, ok := schemaTypeKey.(string); ok {
					if isValidValueType(schemaTypeMapStr) {
						isGenericMap = true
						genericMapKeyType = schemaTypeMapStr
						genericMapValueType = schemaTypeValue
					}
				}
			}
		}

		if isGenericMap {
			for runtimeTypeKey, runtimeTypeValue := range runtimeTypeMap { // Should only be one item
				err := CheckValueRuntimeTypesMatch(runtimeTypeKey, genericMapKeyType)
				if err != nil {
					return err
				}
				err = CheckValueRuntimeTypesMatch(runtimeTypeValue, genericMapValueType)
				if err != nil {
					return errors.Wrap(err, s.UserStrStripped(runtimeTypeKey))
				}
			}
			return nil
		}

		// Non-generic map
		for schemaTypeKey, schemaTypeValue := range schemaTypeMap {
			runtimeTypeValue, ok := runtimeTypeMap[schemaTypeKey]
			if !ok {
				return errors.New(s.UserStrStripped(schemaTypeKey), s.ErrMustBeDefined)
			}
			err := CheckValueRuntimeTypesMatch(runtimeTypeValue, schemaTypeValue)
			if err != nil {
				return errors.Wrap(err, s.UserStrStripped(schemaTypeKey))
			}
		}
		for runtimeTypeKey := range runtimeTypeMap {
			if _, ok := schemaTypeMap[runtimeTypeKey]; !ok {
				return errors.New(s.ErrUnsupportedKey(runtimeTypeKey))
			}
		}
		return nil
	}

	if schemaTypeStrs, ok := cast.InterfaceToStrSlice(schemaType); ok {
		validTypes := strings.Split(schemaTypeStrs[0], "|")
		runtimeTypeStrs, ok := cast.InterfaceToStrSlice(runtimeType)
		if !ok {
			return ErrorUnsupportedDataType(runtimeType, schemaTypeStrs)
		}
		for _, runtimeTypeOption := range strings.Split(runtimeTypeStrs[0], "|") {
			if !slices.HasString(runtimeTypeOption, validTypes) {
				return ErrorUnsupportedDataType(runtimeTypeStrs, schemaTypeStrs)
			}
		}
		return nil
	}

	return ErrorInvalidValueDataType(schemaType) // unexpected
}
