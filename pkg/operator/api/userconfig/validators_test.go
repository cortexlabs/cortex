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
	"testing"

	"github.com/stretchr/testify/require"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
)

func TestValidateOutputSchema(t *testing.T) {
	var err error

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`STRING`))
	require.NoError(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`STRING|INT`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`STRING_COLUMN`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`bad`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`[STRING]`))
	require.NoError(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`[STRING, INT]`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`[STRING|INT]`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`[STRING_COLUMN]`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`[bad]`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`{mean: FLOAT, stddev: FLOAT}`))
	require.NoError(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`{_type: FLOAT}`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`{_mean: FLOAT}`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`{INT: FLOAT}`))
	require.NoError(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`{INT: INT|FLOAT}`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`{INT|FLOAT: FLOAT}`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`{INT: FLOAT_COLUMN}`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`{INT_COLUMN: FLOAT}`))
	require.Error(t, err)

	_, err = ValidateOutputSchema(cr.MustReadYAMLStr(
		`{INT: FLOAT, FLOAT: FLOAT}`))
	require.Error(t, err)
}

func checkCastOutputValueEqual(t *testing.T, outputSchemaYAML string, valueYAML string, expected interface{}) {
	outputSchema, err := ValidateOutputSchema(cr.MustReadYAMLStr(outputSchemaYAML))
	require.NoError(t, err)
	casted, err := CastOutputValue(cr.MustReadYAMLStr(valueYAML), outputSchema)
	require.NoError(t, err)
	require.Equal(t, casted, expected)

	// All output schemas are valid input schemas, so test those too
	checkCastInputValueEqual(t, outputSchemaYAML, valueYAML, expected)
}

func checkCastOutputValueNoError(t *testing.T, outputSchemaYAML string, valueYAML string) {
	outputSchema, err := ValidateOutputSchema(cr.MustReadYAMLStr(outputSchemaYAML))
	require.NoError(t, err)
	_, err = CastOutputValue(cr.MustReadYAMLStr(valueYAML), outputSchema)
	require.NoError(t, err)

	// All output schemas are valid input schemas, so test those too
	checkCastInputValueNoError(t, outputSchemaYAML, valueYAML)
}

func checkCastOutputValueError(t *testing.T, outputSchemaYAML string, valueYAML string) {
	outputSchema, err := ValidateOutputSchema(cr.MustReadYAMLStr(outputSchemaYAML))
	require.NoError(t, err)
	_, err = CastOutputValue(cr.MustReadYAMLStr(valueYAML), outputSchema)
	require.Error(t, err)

	// All output schemas are valid input schemas, so test those too
	checkCastInputValueError(t, outputSchemaYAML, valueYAML)
}

