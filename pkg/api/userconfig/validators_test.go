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

package userconfig_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
)

func TestValidateColumnInputTypes(t *testing.T) {
	var columnTypes map[string]interface{}

	columnTypes = cr.MustReadYAMLStrMap("num: FLOAT_COLUMN")
	require.NoError(t, userconfig.ValidateColumnInputTypes(columnTypes))

	columnTypes = cr.MustReadYAMLStrMap(
		`
     float: FLOAT_COLUMN
     int: INT_COLUMN
     str: STRING_COLUMN
     int_list: FLOAT_LIST_COLUMN
     float_list: INT_LIST_COLUMN
     str_list: STRING_LIST_COLUMN
    `)
	require.NoError(t, userconfig.ValidateColumnInputTypes(columnTypes))

	columnTypes = cr.MustReadYAMLStrMap(
		`
     num1: FLOAT_COLUMN|INT_COLUMN
     num2: INT_COLUMN|FLOAT_COLUMN
     num3: STRING_COLUMN|INT_COLUMN
     num4: INT_COLUMN|FLOAT_COLUMN|STRING_COLUMN
     num5: STRING_COLUMN|INT_COLUMN|FLOAT_COLUMN
     num6: STRING_LIST_COLUMN|INT_LIST_COLUMN|FLOAT_LIST_COLUMN
     num7: STRING_COLUMN|INT_LIST_COLUMN|FLOAT_LIST_COLUMN
    `)
	require.NoError(t, userconfig.ValidateColumnInputTypes(columnTypes))

	columnTypes = cr.MustReadYAMLStrMap(
		`
     nums1: [INT_COLUMN]
     nums2: [FLOAT_COLUMN]
     nums3: [INT_COLUMN|FLOAT_COLUMN]
     nums4: [FLOAT_COLUMN|INT_COLUMN]
     nums5: [STRING_COLUMN|INT_COLUMN|FLOAT_COLUMN]
     nums6: [INT_LIST_COLUMN]
     nums7: [INT_LIST_COLUMN|STRING_LIST_COLUMN]
     nums8: [INT_LIST_COLUMN|STRING_COLUMN]
     strs: [STRING_COLUMN]
     num1: FLOAT_COLUMN
     num2: INT_COLUMN
     str_list: STRING_LIST_COLUMN
    `)
	require.NoError(t, userconfig.ValidateColumnInputTypes(columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("num: bad")
	require.Error(t, userconfig.ValidateColumnInputTypes(columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("num: BOOL")
	require.Error(t, userconfig.ValidateColumnInputTypes(columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("num: [STRING_COLUMN, INT_COLUMN]")
	require.Error(t, userconfig.ValidateColumnInputTypes(columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("num: FLOAT")
	require.Error(t, userconfig.ValidateColumnInputTypes(columnTypes))
	columnTypes = cr.MustReadYAMLStrMap("num: FLOAT_COLUMNs")
	require.Error(t, userconfig.ValidateColumnInputTypes(columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("num: 1")
	require.Error(t, userconfig.ValidateColumnInputTypes(columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("num: [1]")
	require.Error(t, userconfig.ValidateColumnInputTypes(columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("num: {nested: STRING_COLUMN}")
	require.Error(t, userconfig.ValidateColumnInputTypes(columnTypes))
}

func TestValidateColumnInputValues(t *testing.T) {
	var columnInputValues map[string]interface{}

	columnInputValues = cr.MustReadYAMLStrMap("num: age")
	require.NoError(t, userconfig.ValidateColumnInputValues(columnInputValues))

	columnInputValues = cr.MustReadYAMLStrMap(
		`
     num1: age
     num2: income
     str: prior_default
    `)
	require.NoError(t, userconfig.ValidateColumnInputValues(columnInputValues))

	columnInputValues = cr.MustReadYAMLStrMap(
		`
     num1: age
     num2: income
    `)
	require.NoError(t, userconfig.ValidateColumnInputValues(columnInputValues))

	columnInputValues = cr.MustReadYAMLStrMap(
		`
     nums1: [age, income]
     nums2: [income]
     nums3: [age, income, years_employed]
     strs: [prior_default, approved]
     num1: age
     num2: income
     str: prior_default
    `)
	require.NoError(t, userconfig.ValidateColumnInputValues(columnInputValues))

	columnInputValues = cr.MustReadYAMLStrMap("num: 1")
	require.Error(t, userconfig.ValidateColumnInputValues(columnInputValues))

	columnInputValues = cr.MustReadYAMLStrMap("num: [1]")
	require.Error(t, userconfig.ValidateColumnInputValues(columnInputValues))

	columnInputValues = cr.MustReadYAMLStrMap("num: {nested: STRING_COLUMN}")
	require.Error(t, userconfig.ValidateColumnInputValues(columnInputValues))
}

func TestCheckColumnRuntimeTypesMatch(t *testing.T) {
	var columnTypes map[string]interface{}
	var runtimeTypes map[string]interface{}

	columnTypes = cr.MustReadYAMLStrMap("in: INT_COLUMN")
	runtimeTypes = readRuntimeTypes("in: INT_COLUMN")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: FLOAT_COLUMN")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN]")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("in: INT_COLUMN|FLOAT_COLUMN")
	runtimeTypes = readRuntimeTypes("in: INT_COLUMN")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: FLOAT_COLUMN")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: STRING_COLUMN")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("in: STRING_COLUMN|INT_COLUMN|FLOAT_COLUMN")
	runtimeTypes = readRuntimeTypes("in: INT_COLUMN")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: FLOAT_COLUMN")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: STRING_COLUMN")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: BAD_COLUMN")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("in: [INT_COLUMN]")
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN, INT_COLUMN, INT_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: INT_COLUMN")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [FLOAT_COLUMN]")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN, FLOAT_COLUMN, INT_COLUMN]")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("in: [INT_COLUMN|FLOAT_COLUMN]")
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN, INT_COLUMN, INT_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: INT_COLUMN")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [FLOAT_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN, FLOAT_COLUMN, INT_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN, FLOAT_COLUMN, STRING_COLUMN]")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("in: [STRING_COLUMN|INT_COLUMN|FLOAT_COLUMN]")
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN, INT_COLUMN, INT_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: INT_COLUMN")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [BAD_COLUMN]")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [STRING_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in: [INT_COLUMN, FLOAT_COLUMN, STRING_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("in1: [INT_COLUMN]\nin2: STRING_COLUMN")
	runtimeTypes = readRuntimeTypes("in1: [INT_COLUMN]\nin2: STRING_COLUMN")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in1: [INT_COLUMN, INT_COLUMN]\nin2: STRING_COLUMN")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in2: STRING_COLUMN\nin1: [INT_COLUMN, INT_COLUMN]")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in1: [INT_COLUMN]")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in1: [INT_COLUMN]\nin2: STRING_COLUMN\nin3: INT_COLUMN")
	require.Error(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))

	columnTypes = cr.MustReadYAMLStrMap("in1: [INT_COLUMN|FLOAT_COLUMN|STRING_COLUMN]\nin2: STRING_COLUMN")
	runtimeTypes = readRuntimeTypes("in1: [INT_COLUMN]\nin2: STRING_COLUMN")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
	runtimeTypes = readRuntimeTypes("in1: [INT_COLUMN, FLOAT_COLUMN, STRING_COLUMN, FLOAT_COLUMN]\nin2: STRING_COLUMN")
	require.NoError(t, userconfig.CheckColumnRuntimeTypesMatch(runtimeTypes, columnTypes))
}

func readRuntimeTypes(yamlStr string) map[string]interface{} {
	runtimeTypes := make(map[string]interface{})
	runtimeTypesStr := cr.MustReadYAMLStrMap(yamlStr)

	for k, v := range runtimeTypesStr {
		if runtimeTypeStr, ok := v.(string); ok {
			runtimeTypes[k] = userconfig.ColumnTypeFromString(runtimeTypeStr)
		} else if runtimeTypeStrs, ok := cast.InterfaceToStrSlice(v); ok {
			runtimeTypesSlice := make([]userconfig.ColumnType, len(runtimeTypeStrs))
			for i, runtimeTypeStr := range runtimeTypeStrs {
				runtimeTypesSlice[i] = userconfig.ColumnTypeFromString(runtimeTypeStr)
			}
			runtimeTypes[k] = runtimeTypesSlice
		}
	}

	return runtimeTypes
}

func TestValidateArgTypes(t *testing.T) {
	var argTypes map[string]interface{}

	argTypes = cr.MustReadYAMLStrMap("STRING: FLOAT")
	require.Error(t, userconfig.ValidateArgTypes(argTypes))
}

func TestValidateValueType(t *testing.T) {
	var valueType interface{}

	valueType = "FLOAT"
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = "FLOAT|INT|BOOL|STRING"
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = []string{"INT|FLOAT"}
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("STRING: FLOAT")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("num: FLOAT")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("bools: [BOOL]")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("bools: [BOOL|FLOAT|INT]")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("STRING: INT")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {BOOL|FLOAT: INT|STRING}")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {mean: FLOAT, stddev: FLOAT}")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {STRING: {lat: FLOAT, lon: FLOAT}}")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {STRING: {lat: FLOAT, lon: [FLOAT]}}")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {STRING: {FLOAT: INT}}")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {STRING: {FLOAT: [INT]}}")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {STRING: {lat: FLOAT, lon: {lat2: FLOAT, lon2: INT}}}")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map6: {STRING: {lat: FLOAT, lon: {lat2: FLOAT, lon2: {INT: STRING}}}}")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map6: {STRING: {lat: FLOAT, lon: {lat2: FLOAT, lon2: {INT: STRING}, mean: BOOL}}}")
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap(
		`
     num: [INT]
     str: STRING
     map1: {STRING: INT}
     map2: {mean: FLOAT, stddev: FLOAT}
     map3: {STRING: {lat: FLOAT, lon: FLOAT}}
     map3: {STRING: {lat: FLOAT, lon: [FLOAT]}}
     map4: {STRING: {FLOAT: INT}}
     map5: {STRING: {BOOL: [INT]}}
     map6: {STRING: {lat: FLOAT, lon: {lat2: FLOAT, lon2: INT}}}
     map6: {STRING: {lat: FLOAT, lon: {lat2: FLOAT, lon2: {INT: STRING}, mean: BOOL}}}
    `)
	require.NoError(t, userconfig.ValidateValueType(valueType))

	valueType = "FLOAT|INT|BAD"
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = []string{"INT", "FLOAT"}
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("num: FLOATs")
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = 1
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("num: 1")
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("num: [1]")
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("STRING: test")
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {STRING: INT, INT: FLOAT}")
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {STRING: INT, INT: [FLOAT]}")
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {mean: FLOAT, INT: FLOAT}")
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {mean: FLOAT, INT: [FLOAT]}")
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {STRING: {lat: FLOAT, STRING: FLOAT}}")
	require.Error(t, userconfig.ValidateValueType(valueType))

	valueType = cr.MustReadYAMLStrMap("map: {STRING: {STRING: test}}")
	require.Error(t, userconfig.ValidateValueType(valueType))
}

func TestCastValue(t *testing.T) {
	var valueType interface{}
	var value interface{}
	var casted interface{}
	var err error

	valueType = "INT"
	value = int64(2)
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	value = float64(2.2)
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
	value = nil
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)

	valueType = "FLOAT"
	value = float64(2.2)
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	value = int64(2)
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, float64(2))

	valueType = "BOOL"
	value = false
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	value = 2
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)

	valueType = "FLOAT|INT"
	value = float64(2.2)
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	value = int64(2)
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, int64(2))

	valueType = cr.MustReadYAMLStrMap("STRING: FLOAT")
	value = cr.MustReadYAMLStrMap("test: 2.2")
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	value = cr.MustReadYAMLStrMap("test: 2.2\ntest2: 4.4")
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, map[interface{}]interface{}{"test": 2.2, "test2": 4.4})
	value = cr.MustReadYAMLStrMap("test: test2")
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
	value = map[int]float64{2: 2.2}
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
	value = make(map[string]float64)
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	value = cr.MustReadYAMLStrMap("test: 2") // YAML
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, map[interface{}]interface{}{"test": float64(2)})
	value = cr.MustReadJSONStr(`{"test": 2}`) // JSON
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, map[interface{}]interface{}{"test": float64(2)})
	value = cr.MustReadYAMLStrMap("test: 2.0") // YAML
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, map[interface{}]interface{}{"test": float64(2)})
	value = cr.MustReadJSONStr(`{"test": 2.0}`) // JSON
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, map[interface{}]interface{}{"test": float64(2)})

	valueType = cr.MustReadYAMLStrMap("STRING: INT")
	value = cr.MustReadYAMLStrMap("test: 2.2")
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
	value = cr.MustReadYAMLStrMap("test: 2\ntest2: 2.2")
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
	value = cr.MustReadYAMLStrMap("test: 2") // YAML
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, map[interface{}]interface{}{"test": int64(2)})
	value = cr.MustReadJSONStr(`{"test": 2}`) // JSON
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, map[interface{}]interface{}{"test": int64(2)})
	value = cr.MustReadYAMLStrMap("test: 2.0") // YAML
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
	value = cr.MustReadJSONStr(`{"test": 2.0}`) // JSON
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)

	valueType = cr.MustReadYAMLStrMap("STRING: INT|FLOAT")
	value = cr.MustReadYAMLStrMap("test: 2.2")
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, map[interface{}]interface{}{"test": 2.2})
	value = map[string]int{"test": 2}
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, map[interface{}]interface{}{"test": int64(2)})

	valueType = cr.MustReadYAMLStrMap("mean: FLOAT\nsum: INT")
	value = cr.MustReadYAMLStrMap("mean: 2.2\nsum: 4")
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	value = cr.MustReadYAMLStrMap("mean: 2.2\nsum: 4.4")
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
	value = cr.MustReadYAMLStrMap("mean: test\nsum: 4")
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
	value = cr.MustReadYAMLStrMap("mean: 2.2")
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
	value = cr.MustReadYAMLStrMap("mean: 2.2\nsum: null")
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	value = cr.MustReadYAMLStrMap("mean: 2.2\nsum: 4\nextra: test")
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)

	valueType = []string{"INT"}
	value = []int{1, 2, 3}
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, []interface{}{int64(1), int64(2), int64(3)})
	value = []float64{1.1, 2.2, 3.3}
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
	value = []float64{1, 2, 3}
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)

	valueType = []string{"FLOAT"}
	value = []float64{1.1, 2.2, 3.3}
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, []interface{}{1.1, 2.2, 3.3})
	value = []float64{1, 2, 3}
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, []interface{}{float64(1), float64(2), float64(3)})
	value = []int{1, 2, 3}
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, []interface{}{float64(1), float64(2), float64(3)})

	valueType = []string{"FLOAT|INT|BOOL"}
	value = []float64{1.1, 2.2, 3.3}
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, []interface{}{float64(1.1), float64(2.2), float64(3.3)})
	value = []int{1, 2, 3}
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, []interface{}{int64(1), int64(2), int64(3)})
	value = []float64{1, 2, 3}
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, []interface{}{float64(1), float64(2), float64(3)})
	value = []interface{}{int64(1), float64(2), float64(2.2), true, false}
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, []interface{}{int64(1), float64(2), float64(2.2), true, false})
	value = []interface{}{true, "str"}
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)

	valueType = []string{"FLOAT|INT|BOOL|STRING"}
	value = []interface{}{int64(1), float64(2), float64(2.2), "str", false}
	casted, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)
	require.Equal(t, casted, []interface{}{int64(1), float64(2), float64(2.2), "str", false})

	valueType = cr.MustReadYAMLStrMap(
		`
     map: {STRING: FLOAT}
     str: STRING
     floats: [FLOAT]
     map2:
       STRING:
         lat: FLOAT
         lon:
           a: INT
           b: [STRING]
           c: {mean: FLOAT, sum: [INT], stddev: {STRING: INT}}
         bools: [BOOL]
         anything: [BOOL|INT|FLOAT|STRING]
    `)

	value = cr.MustReadYAMLStrMap(
		`
     map: {a: 2.2, b: 3}
     str: test1
     floats: [2.2, 3.3, 4.4]
     map2:
       testA:
         lat: 9.9
         lon:
           a: 17
           b: [test1, test2, test3]
           c: {mean: 8.8, sum: [3, 2, 1], stddev: {a: 1, b: 2}}
         bools: [true]
         anything: []
       testB:
         lat: 3.14
         lon:
           a: 88
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
         anything: [10, 2.2, test, false]
    `)
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)

	value = cr.MustReadYAMLStrMap(
		`
     map: {a: 2.2, b: 3}
     str: test1
     floats: [2.2, 3.3, 4.4]
     map2:
       testA:
         lat: 9.9
         lon:
           a: 17
           b: [test1, test2, test3]
           c: null
         bools: [true]
         anything: []
       testB:
         lat: null
         lon:
           a: 88
           b: null
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
         anything: [10, 2.2, test, false]
       testC: null
    `)
	_, err = userconfig.CastValue(value, valueType)
	require.NoError(t, err)

	value = cr.MustReadYAMLStrMap(
		`
     map: {a: 2.2, b: 3}
     str: test1
     floats: [2.2, 3.3, 4.4]
     map2:
       testA:
         lat: 9.9
         lon:
           a: 17
           b: [test1, test2, test3]
           c: {mean: 8.8, sum: [3, 2, 1], stddev: {a: 1, b: 2}}
         bools: [true]
         anything: []
       testB:
         lat: 3.14
         lon:
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
         anything: [10, 2.2, test, false]
    `)
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)

	value = cr.MustReadYAMLStrMap(
		`
     map: {a: 2.2, b: 3}
     str: test1
     floats: [2.2, 3.3, 4.4]
     map2:
       testA:
         lat: 9.9
         lon:
           a: 17
           b: [test1, test2, test3]
           c: {mean: 8.8, sum: [3, 2, 1], stddev: {a: 1, b: 2}}
         bools: [true]
         anything: []
       testB:
         lat: 3.14
         lon:
           a: 88.8
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
         anything: [10, 2.2, test, false]
    `)
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)

	value = cr.MustReadYAMLStrMap(
		`
     map: {a: 2.2, b: 3}
     str: test1
     floats: [2.2, 3.3, 4.4]
     map2:
       testA:
         lat: 9.9
         lon:
           a: 17
           b: [test1, test2, test3]
           c: {mean: 8.8, sum: [3, 2, 1], stddev: {a: 1, b: 2}}
         bools: [true]
         anything: []
       testB:
         lat: 3.14
         lon:
           a: 88
           b: [testX, testY, 2]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
         anything: [10, 2.2, test, false]
    `)
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)

	value = cr.MustReadYAMLStrMap(
		`
     map: {a: 2.2, b: 3}
     str: test1
     floats: [2.2, 3.3, 4.4]
     map2:
       testA:
         lat: 9.9
         lon:
           a: 17
           b: [test1, test2, test3]
           c: {mean: 8.8, sum: [3, 2, 1], stddev: {a: 1, b: test}}
         bools: [true]
         anything: []
       testB:
         lat: 3.14
         lon:
           a: 88
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
         anything: [10, 2.2, test, false]
    `)
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)

	value = cr.MustReadYAMLStrMap(
		`
     map: {a: 2.2, b: 3}
     str: test1
     floats: [2.2, 3.3, 4.4]
     map2:
       testA:
         lat: 9.9
         lon:
           a: 17
           b: [test1, test2, test3]
           c: {mean: 8.8, sum: [3, 2, 1], stddev: {a: 1, b: 2}}
         bools: [true]
         anything: []
       testB:
         lat: 3.14
         lon:
           a: 88
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: true
         anything: [10, 2.2, test, false]
    `)
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)

	value = cr.MustReadYAMLStrMap(
		`
     map: {a: 2.2, b: 3}
     str: test1
     floats: [2.2, 3.3, 4.4]
     map2:
       testA:
         lat: 9.9
         lon:
           a: 17
           b: [test1, test2, test3]
           c: {mean: 8.8, sum: [3, 2, 1], stddev: {a: 1, b: 2}}
         bools: [true]
         anything: []
       testB:
         lat: 3.14
         lon:
           a: 88
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [1, 2, 3]
         anything: [10, 2.2, test, false]
    `)
	_, err = userconfig.CastValue(value, valueType)
	require.Error(t, err)
}

