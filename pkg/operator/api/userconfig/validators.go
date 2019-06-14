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
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type InputSchema struct {
	Type      InputTypeSchema `json:"_type" yaml:"_type"`
	Optional  bool            `json:"_optional" yaml:"_optional"`
	Default   interface{}     `json:"_default" yaml:"_default"`
	AllowNull bool            `json:"_allow_null" yaml:"_allow_null"`
	MinCount  *int64          `json:"_min_count" yaml:"_min_count"`
	MaxCount  *int64          `json:"_max_count" yaml:"_max_count"`
}

type InputTypeSchema interface{} // CompundType, length-one array of *InputSchema, or map of {scalar|CompoundType -> *InputSchema}

type OutputSchema interface{} // ValueType, length-one array of OutputSchema, or map of {scalar|ValueType -> OutputSchema} (no *_COLUMN types, compound types, or input options like _default)

func inputSchemaValidator(in interface{}) (interface{}, error) {
	if in == nil {
		return nil, nil
	}
	return ValidateInputSchema(in, false, false) // This casts it to *InputSchema
}

func inputSchemaValidatorValueTypesOnly(in interface{}) (interface{}, error) {
	if in == nil {
		return nil, nil
	}
	return ValidateInputSchema(in, true, false) // This casts it to *InputSchema
}

func outputSchemaValidator(in interface{}) (interface{}, error) {
	if in == nil {
		return nil, nil
	}
	return ValidateOutputSchema(in)
}

func ValidateInputSchema(in interface{}, disallowColumnTypes bool, isAlreadyParsed bool) (*InputSchema, error) {
	// Check for cortex options vs short form
	if inMap, ok := cast.InterfaceToStrInterfaceMap(in); ok {
		foundUnderscore, foundNonUnderscore := false, false
		for key := range inMap {
			if strings.HasPrefix(key, "_") {
				foundUnderscore = true
			} else {
				foundNonUnderscore = true
			}
		}

		if foundUnderscore {
			if foundNonUnderscore {
				return nil, ErrorMixedInputArgOptionsAndUserKeys()
			}

			inputSchemaValidation := &cr.StructValidation{
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Type",
						InterfaceValidation: &cr.InterfaceValidation{
							Required: true,
							Validator: func(t interface{}) (interface{}, error) {
								return ValidateInputTypeSchema(t, disallowColumnTypes, isAlreadyParsed)
							},
						},
					},
					{
						StructField:    "Optional",
						BoolValidation: &cr.BoolValidation{},
					},
					{
						StructField: "Default",
						InterfaceValidation: &cr.InterfaceValidation{
							AllowExplicitNull: isAlreadyParsed,
						},
					},
					{
						StructField:         "AllowNull",
						InterfaceValidation: &cr.InterfaceValidation{},
					},
					{
						StructField: "MinCount",
						Int64PtrValidation: &cr.Int64PtrValidation{
							GreaterThanOrEqualTo: pointer.Int64(0),
							AllowExplicitNull:    isAlreadyParsed,
						},
					},
					{
						StructField: "MaxCount",
						Int64PtrValidation: &cr.Int64PtrValidation{
							GreaterThanOrEqualTo: pointer.Int64(0),
							AllowExplicitNull:    isAlreadyParsed,
						},
					},
				},
			}
			inputSchema := &InputSchema{}
			errs := cr.Struct(inputSchema, inMap, inputSchemaValidation)

			if errors.HasErrors(errs) {
				return nil, errors.FirstError(errs...)
			}

			if err := validateInputSchemaOptions(inputSchema); err != nil {
				return nil, err
			}

			return inputSchema, nil
		}
	}

	typeSchema, err := ValidateInputTypeSchema(in, disallowColumnTypes, isAlreadyParsed)
	if err != nil {
		return nil, err
	}
	inputSchema := &InputSchema{
		Type: typeSchema,
	}

	if err := validateInputSchemaOptions(inputSchema); err != nil {
		return nil, err
	}

	return inputSchema, nil
}