func TestCastOutputValue(t *testing.T) {
	checkCastOutputValueEqual(t, `INT`, `2`, int64(2))
	checkCastOutputValueError(t, `INT`, `test`)
	checkCastOutputValueError(t, `INT`, `2.2`)
	checkCastOutputValueEqual(t, `FLOAT`, `2`, float64(2))
	checkCastOutputValueError(t, `FLOAT`, `test`)
	checkCastOutputValueEqual(t, `BOOL`, `true`, true)
	checkCastOutputValueEqual(t, `STRING`, `str`, "str")
	checkCastOutputValueError(t, `STRING`, `1`)

	checkCastOutputValueEqual(t, `{STRING: FLOAT}`, `{test: 2.2, test2: 4.4}`,
		map[interface{}]interface{}{"test": 2.2, "test2": 4.4})
	checkCastOutputValueError(t, `{STRING: FLOAT}`, `{test: test2}`)
	checkCastOutputValueEqual(t, `{STRING: FLOAT}`, `{test: 2}`,
		map[interface{}]interface{}{"test": float64(2)})
	checkCastOutputValueEqual(t, `{STRING: INT}`, `{test: 2}`,
		map[interface{}]interface{}{"test": int64(2)})
	checkCastOutputValueError(t, `{STRING: INT}`, `{test: 2.0}`)

	checkCastOutputValueEqual(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, sum: 4}`,
		map[interface{}]interface{}{"mean": float64(2.2), "sum": int64(4)})
	checkCastOutputValueError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, sum: test}`)
	checkCastOutputValueError(t, `{mean: FLOAT, sum: INT}`, `{mean: false, sum: 4}`)
	checkCastOutputValueError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, 2: 4}`)
	checkCastOutputValueError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, sum: Null}`)
	checkCastOutputValueError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2}`)
	checkCastOutputValueError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, sum: Null}`)
	checkCastOutputValueError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, sum: 4, stddev: 2}`)

	checkCastOutputValueEqual(t, `[INT]`, `[1, 2]`,
		[]interface{}{int64(1), int64(2)})
	checkCastOutputValueError(t, `[INT]`, `[1.0, 2]`)
	checkCastOutputValueEqual(t, `[FLOAT]`, `[1.0, 2]`,
		[]interface{}{float64(1), float64(2)})

	outputSchemaYAML :=
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
    `

	checkCastOutputValueNoError(t, outputSchemaYAML, `
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
       testB:
         lat: 3.14
         lon:
           a: 88
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
    `)

	checkCastOutputValueError(t, outputSchemaYAML, `
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
       testB:
         lat: 3.14
         lon:
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
    `)

	checkCastOutputValueError(t, outputSchemaYAML, `
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
       testB:
         lat: 3.14
         lon:
           a: 88.8
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
    `)

	checkCastOutputValueError(t, outputSchemaYAML, `
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
       testB:
         lat: 3.14
         lon:
           a: 88
           b: [testX, testY, 2]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
    `)

	checkCastOutputValueError(t, outputSchemaYAML, `
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
       testB:
         lat: 3.14
         lon:
           a: 88
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [true, false, true]
    `)

	checkCastOutputValueError(t, outputSchemaYAML, `
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
       testB:
         lat: 3.14
         lon:
           a: 88
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: true
    `)

	checkCastOutputValueError(t, outputSchemaYAML, `
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
       testB:
         lat: 3.14
         lon:
           a: 88
           b: [testX, testY, testZ]
           c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         bools: [1, 2, 3]
    `)
}

func checkCastInputValueEqual(t *testing.T, inputSchemaYAML string, valueYAML string, expected interface{}) {
	inputSchema, err := ValidateInputSchema(cr.MustReadYAMLStr(inputSchemaYAML), false)
	require.NoError(t, err)
	casted, err := CastInputValue(cr.MustReadYAMLStr(valueYAML), inputSchema)
	require.NoError(t, err)
	require.Equal(t, casted, expected)
}

func checkCastInputValueError(t *testing.T, inputSchemaYAML string, valueYAML string) {
	inputSchema, err := ValidateInputSchema(cr.MustReadYAMLStr(inputSchemaYAML), false)
	require.NoError(t, err)
	_, err = CastInputValue(cr.MustReadYAMLStr(valueYAML), inputSchema)
	require.Error(t, err)
}

func checkCastInputValueNoError(t *testing.T, inputSchemaYAML string, valueYAML string) {
	inputSchema, err := ValidateInputSchema(cr.MustReadYAMLStr(inputSchemaYAML), false)
	require.NoError(t, err)
	_, err = CastInputValue(cr.MustReadYAMLStr(valueYAML), inputSchema)
	require.NoError(t, err)
}