func TestCheckValueRuntimeTypesMatch(t *testing.T) {
	var schemaType interface{}
	var runtimeType interface{}

	schemaType = "INT"
	runtimeType = "INT"
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = "FLOAT"
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = "FLOAT|INT"
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = "FLOAT|INT"
	runtimeType = "FLOAT|INT"
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = "INT|FLOAT"
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = "FLOAT"
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = "INT"
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = "STRING"
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = []string{"INT"}
	runtimeType = []string{"INT"}
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = []string{"STRING"}
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = []string{"BOOL"}
	runtimeType = []string{"BOOL"}
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = []string{"INT"}
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = []string{"FLOAT|INT"}
	runtimeType = []string{"INT|FLOAT"}
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = []string{"INT"}
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = []string{"BOOL|FLOAT|INT"}
	runtimeType = []string{"FLOAT|INT|BOOL"}
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = []string{"INT|FLOAT"}
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = []string{"FLOAT"}
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = []string{"INT"}
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = []string{"BOOL"}
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = []string{"STRING"}
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = "FLOAT"
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = cr.MustReadYAMLStrMap("STRING: FLOAT")
	runtimeType = cr.MustReadYAMLStrMap("STRING: FLOAT")
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("STRING: INT")
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = cr.MustReadYAMLStrMap("STRING: [INT|FLOAT]")
	runtimeType = cr.MustReadYAMLStrMap("STRING: [FLOAT|INT]")
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("STRING: [INT]")
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("STRING: [BOOL]")
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("STRING: INT")
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = cr.MustReadYAMLStrMap("INT|FLOAT: STRING")
	runtimeType = cr.MustReadYAMLStrMap("FLOAT|INT: STRING")
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("INT: STRING")
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("BOOL: STRING")
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = cr.MustReadYAMLStrMap("mean: FLOAT\nsum: INT")
	runtimeType = cr.MustReadYAMLStrMap("mean: FLOAT\nsum: INT")
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("sum: INT\nmean: FLOAT")
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("sum: INT\nmean: INT")
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("mean: FLOAT")
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("mean: FLOAT\nsum: INT\nextra: STRING")
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = cr.MustReadYAMLStrMap("mean: FLOAT\nsum: INT|FLOAT")
	runtimeType = cr.MustReadYAMLStrMap("mean: FLOAT\nsum: FLOAT|INT")
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("sum: FLOAT\nmean: FLOAT")
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("sum: INT\nmean: FLOAT")
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap("sum: INT\nmean: INT")
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))

	schemaType = cr.MustReadYAMLStrMap(
		`
      map: {STRING: FLOAT}
      str: STRING
      floats: [FLOAT]
      map2:
        STRING:
          lat: FLOAT
          lon:
            a: INT|FLOAT
            b: [STRING]
            c: {mean: FLOAT, sum: [INT], stddev: {STRING: INT|FLOAT}}
            d: [BOOL]
     `)
	runtimeType = cr.MustReadYAMLStrMap(
		`
      floats: [FLOAT]
      str: STRING
      map2:
        STRING:
          lat: FLOAT
          lon:
            c: {sum: [INT], mean: FLOAT, stddev: {STRING: FLOAT|INT}}
            b: [STRING]
            a: FLOAT|INT
            d: [BOOL]
      map: {STRING: FLOAT}
     `)
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap(
		`
      floats: [FLOAT]
      str: STRING
      map2:
        STRING:
          lat: FLOAT
          lon:
            c: {sum: [INT], mean: FLOAT, stddev: {STRING: FLOAT|INT}}
            b: [STRING]
            a: INT
            d: [BOOL]
      map: {STRING: FLOAT}
     `)
	require.NoError(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap(
		`
      floats: [FLOAT]
      str: STRING
      map2:
        STRING:
          lat: FLOAT
          lon:
            c: {sum: [INT], mean: FLOAT, stddev: {STRING: FLOAT|INT}}
            b: STRING
            a: FLOAT|INT
            d: [BOOL]
      map: {STRING: FLOAT}
     `)
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap(
		`
      floats: [FLOAT]
      str: STRING
      map2:
        STRING:
          lat: FLOAT
          lon:
            c: {sum: [INT], stddev: {STRING: FLOAT|INT}}
            b: [STRING]
            a: FLOAT|INT
            d: [BOOL]
      map: {STRING: FLOAT}
     `)
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
	runtimeType = cr.MustReadYAMLStrMap(
		`
      floats: [FLOAT]
      str: STRING
      map2:
        STRING:
          lat: FLOAT
          lon:
            c: {sum: [INT], mean: FLOAT, stddev: {STRING: FLOAT|INT}}
            b: [STRING]
            a: FLOAT|INT
            d: BOOL
      map: {STRING: FLOAT}
     `)
	require.Error(t, userconfig.CheckValueRuntimeTypesMatch(runtimeType, schemaType))
}
