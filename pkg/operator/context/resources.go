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

package context

import (
	"github.com/cortexlabs/yaml"

	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func ValidateInput(
	input interface{},
	schema *userconfig.InputSchema,
	validResourceTypes []resource.Type, // this is just used for error messages
	validResources []context.Resource,
	allResources map[string][]userconfig.Resource, // key is resource name
	aggregators context.Aggregators,
	transformers context.Transformers,
) (interface{}, string, error) {

	validResourcesMap := make(map[string]context.Resource, len(validResources))
	for _, res := range validResources {
		validResourcesMap[res.GetName()] = res
	}

	inputWithIDs, err := validateResourceReferences(input, validResourceTypes, validResourcesMap, allResources)
	if err != nil {
		return nil, "", err
	}

	castedInput := input

	if schema != nil {
		if input == nil {
			if schema.Optional {
				return nil, hash.Any(nil), nil
			}
			return nil, "", userconfig.ErrorMustBeDefined(schema)
		}

		castedInput, err = validateRuntimeTypes(input, schema, validResourcesMap, aggregators, transformers, false)
		if err != nil {
			return nil, "", err
		}
	}

	return castedInput, hash.Any(inputWithIDs), nil
}

// Return a copy of the input with all resource references replaced by their IDs
func validateResourceReferences(
	input interface{},
	validResourceTypes []resource.Type, // this is just used for error messages
	validResources map[string]context.Resource, // key is resource name
	allResources map[string][]userconfig.Resource, // key is resource name
) (interface{}, error) {

	if input == nil {
		return nil, nil
	}

	if resourceName, ok := yaml.ExtractAtSymbolTextInter(input); ok {
		if res, ok := validResources[resourceName]; ok {
			return res.GetID(), nil
		}

		if len(allResources[resourceName]) > 0 {
			return nil, userconfig.ErrorResourceWrongType(allResources[resourceName], validResourceTypes...)
		}

		return nil, userconfig.ErrorUndefinedResource(resourceName, validResourceTypes...)
	}

	if inputSlice, ok := cast.InterfaceToInterfaceSlice(input); ok {
		sliceWithIDs := make([]interface{}, len(inputSlice))
		for i, elem := range inputSlice {
			elemWithIDs, err := validateResourceReferences(elem, validResourceTypes, validResources, allResources)
			if err != nil {
				return nil, errors.Wrap(err, s.Index(i))
			}
			sliceWithIDs[i] = elemWithIDs
		}
		return sliceWithIDs, nil
	}

	if inputMap, ok := cast.InterfaceToInterfaceInterfaceMap(input); ok {
		mapWithIDs := make(map[interface{}]interface{}, len(inputMap))
		for key, val := range inputMap {
			keyWithIDs, err := validateResourceReferences(key, validResourceTypes, validResources, allResources)
			if err != nil {
				return nil, err
			}
			valWithIDs, err := validateResourceReferences(val, validResourceTypes, validResources, allResources)
			if err != nil {
				return nil, errors.Wrap(err, s.UserStrStripped(key))
			}
			mapWithIDs[keyWithIDs] = valWithIDs
		}
		return mapWithIDs, nil
	}

	return input, nil
}

// resource references have already been validated to exist in validResources
func validateRuntimeTypes(
	input interface{},
	schema *userconfig.InputSchema,
	validResources map[string]context.Resource, // key is resource name
	aggregators context.Aggregators,
	transformers context.Transformers,
	isNestedConstant bool,
) (interface{}, error) {

	// Check for null
	if input == nil {
		if schema.AllowNull {
			return nil, nil
		}
		return nil, userconfig.ErrorCannotBeNull()
	}

	// Check if input is Cortex resource
	if resourceName, ok := yaml.ExtractAtSymbolTextInter(input); ok {
		res := validResources[resourceName]
		if res == nil {
			return nil, errors.New(resourceName, "missing resource") // unexpected
		}
		switch res.GetResourceType() {
		case resource.ConstantType:
			constant := res.(*context.Constant)
			if constant.Value != nil {
				_, err := validateRuntimeTypes(constant.Value, schema, validResources, aggregators, transformers, true)
				if err != nil {
					return nil, errors.Wrap(err, userconfig.Identify(constant), userconfig.ValueKey)
				}
			} else if constant.Type != nil {
				err := validateInputRuntimeOutputTypes(constant.Type, schema)
				if err != nil {
					return nil, errors.Wrap(err, userconfig.Identify(constant), userconfig.TypeKey)
				}
			}
			return input, nil
		case resource.RawColumnType:
			rawColumn := res.(context.RawColumn)
			if err := validateInputRuntimeOutputTypes(rawColumn.GetColumnType(), schema); err != nil {
				return nil, errors.Wrap(err, userconfig.Identify(rawColumn), userconfig.TypeKey)
			}
			return input, nil
		case resource.AggregateType:
			aggregate := res.(*context.Aggregate)
			aggregator := aggregators[aggregate.Aggregator]
			if aggregator.OutputType != nil {
				if err := validateInputRuntimeOutputTypes(aggregator.OutputType, schema); err != nil {
					return nil, errors.Wrap(err, userconfig.Identify(aggregate), userconfig.Identify(aggregator), userconfig.OutputTypeKey)
				}
			}
			return input, nil
		case resource.TransformedColumnType:
			transformedColumn := res.(*context.TransformedColumn)
			transformer := transformers[transformedColumn.Transformer]
			if err := validateInputRuntimeOutputTypes(transformer.OutputType, schema); err != nil {
				return nil, errors.Wrap(err, userconfig.Identify(transformedColumn), userconfig.Identify(transformer), userconfig.OutputTypeKey)
			}
			return input, nil
		default:
			return nil, errors.New(res.GetResourceType().String(), "unsupported resource type") // unexpected
		}
	}

	typeSchema := schema.Type

	// CompoundType
	if compoundType, ok := typeSchema.(userconfig.CompoundType); ok {
		return compoundType.CastValue(input)
	}

	// array of *InputSchema
	if inputSchemas, ok := cast.InterfaceToInterfaceSlice(typeSchema); ok {
		values, ok := cast.InterfaceToInterfaceSlice(input)
		if !ok {
			return nil, userconfig.ErrorUnsupportedLiteralType(input, typeSchema)
		}

		if schema.MinCount != nil && int64(len(values)) < *schema.MinCount {
			return nil, userconfig.ErrorTooFewElements(configreader.PrimTypeList, *schema.MinCount)
		}
		if schema.MaxCount != nil && int64(len(values)) > *schema.MaxCount {
			return nil, userconfig.ErrorTooManyElements(configreader.PrimTypeList, *schema.MaxCount)
		}

		valuesCasted := make([]interface{}, len(values))
		for i, valueItem := range values {
			valueItemCasted, err := validateRuntimeTypes(valueItem, inputSchemas[0].(*userconfig.InputSchema), validResources, aggregators, transformers, false)
			if err != nil {
				return nil, errors.Wrap(err, s.Index(i))
			}
			valuesCasted[i] = valueItemCasted
		}
		return valuesCasted, nil
	}

	// Map
	if typeSchemaMap, ok := cast.InterfaceToInterfaceInterfaceMap(typeSchema); ok {
		valueMap, ok := cast.InterfaceToInterfaceInterfaceMap(input)
		if !ok {
			return nil, userconfig.ErrorUnsupportedLiteralType(input, typeSchema)
		}

		var genericKey userconfig.CompoundType
		var genericValue *userconfig.InputSchema
		for k, v := range typeSchemaMap {
			ok := false
			if genericKey, ok = k.(userconfig.CompoundType); ok {
				genericValue = v.(*userconfig.InputSchema)
			}
		}

		valueMapCasted := make(map[interface{}]interface{}, len(valueMap))

		// Generic map
		if genericValue != nil {
			if schema.MinCount != nil && int64(len(valueMap)) < *schema.MinCount {
				return nil, userconfig.ErrorTooFewElements(configreader.PrimTypeMap, *schema.MinCount)
			}
			if schema.MaxCount != nil && int64(len(valueMap)) > *schema.MaxCount {
				return nil, userconfig.ErrorTooManyElements(configreader.PrimTypeMap, *schema.MaxCount)
			}

			for valueKey, valueVal := range valueMap {
				valueKeyCasted, err := validateRuntimeTypes(valueKey, &userconfig.InputSchema{Type: genericKey}, validResources, aggregators, transformers, false)
				if err != nil {
					return nil, err
				}
				valueValCasted, err := validateRuntimeTypes(valueVal, genericValue, validResources, aggregators, transformers, false)
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
				valueValCasted, err := validateRuntimeTypes(valueVal, typeSchemaValue.(*userconfig.InputSchema), validResources, aggregators, transformers, false)
				if err != nil {
					return nil, errors.Wrap(err, s.UserStrStripped(typeSchemaKey))
				}
				valueMapCasted[typeSchemaKey] = valueValCasted
			} else {
				if !typeSchemaValue.(*userconfig.InputSchema).Optional {
					return nil, errors.Wrap(userconfig.ErrorMustBeDefined(typeSchemaValue), s.UserStrStripped(typeSchemaKey))
				}
				// don't set default (python has to)
			}
		}
		if !isNestedConstant {
			for valueKey := range valueMap {
				if _, ok := typeSchemaMap[valueKey]; !ok {
					return nil, userconfig.ErrorUnsupportedLiteralMapKey(valueKey, typeSchemaMap)
				}
			}
		}
		return valueMapCasted, nil
	}

	return nil, userconfig.ErrorInvalidInputType(typeSchema) // unexpected
}

// outputType should be ValueType|ColumnType, length-one array of <recursive>, or map of {scalar|ValueType -> <recursive>}
func validateInputRuntimeOutputTypes(outputType interface{}, schema *userconfig.InputSchema) error {
	// Check for missing
	if outputType == nil {
		if schema.AllowNull {
			return nil
		}
		return userconfig.ErrorCannotBeNull()
	}

	typeSchema := schema.Type

	// CompoundType
	if compoundType, ok := typeSchema.(userconfig.CompoundType); ok {
		if !compoundType.SupportsType(outputType) {
			return userconfig.ErrorUnsupportedOutputType(outputType, compoundType)
		}
		return nil
	}

	// array of *InputSchema
	if inputSchemas, ok := cast.InterfaceToInterfaceSlice(typeSchema); ok {
		outputTypes, ok := cast.InterfaceToInterfaceSlice(outputType)
		if !ok {
			return userconfig.ErrorUnsupportedOutputType(outputType, inputSchemas)
		}

		err := validateInputRuntimeOutputTypes(outputTypes[0], inputSchemas[0].(*userconfig.InputSchema))
		if err != nil {
			return errors.Wrap(err, s.Index(0))
		}
		return nil
	}

	// Map
	if typeSchemaMap, ok := cast.InterfaceToInterfaceInterfaceMap(typeSchema); ok {
		outputTypeMap, ok := cast.InterfaceToInterfaceInterfaceMap(outputType)
		if !ok {
			return userconfig.ErrorUnsupportedOutputType(outputType, typeSchemaMap)
		}

		var typeSchemaGenericKey userconfig.CompoundType
		var typeSchemaGenericValue *userconfig.InputSchema
		for k, v := range typeSchemaMap {
			ok := false
			if typeSchemaGenericKey, ok = k.(userconfig.CompoundType); ok {
				typeSchemaGenericValue = v.(*userconfig.InputSchema)
			}
		}

		var outputTypeGenericKey userconfig.ValueType
		var outputTypeGenericValue interface{}
		for k, v := range outputTypeMap {
			ok := false
			if outputTypeGenericKey, ok = k.(userconfig.ValueType); ok {
				outputTypeGenericValue = v
			}
		}

		// Check length if fixed outputType
		if outputTypeGenericValue == nil {
			if schema.MinCount != nil && int64(len(outputTypeMap)) < *schema.MinCount {
				return userconfig.ErrorTooFewElements(configreader.PrimTypeMap, *schema.MinCount)
			}
			if schema.MaxCount != nil && int64(len(outputTypeMap)) > *schema.MaxCount {
				return userconfig.ErrorTooManyElements(configreader.PrimTypeMap, *schema.MaxCount)
			}
		}

		// Generic schema map and generic outputType
		if typeSchemaGenericValue != nil && outputTypeGenericValue != nil {
			if err := validateInputRuntimeOutputTypes(outputTypeGenericKey, &userconfig.InputSchema{Type: typeSchemaGenericKey}); err != nil {
				return err
			}
			if err := validateInputRuntimeOutputTypes(outputTypeGenericValue, typeSchemaGenericValue); err != nil {
				return errors.Wrap(err, s.UserStrStripped(outputTypeGenericKey))
			}
			return nil
		}

		// Generic schema map and fixed outputType (we'll check the types of the fixed map)
		if typeSchemaGenericValue != nil && outputTypeGenericValue == nil {
			for outputTypeKey, outputTypeValue := range outputTypeMap {
				if _, err := typeSchemaGenericKey.CastValue(outputTypeKey); err != nil {
					return err
				}
				if err := validateInputRuntimeOutputTypes(outputTypeValue, typeSchemaGenericValue); err != nil {
					return errors.Wrap(err, s.UserStrStripped(outputTypeKey))
				}
			}
			return nil
		}

		// Generic outputType map and fixed schema map
		if typeSchemaGenericValue == nil && outputTypeGenericValue != nil {
			return userconfig.ErrorUnsupportedOutputType(outputType, typeSchemaMap)
			// This code would allow for this case (for now we are considering it an error):
			// for typeSchemaKey, typeSchemaValue := range typeSchemaMap {
			// 	if _, err := outputTypeGenericKey.CastValue(typeSchemaKey); err != nil {
			// 		return err
			// 	}
			// 	if err := validateInputRuntimeOutputTypes(outputTypeGenericValue, typeSchemaValue.(*userconfig.InputSchema)); err != nil {
			// 		return errors.Wrap(err, s.UserStrStripped(typeSchemaKey))
			// 	}
			// }
			// return nil
		}

		// Fixed outputType map and fixed schema map
		if typeSchemaGenericValue == nil && outputTypeGenericValue == nil {
			for typeSchemaKey, typeSchemaValue := range typeSchemaMap {
				outputTypeValue, ok := outputTypeMap[typeSchemaKey]
				if ok {
					if err := validateInputRuntimeOutputTypes(outputTypeValue, typeSchemaValue.(*userconfig.InputSchema)); err != nil {
						return errors.Wrap(err, s.UserStrStripped(typeSchemaKey))
					}
				} else {
					if !typeSchemaValue.(*userconfig.InputSchema).Optional {
						return errors.Wrap(userconfig.ErrorMustBeDefined(typeSchemaValue), s.UserStrStripped(typeSchemaKey))
					}
				}
			}
			return nil
		}
	}

	return userconfig.ErrorInvalidInputType(typeSchema) // unexpected
}