func ValidateInputTypeSchema(in interface{}, disallowColumnTypes bool, isAlreadyParsed bool) (InputTypeSchema, error) {
	// String
	if inStr, ok := in.(string); ok {
		compoundType, err := CompoundTypeFromString(inStr)
		if err != nil {
			return nil, err
		}
		if disallowColumnTypes && compoundType.IsColumns() {
			return nil, ErrorColumnTypeNotAllowed(inStr)
		}
		return compoundType, nil
	}

	// List
	if inSlice, ok := cast.InterfaceToInterfaceSlice(in); ok {
		if len(inSlice) != 1 {
			return nil, ErrorTypeListLength(inSlice)
		}
		inputSchema, err := ValidateInputSchema(inSlice[0], disallowColumnTypes, isAlreadyParsed)
		if err != nil {
			return nil, errors.Wrap(err, s.Index(0))
		}
		return []interface{}{inputSchema}, nil
	}

	// Map
	if inMap, ok := cast.InterfaceToInterfaceInterfaceMap(in); ok {
		if len(inMap) == 0 {
			return nil, ErrorTypeMapZeroLength(inMap)
		}

		var typeKey CompoundType
		var typeValue interface{}
		for k, v := range inMap {
			var err error
			typeKey, err = CompoundTypeFromString(k)
			if err == nil {
				typeValue = v
				break
			}
		}

		// Generic map
		if typeValue != nil {
			if len(inMap) != 1 {
				return nil, ErrorGenericTypeMapLength(inMap)
			}
			if disallowColumnTypes && typeKey.IsColumns() {
				return nil, ErrorColumnTypeNotAllowed(typeKey)
			}
			valueInputSchema, err := ValidateInputSchema(typeValue, disallowColumnTypes, isAlreadyParsed)
			if err != nil {
				return nil, errors.Wrap(err, string(typeKey))
			}
			return map[interface{}]interface{}{typeKey: valueInputSchema}, nil
		}

		// Fixed map
		outMap := map[interface{}]interface{}{}
		for key, value := range inMap {
			if !cast.IsScalarType(key) {
				return nil, configreader.ErrorInvalidPrimitiveType(key, configreader.PrimTypeScalars...)
			}
			if keyStr, ok := key.(string); ok {
				if strings.HasPrefix(keyStr, "_") {
					return nil, ErrorUserKeysCannotStartWithUnderscore(keyStr)
				}
			}

			valueInputSchema, err := ValidateInputSchema(value, disallowColumnTypes, isAlreadyParsed)
			if err != nil {
				return nil, errors.Wrap(err, s.UserStrStripped(key))
			}
			outMap[key] = valueInputSchema
		}
		return outMap, nil
	}

	return nil, ErrorInvalidInputType(in)
}

func validateInputSchemaOptions(inputSchema *InputSchema) error {
	if inputSchema.Default != nil {
		inputSchema.Optional = true
	}

	_, isSlice := cast.InterfaceToInterfaceSlice(inputSchema.Type)
	isGenericMap := false
	if interfaceMap, ok := cast.InterfaceToInterfaceInterfaceMap(inputSchema.Type); ok {
		for k := range interfaceMap {
			_, isGenericMap = k.(CompoundType)
			break
		}
	}

	if inputSchema.MinCount != nil {
		if !isGenericMap && !isSlice {
			return ErrorOptionOnNonIterable(MinCountOptKey)
		}
	}

	if inputSchema.MaxCount != nil {
		if !isGenericMap && !isSlice {
			return ErrorOptionOnNonIterable(MaxCountOptKey)
		}
	}

	if inputSchema.MinCount != nil && inputSchema.MaxCount != nil && *inputSchema.MinCount > *inputSchema.MaxCount {
		return ErrorMinCountGreaterThanMaxCount()
	}

	// Validate default against schema
	if inputSchema.Default != nil {
		var err error
		inputSchema.Default, err = CastInputValue(inputSchema.Default, inputSchema)
		if err != nil {
			return errors.Wrap(err, DefaultOptKey)
		}
	}

	return nil
}

