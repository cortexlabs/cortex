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

package cast

import (
	"encoding/json"
	"reflect"
)

func InterfaceToInt8(in interface{}) (int8, bool) {
	var ok bool
	if in, ok = JSONNumberToInt(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return casted, true
	case int16:
		if val := int8(casted); int16(val) == casted {
			return val, true
		}
	case int32:
		if val := int8(casted); int32(val) == casted {
			return val, true
		}
	case int:
		if val := int8(casted); int(val) == casted {
			return val, true
		}
	case int64:
		if val := int8(casted); int64(val) == casted {
			return val, true
		}
	}
	return 0, false
}

func InterfaceToInt8Downcast(in interface{}) (int8, bool) {
	var ok bool
	if in, ok = JSONNumberToIntOrFloat(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return casted, true
	case int16:
		if val := int8(casted); int16(val) == casted {
			return val, true
		}
	case int32:
		if val := int8(casted); int32(val) == casted {
			return val, true
		}
	case int:
		if val := int8(casted); int(val) == casted {
			return val, true
		}
	case int64:
		if val := int8(casted); int64(val) == casted {
			return val, true
		}
	case float32:
		if val := int8(casted); float32(val) == casted {
			return val, true
		}
	case float64:
		if val := int8(casted); float64(val) == casted {
			return val, true
		}
	}
	return 0, false
}

func InterfaceToInt16(in interface{}) (int16, bool) {
	var ok bool
	if in, ok = JSONNumberToInt(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return int16(casted), true
	case int16:
		return casted, true
	case int32:
		if val := int16(casted); int32(val) == casted {
			return val, true
		}
	case int:
		if val := int16(casted); int(val) == casted {
			return val, true
		}
	case int64:
		if val := int16(casted); int64(val) == casted {
			return val, true
		}
	}
	return 0, false
}

func InterfaceToInt16Downcast(in interface{}) (int16, bool) {
	var ok bool
	if in, ok = JSONNumberToIntOrFloat(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return int16(casted), true
	case int16:
		return casted, true
	case int32:
		if val := int16(casted); int32(val) == casted {
			return val, true
		}
	case int:
		if val := int16(casted); int(val) == casted {
			return val, true
		}
	case int64:
		if val := int16(casted); int64(val) == casted {
			return val, true
		}
	case float32:
		if val := int16(casted); float32(val) == casted {
			return val, true
		}
	case float64:
		if val := int16(casted); float64(val) == casted {
			return val, true
		}
	}
	return 0, false
}

func InterfaceToInt32(in interface{}) (int32, bool) {
	var ok bool
	if in, ok = JSONNumberToInt(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return int32(casted), true
	case int16:
		return int32(casted), true
	case int32:
		return casted, true
	case int:
		if val := int32(casted); int(val) == casted {
			return val, true
		}
	case int64:
		if val := int32(casted); int64(val) == casted {
			return val, true
		}
	}
	return 0, false
}

func InterfaceToInt32Downcast(in interface{}) (int32, bool) {
	var ok bool
	if in, ok = JSONNumberToIntOrFloat(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return int32(casted), true
	case int16:
		return int32(casted), true
	case int32:
		return casted, true
	case int:
		if val := int32(casted); int(val) == casted {
			return val, true
		}
	case int64:
		if val := int32(casted); int64(val) == casted {
			return val, true
		}
	case float32:
		if val := int32(casted); float32(val) == casted {
			return val, true
		}
	case float64:
		if val := int32(casted); float64(val) == casted {
			return val, true
		}
	}
	return 0, false
}

func InterfaceToInt(in interface{}) (int, bool) {
	var ok bool
	if in, ok = JSONNumberToInt(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return int(casted), true
	case int16:
		return int(casted), true
	case int32:
		return int(casted), true
	case int:
		return casted, true
	case int64:
		if val := int(casted); int64(val) == casted {
			return val, true
		}
	}
	return 0, false
}

func InterfaceToIntDowncast(in interface{}) (int, bool) {
	var ok bool
	if in, ok = JSONNumberToIntOrFloat(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return int(casted), true
	case int16:
		return int(casted), true
	case int32:
		return int(casted), true
	case int:
		return casted, true
	case int64:
		if val := int(casted); int64(val) == casted {
			return val, true
		}
	case float32:
		if val := int(casted); float32(val) == casted {
			return val, true
		}
	case float64:
		if val := int(casted); float64(val) == casted {
			return val, true
		}
	}
	return 0, false
}