func TestCastInputValue(t *testing.T) {
	// Note: all test cases in TestCastOutputValue also test CastInputValue() since output schemas are valid input schemas

	checkCastInputValueEqual(t, `FLOAT|INT`, `2`, int64(2))
	checkCastInputValueEqual(t, `INT|FLOAT`, `2`, int64(2))
	checkCastInputValueEqual(t, `FLOAT|INT`, `2.2`, float64(2.2))
	checkCastInputValueEqual(t, `INT|FLOAT`, `2.2`, float64(2.2))
	checkCastInputValueError(t, `STRING`, `2`)
	checkCastInputValueEqual(t, `STRING|FLOAT`, `2`, float64(2))
	checkCastInputValueEqual(t, `{_type: [INT], _max_count: 2}`, `[2]`, []interface{}{int64(2)})
	checkCastInputValueError(t, `{_type: [INT], _max_count: 2}`, `[2, 3, 4]`)
	checkCastInputValueEqual(t, `{_type: [INT], _min_count: 2}`, `[2, 3, 4]`, []interface{}{int64(2), int64(3), int64(4)})
	checkCastInputValueError(t, `{_type: [INT], _min_count: 2}`, `[2]`)
	checkCastInputValueError(t, `{_type: INT, _optional: true}`, `Null`)
	checkCastInputValueError(t, `{_type: INT, _optional: true}`, ``)
	checkCastInputValueEqual(t, `{_type: INT, _allow_null: true}`, `Null`, nil)
	checkCastInputValueEqual(t, `{_type: INT, _allow_null: true}`, ``, nil)
	checkCastInputValueError(t, `{_type: {a: INT}}`, `Null`)
	checkCastInputValueError(t, `{_type: {a: INT}, _optional: true}`, `Null`)
	checkCastInputValueEqual(t, `{_type: {a: INT}, _allow_null: true}`, `Null`, nil)
	checkCastInputValueEqual(t, `{_type: {a: INT}}`, `{a: 2}`, map[interface{}]interface{}{"a": int64(2)})
	checkCastInputValueError(t, `{_type: {a: INT}}`, `{a: Null}`)
	checkCastInputValueError(t, `{a: {_type: INT, _optional: false}}`, `{a: Null}`)
	checkCastInputValueError(t, `{a: {_type: INT, _optional: false}}`, `{}`)
	checkCastInputValueError(t, `{a: {_type: INT, _optional: true}}`, `{a: Null}`)
	checkCastInputValueEqual(t, `{a: {_type: INT, _optional: true}}`, `{}`, map[interface{}]interface{}{})
	checkCastInputValueEqual(t, `{a: {_type: INT, _allow_null: true}}`, `{a: Null}`, map[interface{}]interface{}{"a": nil})
	checkCastInputValueError(t, `{a: {_type: INT, _allow_null: true}}`, `{}`)
	checkCastInputValueEqual(t, `{a: {_type: INT, _allow_null: true, _optional: true}}`, `{}`, map[interface{}]interface{}{})
}