func CastInputValue(value interface{}, inputSchema *InputSchema) (interface{}, error) {
	// Check for null
	if value == nil {
		if inputSchema.AllowNull {
			return nil, nil
		}
		return nil, ErrorCannotBeNull()
	}

	typeSchema := inputSchema.Type

	// CompoundType
	if compoundType, ok := typeSchema.(CompoundType); ok {
		return compoundType.CastValue(value)
	}

	// array of *InputSchema
	if inputSchemas, ok := cast.InterfaceToInterfaceSlice(typeSchema); ok {
		values, ok := cast.InterfaceToInterfaceSlice(value)
		if !ok {
			return nil, ErrorUnsupportedLiteralType(value, typeSchema)
		}

		if inputSchema.MinCount != nil && int64(len(values)) < *inputSchema.MinCount {
			return nil, ErrorTooFewElements(configreader.PrimTypeList, *inputSchema.MinCount)
		}
		if inputSchema.MaxCount != nil && int64(len(values)) > *inputSchema.MaxCount {
			return nil, ErrorTooManyElements(configreader.PrimTypeList, *inputSchema.MaxCount)
		}

		valuesCasted := make([]interface{}, len(values))
		for i, valueItem := range values {
			valueItemCasted, err := CastInputValue(valueItem, inputSchemas[0].(*InputSchema))
			if err != nil {
				return nil, errors.Wrap(err, s.Index(i))
			}
			valuesCasted[i] = valueItemCasted
		}
		return valuesCasted, nil
	}

	// Map
	if typeSchemaMap, ok := cast.InterfaceToInterfaceInterfaceMap(typeSchema); ok {
		valueMap, ok := cast.InterfaceToInterfaceInterfaceMap(value)
		if !ok {
			return nil, ErrorUnsupportedLiteralType(value, typeSchema)
		}

		var genericKey CompoundType
		var genericValue *InputSchema
		for k, v := range typeSchemaMap {
			ok := false
			if genericKey, ok = k.(CompoundType); ok {
				genericValue = v.(*InputSchema)
			}
		}

		valueMapCasted := make(map[interface{}]interface{}, len(valueMap))

		// Generic map
		if genericValue != nil {
			if inputSchema.MinCount != nil && int64(len(valueMap)) < *inputSchema.MinCount {
				return nil, ErrorTooFewElements(configreader.PrimTypeMap, *inputSchema.MinCount)
			}
			if inputSchema.MaxCount != nil && int64(len(valueMap)) > *inputSchema.MaxCount {
				return nil, ErrorTooManyElements(configreader.PrimTypeMap, *inputSchema.MaxCount)
			}

			for valueKey, valueVal := range valueMap {
				valueKeyCasted, err := CastInputValue(valueKey, &InputSchema{Type: genericKey})
				if err != nil {
					return nil, err
				}
				valueValCasted, err := CastInputValue(valueVal, genericValue)
				if err != nil {
					return nil, errors.Wrap(err, s.UserStrStripped(valueKey))
				}
				valueMapCasted[valueKeyCasted] = valueValCasted
			}
			return valueMapCasted, nil
		}

		// Fixed map
		for typeSchemaKey, typeSchemaValue := range typeSchemaMap {
			valueVal, ok := valueMap[typeSchemaKey]
			if ok {
				valueValCasted, err := CastInputValue(valueVal, typeSchemaValue.(*InputSchema))
				if err != nil {
					return nil, errors.Wrap(err, s.UserStrStripped(typeSchemaKey))
				}
				valueMapCasted[typeSchemaKey] = valueValCasted
			} else {
				if !typeSchemaValue.(*InputSchema).Optional {
					return nil, errors.Wrap(ErrorMustBeDefined(typeSchemaValue), s.UserStrStripped(typeSchemaKey))
				}
				// don't set default (python has to)
			}
		}
		for valueKey := range valueMap {
			if _, ok := typeSchemaMap[valueKey]; !ok {
				return nil, ErrorUnsupportedLiteralMapKey(valueKey, typeSchemaMap)
			}
		}
		return valueMapCasted, nil
	}

	return nil, ErrorInvalidInputType(typeSchema) // unexpected
}