func InterfaceToInt64(in interface{}) (int64, bool) {
	var ok bool
	if in, ok = JSONNumberToInt(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return int64(casted), true
	case int16:
		return int64(casted), true
	case int32:
		return int64(casted), true
	case int:
		return int64(casted), true
	case int64:
		return casted, true
	}
	return 0, false
}

func InterfaceToInt64Downcast(in interface{}) (int64, bool) {
	var ok bool
	if in, ok = JSONNumberToIntOrFloat(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return int64(casted), true
	case int16:
		return int64(casted), true
	case int32:
		return int64(casted), true
	case int:
		return int64(casted), true
	case int64:
		return casted, true
	case float32:
		if val := int64(casted); float32(val) == casted {
			return val, true
		}
	case float64:
		if val := int64(casted); float64(val) == casted {
			return val, true
		}
	}
	return 0, false
}

// InterfaceToFloat32 will convert any int or float type
func InterfaceToFloat32(in interface{}) (float32, bool) {
	var ok bool
	if in, ok = JSONNumberToIntOrFloat(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return float32(casted), true
	case int16:
		return float32(casted), true
	case int32:
		return float32(casted), true
	case int:
		return float32(casted), true
	case int64:
		return float32(casted), true
	case float32:
		return casted, true
	case float64:
		return float32(casted), true
	}
	return 0, false
}

// InterfaceToFloat64 will convert any int or float type
func InterfaceToFloat64(in interface{}) (float64, bool) {
	var ok bool
	if in, ok = JSONNumberToIntOrFloat(in); !ok {
		return 0, false
	}

	switch casted := in.(type) {
	case int8:
		return float64(casted), true
	case int16:
		return float64(casted), true
	case int32:
		return float64(casted), true
	case int:
		return float64(casted), true
	case int64:
		return float64(casted), true
	case float32:
		return float64(casted), true
	case float64:
		return casted, true
	}
	return 0, false
}

func JSONNumberToInt(in interface{}) (interface{}, bool) {
	number, ok := in.(json.Number)
	if !ok {
		return in, true
	}
	inInt, err := number.Int64()
	if err == nil {
		return inInt, true
	}
	return nil, false
}

func JSONNumberToIntOrFloat(in interface{}) (interface{}, bool) {
	number, ok := in.(json.Number)
	if !ok {
		return in, true
	}
	inInt, err := number.Int64()
	if err == nil {
		return inInt, true
	}
	inFloat, err := number.Float64()
	if err == nil {
		return inFloat, true
	}
	return nil, false
}

func JSONNumber(in interface{}) interface{} {
	number, ok := in.(json.Number)
	if !ok {
		return in
	}
	inInt, err := number.Int64()
	if err == nil {
		return inInt
	}
	inFloat, err := number.Float64()
	if err == nil {
		return inFloat
	}
	return in // unexpected
}

func JSONNumbers(in []interface{}) []interface{} {
	casted := make([]interface{}, len(in))
	for i, element := range in {
		casted[i] = JSONNumber(element)
	}
	return casted
}

func InterfaceToInterfaceSlice(in interface{}) ([]interface{}, bool) {
	if in == nil {
		return nil, true
	}

	if inSlice, ok := in.([]interface{}); ok {
		return inSlice, true
	}

	if reflect.TypeOf(in).Kind() != reflect.Slice {
		return nil, false
	}

	inVal := reflect.ValueOf(in)
	if inVal.IsNil() {
		return nil, true
	}

	out := make([]interface{}, inVal.Len())
	for i := 0; i < inVal.Len(); i++ {
		out[i] = inVal.Index(i).Interface()
	}
	return out, true
}

func InterfaceToIntSlice(in interface{}) ([]int, bool) {
	if in == nil {
		return nil, true
	}

	if intSlice, ok := in.([]int); ok {
		return intSlice, true
	}

	inSlice, ok := InterfaceToInterfaceSlice(in)
	if !ok {
		return nil, false
	}
	out := make([]int, len(inSlice))

	for i, elem := range inSlice {
		casted, ok := InterfaceToInt(elem)
		if !ok {
			return nil, false
		}
		out[i] = casted
	}
	return out, true
}

