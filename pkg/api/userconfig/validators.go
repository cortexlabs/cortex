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
	"github.com/cortexlabs/cortex/pkg/utils/cast"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func isValidFeatureOutputType(featureTypeStr string) bool {
	return util.IsStrInSlice(featureTypeStr, FeatureTypeStrings())
}

func isValidFeatureInputType(featureTypeStr string) bool {
	for _, featureTypeStrItem := range strings.Split(featureTypeStr, "|") {
		if !util.IsStrInSlice(featureTypeStrItem, FeatureTypeStrings()) {
			return false
		}
	}
	return true
}

func isValidValueType(valueTypeStr string) bool {
	for _, valueTypeStrItem := range strings.Split(valueTypeStr, "|") {
		if !util.IsStrInSlice(valueTypeStrItem, ValueTypeStrings()) {
			return false
		}
	}
	return true
}

func ValidateFeatureInputTypes(featureTypes map[string]interface{}) error {
	for featureInputName, featureType := range featureTypes {
		if featureTypeStr, ok := featureType.(string); ok {
			if !isValidFeatureInputType(featureTypeStr) {
				return errors.Wrap(ErrorInvalidFeatureInputType(featureTypeStr), featureInputName)
			}
			continue
		}

		if featureTypeStrs, ok := cast.InterfaceToStrSlice(featureType); ok {
			if len(featureTypeStrs) != 1 {
				return errors.Wrap(ErrorTypeListLength(featureTypeStrs), featureInputName)
			}
			if !isValidFeatureInputType(featureTypeStrs[0]) {
				return errors.Wrap(ErrorInvalidFeatureInputType(featureTypeStrs), featureInputName)
			}
			continue
		}

		return errors.Wrap(ErrorInvalidFeatureInputType(featureType), featureInputName)
	}

	return nil
}

func ValidateFeatureValues(featuresValues map[string]interface{}) error {
	for featureInputName, featureValue := range featuresValues {
		if _, ok := featureValue.(string); ok {
			continue
		}
		if featureNames, ok := cast.InterfaceToStrSlice(featureValue); ok {
			if featureNames == nil {
				return errors.New(featureInputName, s.ErrCannotBeNull)
			}
			continue
		}
		return errors.New(featureInputName, s.ErrInvalidPrimitiveType(featureValue, s.PrimTypeString, s.PrimTypeStringList))
	}

	return nil
}

func ValidateFeatureRuntimeTypes(featureRuntimeTypes map[string]interface{}) error {
	for featureInputName, featureType := range featureRuntimeTypes {
		if featureTypeStr, ok := featureType.(string); ok {
			if !isValidFeatureOutputType(featureTypeStr) {
				return errors.Wrap(ErrorInvalidFeatureRuntimeType(featureTypeStr), featureInputName)
			}
			continue
		}
		if featureTypeStrs, ok := cast.InterfaceToStrSlice(featureType); ok {
			for i, featureTypeStr := range featureTypeStrs {
				if !isValidFeatureOutputType(featureTypeStr) {
					return errors.Wrap(ErrorInvalidFeatureRuntimeType(featureTypeStr), featureInputName, s.Index(i))
				}
			}
			continue
		}
		return errors.Wrap(ErrorInvalidFeatureRuntimeType(featureType), featureInputName)
	}

	return nil
}

func CheckFeatureRuntimeTypesMatch(featureRuntimeTypes map[string]interface{}, featureSchemaTypes map[string]interface{}) error {
	err := ValidateFeatureInputTypes(featureSchemaTypes)
	if err != nil {
		return err
	}
	err = ValidateFeatureRuntimeTypes(featureRuntimeTypes)
	if err != nil {
		return err
	}

	for featureInputName, featureSchemaType := range featureSchemaTypes {
		if len(featureRuntimeTypes) == 0 {
			return errors.New(s.MapMustBeDefined(util.InterfaceMapKeys(featureSchemaTypes)...))
		}

		featureRuntimeType, ok := featureRuntimeTypes[featureInputName]
		if !ok {
			return errors.New(featureInputName, s.ErrMustBeDefined)
		}

		if featureSchemaTypeStr, ok := featureSchemaType.(string); ok {
			validTypes := strings.Split(featureSchemaTypeStr, "|")
			featureRuntimeTypeStr, ok := featureRuntimeType.(string)
			if !ok {
				return errors.Wrap(ErrorUnsupportedFeatureType(featureRuntimeType, validTypes), featureInputName)
			}
			if !util.IsStrInSlice(featureRuntimeTypeStr, validTypes) {
				return errors.Wrap(ErrorUnsupportedFeatureType(featureRuntimeTypeStr, validTypes), featureInputName)
			}
			continue
		}

		if featureSchemaTypeStrs, ok := cast.InterfaceToStrSlice(featureSchemaType); ok {
			validTypes := strings.Split(featureSchemaTypeStrs[0], "|")
			featureRuntimeTypeStrs, ok := cast.InterfaceToStrSlice(featureRuntimeType)
			if !ok {
				return errors.Wrap(ErrorUnsupportedFeatureType(featureRuntimeType, featureSchemaTypeStrs), featureInputName)
			}
			for i, featureRuntimeTypeStr := range featureRuntimeTypeStrs {
				if !util.IsStrInSlice(featureRuntimeTypeStr, validTypes) {
					return errors.Wrap(ErrorUnsupportedFeatureType(featureRuntimeTypeStr, validTypes), featureInputName, s.Index(i))
				}
			}
			continue
		}

		return errors.Wrap(ErrorInvalidFeatureInputType(featureSchemaType), featureInputName) // unexpected
	}

	for featureInputName := range featureRuntimeTypes {
		if _, ok := featureSchemaTypes[featureInputName]; !ok {
			return errors.New(s.ErrUnsupportedKey(featureInputName))
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
		for key, _ := range valueTypeMap {
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

		if util.IsStrInSlice(IntegerValueType.String(), validTypes) {
			validTypeNames = append(validTypeNames, s.PrimTypeInt)
			valueInt, ok := cast.InterfaceToInt64(value)
			if ok {
				return valueInt, nil
			}
		}
		if util.IsStrInSlice(FloatValueType.String(), validTypes) {
			validTypeNames = append(validTypeNames, s.PrimTypeFloat)
			valueFloat, ok := cast.InterfaceToFloat64(value)
			if ok {
				return valueFloat, nil
			}
		}
		if util.IsStrInSlice(StringValueType.String(), validTypes) {
			validTypeNames = append(validTypeNames, s.PrimTypeString)
			if valueStr, ok := value.(string); ok {
				return valueStr, nil
			}
		}
		if util.IsStrInSlice(BoolValueType.String(), validTypes) {
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
			return errors.New(s.MapMustBeDefined(util.InterfaceMapKeys(argSchemaTypes)...))
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
			if !util.IsStrInSlice(runtimeTypeOption, validTypes) {
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
			if !util.IsStrInSlice(runtimeTypeOption, validTypes) {
				return ErrorUnsupportedDataType(runtimeTypeStrs, schemaTypeStrs)
			}
		}
		return nil
	}

	return ErrorInvalidValueDataType(schemaType) // unexpected
}