func ValidateOutputSchema(in interface{}) (OutputSchema, error) {
	// String
	if inStr, ok := in.(string); ok {
		valueType := ValueTypeFromString(inStr)
		if valueType == UnknownValueType {
			if colType := ColumnTypeFromString(inStr); colType != UnknownColumnType && colType != InferredColumnType {
				return nil, ErrorColumnTypeNotAllowed(inStr)
			}
			if _, err := CompoundTypeFromString(inStr); err == nil {
				return nil, ErrorCompoundTypeInOutputType(inStr)
			}
			return nil, ErrorInvalidOutputType(inStr)
		}
		return valueType, nil
	}

	// List
	if inSlice, ok := cast.InterfaceToInterfaceSlice(in); ok {
		if len(inSlice) != 1 {
			return nil, ErrorTypeListLength(inSlice)
		}
		outputSchema, err := ValidateOutputSchema(inSlice[0])
		if err != nil {
			return nil, errors.Wrap(err, s.Index(0))
		}
		return []interface{}{outputSchema}, nil
	}

	// Map
	if inMap, ok := cast.InterfaceToInterfaceInterfaceMap(in); ok {
		if len(inMap) == 0 {
			return nil, ErrorTypeMapZeroLength(inMap)
		}

		var typeKey ValueType
		var typeValue interface{}
		for k, v := range inMap {
			if kStr, ok := k.(string); ok {
				typeKey = ValueTypeFromString(kStr)
				if typeKey != UnknownValueType {
					typeValue = v
					break
				}
				if colType := ColumnTypeFromString(kStr); colType != UnknownColumnType && colType != InferredColumnType {
					return nil, ErrorColumnTypeNotAllowed(kStr)
				}
				if _, err := CompoundTypeFromString(kStr); err == nil {
					return nil, ErrorCompoundTypeInOutputType(kStr)
				}
			}
		}

		// Generic map
		if typeValue != nil {
			if len(inMap) != 1 {
				return nil, ErrorGenericTypeMapLength(inMap)
			}
			valueOutputSchema, err := ValidateOutputSchema(typeValue)
			if err != nil {
				return nil, errors.Wrap(err, string(typeKey))
			}
			return map[interface{}]interface{}{typeKey: valueOutputSchema}, nil
		}

		// Fixed map
		castedSchemaMap := map[interface{}]interface{}{}
		for key, value := range inMap {
			if !cast.IsScalarType(key) {
				return nil, configreader.ErrorInvalidPrimitiveType(key, configreader.PrimTypeScalars...)
			}
			if keyStr, ok := key.(string); ok {
				if strings.HasPrefix(keyStr, "_") {
					return nil, ErrorUserKeysCannotStartWithUnderscore(keyStr)
				}
			}

			valueOutputSchema, err := ValidateOutputSchema(value)
			if err != nil {
				return nil, errors.Wrap(err, s.UserStrStripped(key))
			}
			castedSchemaMap[key] = valueOutputSchema
		}
		return castedSchemaMap, nil
	}

	return nil, ErrorInvalidOutputType(in)
}

func CastOutputValue(value interface{}, outputSchema OutputSchema) (interface{}, error) {
	// Check for missing
	if value == nil {
		return nil, ErrorCannotBeNull()
	}

	// ValueType
	if valueType, ok := outputSchema.(ValueType); ok {
		return valueType.CastValue(value)
	}

	// Array
	if typeSchemas, ok := cast.InterfaceToInterfaceSlice(outputSchema); ok {
		values, ok := cast.InterfaceToInterfaceSlice(value)
		if !ok {
			return nil, ErrorUnsupportedLiteralType(value, outputSchema)
		}
		valuesCasted := make([]interface{}, len(values))
		for i, valueItem := range values {
			valueItemCasted, err := CastOutputValue(valueItem, typeSchemas[0])
			if err != nil {
				return nil, errors.Wrap(err, s.Index(i))
			}
			valuesCasted[i] = valueItemCasted
		}
		return valuesCasted, nil
	}

	// Map
	if typeSchemaMap, ok := cast.InterfaceToInterfaceInterfaceMap(outputSchema); ok {
		valueMap, ok := cast.InterfaceToInterfaceInterfaceMap(value)
		if !ok {
			return nil, ErrorUnsupportedLiteralType(value, outputSchema)
		}

		isGeneric := false
		var genericKey ValueType
		var genericValue interface{}
		for k, v := range typeSchemaMap {
			ok := false
			if genericKey, ok = k.(ValueType); ok {
				isGeneric = true
				genericValue = v
			}
		}

		valueMapCasted := make(map[interface{}]interface{}, len(valueMap))

		// Generic map
		if isGeneric {
			for valueKey, valueVal := range valueMap {
				valueKeyCasted, err := CastOutputValue(valueKey, genericKey)
				if err != nil {
					return nil, err
				}
				valueValCasted, err := CastOutputValue(valueVal, genericValue)
				if err != nil {
					return nil, errors.Wrap(err, s.UserStrStripped(valueKey))
				}
				valueMapCasted[valueKeyCasted] = valueValCasted
			}
			return valueMapCasted, nil
		}

		// Fixed map
		for typeSchemaKey, typeSchemaValue := range typeSchemaMap {
			valueVal, ok := valueMap[typeSchemaKey]
			if !ok {
				return nil, errors.Wrap(ErrorMustBeDefined(typeSchemaValue), s.UserStrStripped(typeSchemaKey))
			}
			valueValCasted, err := CastOutputValue(valueVal, typeSchemaValue)
			if err != nil {
				return nil, errors.Wrap(err, s.UserStrStripped(typeSchemaKey))
			}
			valueMapCasted[typeSchemaKey] = valueValCasted
		}
		for valueKey := range valueMap {
			if _, ok := typeSchemaMap[valueKey]; !ok {
				return nil, ErrorUnsupportedLiteralMapKey(valueKey, typeSchemaMap)
			}
		}
		return valueMapCasted, nil
	}

	return nil, ErrorInvalidOutputType(outputSchema) // unexpected
}