func TestValidateInputSchema(t *testing.T) {
	var inputSchema, inputSchema2, inputSchema3, inputSchema4 *InputSchema
	var err error

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`STRING`), false)
	require.NoError(t, err)
	inputSchema2, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`_type: STRING`), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema, inputSchema2)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`STRING_COLUMN`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`STRING_COLUMN`), true)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`INFERRED_COLUMN`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`BAD_COLUMN`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: STRING
     _default: test
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: STRING_COLUMN
     _default: test
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: STRING
     _default: Null
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: STRING
     _default: 2
    `), false)
	require.Error(t, err)

	// Lists

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`[STRING]`), false)
	require.NoError(t, err)
	inputSchema2, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`_type: [STRING]`), false)
	require.NoError(t, err)
	inputSchema3, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
       - _type: STRING
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema, inputSchema2)
	require.Equal(t, inputSchema, inputSchema3)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`[STRING|INT]`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`[STRING_COLUMN|INT_COLUMN]`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`[STRING_COLUMN|INT_COLUMN]`), true)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`[STRING|INT_COLUMN]`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _default: [test1, test2, test3]
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING_COLUMN]
     _default: [test1, test2, test3]
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _default: [test1, 2, test3]
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING|INT]
     _default: [test1, 2, test3]
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING|FLOAT]
     _default: [test1, 2, test3]
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _default: test1
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _min_count: 2
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _min_count: 2
     _max_count: 2
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _min_count: 2
     _max_count: 1
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _default: [test1]
     _min_count: 2
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _default: [test1, test2]
     _min_count: 2
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _min_count: -1
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _min_count: test
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _default: [test1, test2, test3]
     _max_count: 2
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [STRING]
     _default: [test1, test2]
     _max_count: 2
    `), false)
	require.NoError(t, err)

	// Maps

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`arg1: STRING`), false)
	require.NoError(t, err)
	inputSchema2, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
       _type: STRING
    `), false)
	require.NoError(t, err)
	inputSchema3, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {arg1: STRING}
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema, inputSchema2)
	require.Equal(t, inputSchema, inputSchema3)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`arg1: STRING_COLUMN`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`arg1: STRING_COLUMN`), true)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`STRING_COLUMN: STRING`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`STRING_COLUMN: STRING`), true)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`_arg1: STRING`), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`arg1: test`), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`STRING: test`), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`STRING_COLUMN: test`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {arg1: STRING}
     _min_count: 2
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: INT}
     _default: {test: 2}
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {FLOAT: INT}
     _default: {2: 2}
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: INT}
     _default: {test: test}
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: INT|STRING}
     _default: {test: test}
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: INT_COLUMN}
     _min_count: 2
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING_COLUMN: INT}
     _min_count: 2
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING_COLUMN: INT}
     _default: {test: 2}
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING_COLUMN: INT_COLUMN}
     _min_count: 2
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING_COLUMN: INT_COLUMN|STRING}
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING_COLUMN: INT_COLUMN|STRING_COLUMN}
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: INT}
     _min_count: 2
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
       _type: STRING
       _optional: true
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
       _type: STRING
       _default: test
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
       _type: STRING
       _default: 2
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
       _type: STRING
       _default: Null
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
       _type: STRING
       _default: Null
    `), false)
	require.Error(t, err)

	// Mixed

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`[[STRING]]`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [[STRING]]
     _default: [[test1, test2]]
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [[STRING_COLUMN]]
     _default: [[test1, test2]]
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     - arg1: STRING
       arg2: INT
    `), false)
	require.NoError(t, err)
	inputSchema2, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
     - arg1: STRING
       arg2: INT
    `), false)
	require.NoError(t, err)
	inputSchema3, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     - arg1: {_type: STRING}
       arg2: {_type: INT}
    `), false)
	require.NoError(t, err)
	inputSchema4, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
     - arg1:
         _type: STRING
       arg2:
         _type: INT
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema, inputSchema2)
	require.Equal(t, inputSchema, inputSchema3)
	require.Equal(t, inputSchema, inputSchema4)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     - arg1:
         _type: STRING
         _default: test
       arg2:
         _type: INT
         _default: 2
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     - arg1:
         _type:
           arg_a: STRING
           arg_b:
             _type: INT
             _default: 1
       arg2:
         _type: INT
         _default: 2
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg_a:
       arg1:
         _type:
           arg_a: STRING
           arg_b:
             _type: INT
             _default: 1
       arg2:
         _type: INT
         _default: 2
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
       INT:
         arg_a: INT
         arg_b:
           _type: STRING
           _default: test
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     - arg1:
         INT:
           arg_a: INT
           arg_b:
             _type: STRING
             _default: test
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
     - INT:
         arg_a: INT
         arg_b:
           _type: STRING
           _default: test
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
       2: STRING
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
       _type: {2: STRING}
       _default: {2: test}
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     arg1:
       2:
         _type: STRING
         _default: test
    `), false)
	require.NoError(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`[{INT_COLUMN: STRING|INT}]`), false)
	require.NoError(t, err)
	inputSchema2, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [{INT_COLUMN: STRING|INT}]
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema, inputSchema2)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {BOOL|FLOAT: INT|STRING}`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {mean: FLOAT, stddev: FLOAT}`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {STRING: {lat: FLOAT, lon: FLOAT}}`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {STRING: {lat: FLOAT, lon: [FLOAT]}}`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {STRING: {FLOAT: INT}}`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {STRING: {FLOAT: [INT]}}`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {STRING: {lat: FLOAT, lon: {lat2: FLOAT, lon2: INT}}}`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map6: {STRING: {lat: FLOAT, lon: {lat2: FLOAT, lon2: {INT: STRING}}}}`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map6: {STRING: {lat: FLOAT, lon: {lat2: FLOAT, lon2: {INT: STRING}, mean: BOOL}}}`), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
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
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {STRING: INT, INT: FLOAT}`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {STRING: INT, INT: [FLOAT]}`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {mean: FLOAT, INT: FLOAT}`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {mean: FLOAT, INT: [FLOAT]}`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {STRING: {lat: FLOAT, STRING: FLOAT}}`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`map: {STRING: {STRING: test}}`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`cols: [STRING_COLUMN, INT_COLUMN]`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`cols: [STRING_COLUMNs]`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`cols: [STRING_COLUMN|BAD]`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`cols: Null`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`cols: 1`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`cols: [1]`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`cols: []`), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     float: FLOAT_COLUMN
     int: INT_COLUMN
     str: STRING_COLUMN
     int_list: FLOAT_LIST_COLUMN
     float_list: INT_LIST_COLUMN
     str_list: STRING_LIST_COLUMN
     num1: FLOAT_COLUMN|INT_COLUMN
     num2: INT_COLUMN|FLOAT_COLUMN
     num3: STRING_COLUMN|INT_COLUMN
     num4: INT_COLUMN|FLOAT_COLUMN|STRING_COLUMN
     num5: STRING_COLUMN|INT_COLUMN|FLOAT_COLUMN
     num6: STRING_LIST_COLUMN|INT_LIST_COLUMN|FLOAT_LIST_COLUMN
     num7: STRING_COLUMN|INT_LIST_COLUMN|FLOAT_LIST_COLUMN
     nums1: [INT_COLUMN]
     nums2: [FLOAT_COLUMN]
     nums3: [INT_COLUMN|FLOAT_COLUMN]
     nums4: [FLOAT_COLUMN|INT_COLUMN]
     nums5: [STRING_COLUMN|INT_COLUMN|FLOAT_COLUMN]
     nums6: [INT_LIST_COLUMN]
     nums7: [INT_LIST_COLUMN|STRING_LIST_COLUMN]
     nums8: [INT_LIST_COLUMN|STRING_COLUMN]
     float_scalar: FLOAT
     int_scalar: INT
     str_scalar: STRING
     bool_scalar: BOOL
     num1_scalar: FLOAT|INT
     num2_scalar: INT|FLOAT
     num3_scalar: STRING|INT
     num4_scalar: INT|FLOAT|STRING
     num5_scalar: STRING|INT|FLOAT
     nums1_scalar: [INT]
     nums2_scalar: [FLOAT]
     nums3_scalar: [INT|FLOAT]
     nums4_scalar: [FLOAT|INT]
     nums5_scalar: [STRING|INT|FLOAT]
     nums6_scalar: [STRING|INT|FLOAT|BOOL]
    `), false)
	require.NoError(t, err)

	// Casting defaults

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: INT
     _default: 2
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, int64(2))

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: INT
     _default: test
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: INT
     _default: 2.2
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: FLOAT
     _default: 2
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, float64(2))

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: FLOAT|INT
     _default: 2
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, int64(2))

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: BOOL
     _default: true
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, true)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: FLOAT}
     _default: {test: 2.2, test2: 4.4}
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, map[interface{}]interface{}{"test": 2.2, "test2": 4.4})

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: FLOAT}
     _default: {test: test2}
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: FLOAT}
     _default: {test: 2}
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, map[interface{}]interface{}{"test": float64(2)})

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: FLOAT}
     _default: {test: 2.0}
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, map[interface{}]interface{}{"test": float64(2)})

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: INT}
     _default: {test: 2}
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, map[interface{}]interface{}{"test": int64(2)})

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {STRING: INT}
     _default: {test: 2.0}
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {mean: FLOAT, sum: INT}
     _default: {mean: 2.2, sum: 4}
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, map[interface{}]interface{}{"mean": float64(2.2), "sum": int64(4)})

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {mean: FLOAT, sum: INT}
     _default: {mean: 2.2, sum: test}
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {mean: FLOAT, sum: INT}
     _default: {mean: false, sum: 4}
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {mean: FLOAT, sum: INT}
     _default: {mean: 2.2, 2: 4}
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {mean: FLOAT, sum: INT}
     _default: {mean: 2.2, sum: Null}
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {mean: FLOAT, sum: INT}
     _default: {mean: 2.2}
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: {mean: FLOAT, sum: INT}
     _default: {mean: 2.2, sum: 4, stddev: 2}
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [INT]
     _default: [1, 2]
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, []interface{}{int64(1), int64(2)})

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [INT]
     _default: [1.0, 2]
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [FLOAT]
     _default: [1.0, 2]
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, []interface{}{float64(1), float64(2)})

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [FLOAT|INT]
     _default: [1.0, 2]
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, []interface{}{float64(1), int64(2)})

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [FLOAT|INT|BOOL]
     _default: [1.0, 2, true, test]
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type: [FLOAT|INT|BOOL|STRING]
     _default: [1.0, 2, true, test]
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, []interface{}{float64(1), int64(2), true, "test"})

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
       STRING:
         a:
           _type: INT
           _optional: true
         b:
           _type: [STRING]
           _optional: true
         c:
           _type: {mean: FLOAT, sum: [INT], stddev: {STRING: INT}}
           _optional: true
         d:
           _type: INT
           _default: 2
     _default:
       testA: {}
       testB:
         a: 88
         b: [testX, testY, testZ]
         c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         d: 17
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, map[interface{}]interface{}{
		"testA": map[interface{}]interface{}{},
		"testB": map[interface{}]interface{}{
			"a": int64(88),
			"b": []interface{}{"testX", "testY", "testZ"},
			"c": map[interface{}]interface{}{
				"mean":   float64(1.7),
				"sum":    []interface{}{int64(1)},
				"stddev": map[interface{}]interface{}{"z": int64(12)},
			},
			"d": int64(17),
		},
	})

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
       STRING:
         a:
           _type: INT
           _optional: true
         b:
           _type: [STRING]
         c:
           _type: {mean: FLOAT, sum: [INT], stddev: {STRING: INT}}
           _optional: true
         d:
           _type: INT
           _default: 2
     _default:
       testA: Null
    `), false)
	require.Error(t, err)

	inputSchema, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
       STRING:
         _allow_null: true
         _type:
           a:
             _type: INT
             _optional: true
           b:
             _type: [STRING]
           c:
             _type: {mean: FLOAT, sum: [INT], stddev: {STRING: INT}}
             _optional: true
           d:
             _type: INT
             _default: 2
     _default:
       testA: Null
    `), false)
	require.NoError(t, err)
	require.Equal(t, inputSchema.Default, map[interface{}]interface{}{
		"testA": nil,
	})

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
       STRING:
         a:
           _type: INT
           _optional: true
         b:
           _type: [STRING]
         c:
           _type: {mean: FLOAT, sum: [INT], stddev: {STRING: INT}}
           _optional: true
         d:
           _type: INT
           _default: 2
     _default:
       testA: {}
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
       STRING:
         a:
           _type: INT
           _optional: true
         b:
           _type: [STRING]
         c:
           _type: {mean: FLOAT, sum: [INT], stddev: {STRING: INT}}
           _optional: true
         d:
           _type: INT
           _default: 2
     _default:
       testA:
         a: 88
         c: {mean: 1.7, sum: [1], stddev: {z: 12}}
         d: 17
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
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
     _default:
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
    `), false)
	require.NoError(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
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
     _default:
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
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
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
     _default:
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
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
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
     _default:
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
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
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
     _default:
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
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
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
     _default:
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
    `), false)
	require.Error(t, err)

	_, err = ValidateInputSchema(cr.MustReadYAMLStr(
		`
     _type:
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
     _default:
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
    `), false)
	require.Error(t, err)
}