func InterfaceToInt32Slice(in interface{}) ([]int32, bool) {
	if in == nil {
		return nil, true
	}

	if intSlice, ok := in.([]int32); ok {
		return intSlice, true
	}

	inSlice, ok := InterfaceToInterfaceSlice(in)
	if !ok {
		return nil, false
	}
	out := make([]int32, len(inSlice))

	for i, elem := range inSlice {
		casted, ok := InterfaceToInt32(elem)
		if !ok {
			return nil, false
		}
		out[i] = casted
	}
	return out, true
}

func InterfaceToInt64Slice(in interface{}) ([]int64, bool) {
	if in == nil {
		return nil, true
	}

	if intSlice, ok := in.([]int64); ok {
		return intSlice, true
	}

	inSlice, ok := InterfaceToInterfaceSlice(in)
	if !ok {
		return nil, false
	}
	out := make([]int64, len(inSlice))

	for i, elem := range inSlice {
		casted, ok := InterfaceToInt64(elem)
		if !ok {
			return nil, false
		}
		out[i] = casted
	}
	return out, true
}

func InterfaceToFloat32Slice(in interface{}) ([]float32, bool) {
	if in == nil {
		return nil, true
	}

	if floatSlice, ok := in.([]float32); ok {
		return floatSlice, true
	}

	inSlice, ok := InterfaceToInterfaceSlice(in)
	if !ok {
		return nil, false
	}
	out := make([]float32, len(inSlice))

	for i, elem := range inSlice {
		casted, ok := InterfaceToFloat32(elem)
		if !ok {
			return nil, false
		}
		out[i] = casted
	}
	return out, true
}

func InterfaceToFloat64Slice(in interface{}) ([]float64, bool) {
	if in == nil {
		return nil, true
	}

	if floatSlice, ok := in.([]float64); ok {
		return floatSlice, true
	}

	inSlice, ok := InterfaceToInterfaceSlice(in)
	if !ok {
		return nil, false
	}
	out := make([]float64, len(inSlice))

	for i, elem := range inSlice {
		casted, ok := InterfaceToFloat64(elem)
		if !ok {
			return nil, false
		}
		out[i] = casted
	}
	return out, true
}

func InterfaceToStrSlice(in interface{}) ([]string, bool) {
	if in == nil {
		return nil, true
	}

	if strSlice, ok := in.([]string); ok {
		return strSlice, true
	}

	inSlice, ok := InterfaceToInterfaceSlice(in)
	if !ok {
		return nil, false
	}

	out := make([]string, len(inSlice))

	for i, elem := range inSlice {
		casted, ok := elem.(string)
		if !ok {
			return nil, false
		}
		out[i] = casted
	}
	return out, true
}

func InterfaceToBoolSlice(in interface{}) ([]bool, bool) {
	if in == nil {
		return nil, true
	}

	if boolSlice, ok := in.([]bool); ok {
		return boolSlice, true
	}

	inSlice, ok := InterfaceToInterfaceSlice(in)
	if !ok {
		return nil, false
	}

	out := make([]bool, len(inSlice))

	for i, elem := range inSlice {
		casted, ok := elem.(bool)
		if !ok {
			return nil, false
		}
		out[i] = casted
	}
	return out, true
}

func InterfaceToStrInterfaceMapSlice(in interface{}) ([]map[string]interface{}, bool) {
	if in == nil {
		return nil, true
	}

	if strMapSlice, ok := in.([]map[string]interface{}); ok {
		return strMapSlice, true
	}

	inSlice, ok := InterfaceToInterfaceSlice(in)
	if !ok {
		return nil, false
	}

	out := make([]map[string]interface{}, len(inSlice))

	for i, elem := range inSlice {
		casted, ok := InterfaceToStrInterfaceMap(elem)
		if !ok {
			return nil, false
		}
		out[i] = casted
	}
	return out, true
}

func InterfaceToInterfaceInterfaceMap(in interface{}) (map[interface{}]interface{}, bool) {
	if in == nil {
		return nil, true
	}

	if inMap, ok := in.(map[interface{}]interface{}); ok {
		return inMap, true
	}

	if reflect.TypeOf(in).Kind() != reflect.Map {
		return nil, false
	}

	inVal := reflect.ValueOf(in)
	if inVal.IsNil() {
		return nil, true
	}

	out := make(map[interface{}]interface{}, inVal.Len())
	for _, key := range inVal.MapKeys() {
		out[key.Interface()] = inVal.MapIndex(key).Interface()
	}
	return out, true
}

func InterfaceToStrInterfaceMap(in interface{}) (map[string]interface{}, bool) {
	if in == nil {
		return nil, true
	}

	if strMap, ok := in.(map[string]interface{}); ok {
		return strMap, true
	}

	inMap, ok := InterfaceToInterfaceInterfaceMap(in)
	if !ok {
		return nil, false
	}

	out := map[string]interface{}{}

	for key, value := range inMap {
		casted, ok := key.(string)
		if !ok {
			return nil, false
		}
		out[casted] = value
	}
	return out, true
}

// Recursively casts interface->interface maps to string->interface maps
func JSONMarshallable(in interface{}) (interface{}, bool) {
	if in == nil {
		return nil, true
	}

	if inMap, ok := InterfaceToInterfaceInterfaceMap(in); ok {
		out := map[string]interface{}{}
		for key, value := range inMap {
			castedKey, ok := key.(string)
			if !ok {
				return nil, false
			}
			castedValue, ok := JSONMarshallable(value)
			if !ok {
				return nil, false
			}
			out[castedKey] = castedValue
		}
		return out, true
	} else if inSlice, ok := InterfaceToInterfaceSlice(in); ok {
		out := make([]interface{}, 0, len(inSlice))
		for _, inValue := range inSlice {
			castedInValue, ok := JSONMarshallable(inValue)
			if !ok {
				return nil, false
			}
			out = append(out, castedInValue)
		}
		return out, true
	}
	return in, true
}

func InterfaceToStrStrMap(in interface{}) (map[string]string, bool) {
	if in == nil {
		return nil, true
	}

	if strMap, ok := in.(map[string]string); ok {
		return strMap, true
	}

	inMap, ok := InterfaceToInterfaceInterfaceMap(in)
	if !ok {
		return nil, false
	}

	out := map[string]string{}

	for key, value := range inMap {
		castedKey, ok := key.(string)
		if !ok {
			return nil, false
		}
		castedVal, ok := value.(string)
		if !ok {
			return nil, false
		}
		out[castedKey] = castedVal
	}
	return out, true
}

func StrMapToStrInterfaceMap(in map[string]string) map[string]interface{} {
	if in == nil {
		return nil
	}

	out := map[string]interface{}{}
	for k, v := range in {
		out[k] = v
	}

	return out
}

func IsIntType(in interface{}) bool {
	switch in.(type) {
	case json.Number:
		_, err := in.(json.Number).Int64()
		return err == nil
	case int8:
		return true
	case int16:
		return true
	case int32:
		return true
	case int64:
		return true
	case int:
		return true
	}
	return false
}

func IsFloatType(in interface{}) bool {
	switch in.(type) {
	case json.Number:
		_, intErr := in.(json.Number).Int64()
		_, floatErr := in.(json.Number).Float64()
		return floatErr == nil && intErr != nil
	case float32:
		return true
	case float64:
		return true
	}
	return false
}

func IsNumericType(in interface{}) bool {
	return IsIntType(in) || IsFloatType(in)
}

func IsScalarType(in interface{}) bool {
	if IsNumericType(in) {
		return true
	}

	switch in.(type) {
	case string:
		return true
	case bool:
		return true
	}

	return false
}

func FlattenInterfaceSlices(in ...interface{}) []interface{} {
	var result []interface{}

	for _, item := range in {
		if item == nil {
			result = append(result, nil)
			continue
		}
		if subItems, ok := InterfaceToInterfaceSlice(item); ok {
			if len(subItems) != 0 {
				result = append(result, FlattenInterfaceSlices(subItems...)...)
			}
			continue
		}
		result = append(result, item)
	}

	return result
}
