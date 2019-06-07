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
	"testing"

	"github.com/stretchr/testify/require"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func checkValidateRuntimeTypesEqual(t *testing.T, schemaYAML string, inputYAML string, expected interface{}) {
	schema, err := userconfig.ValidateInputSchema(cr.MustReadYAMLStr(schemaYAML), false, false)
	require.NoError(t, err)
	input := cr.MustReadYAMLStr(inputYAML)
	casted, err := validateRuntimeTypes(input, schema, allResourcesMap, aggregators, transformers, false)
	require.NoError(t, err)
	require.Equal(t, expected, casted)
}

func checkValidateRuntimeTypesError(t *testing.T, schemaYAML string, inputYAML string) {
	schema, err := userconfig.ValidateInputSchema(cr.MustReadYAMLStr(schemaYAML), false, false)
	require.NoError(t, err)
	input := cr.MustReadYAMLStr(inputYAML)
	_, err = validateRuntimeTypes(input, schema, allResourcesMap, aggregators, transformers, false)
	require.Error(t, err)
}

func checkValidateRuntimeTypesNoError(t *testing.T, schemaYAML string, inputYAML string) {
	schema, err := userconfig.ValidateInputSchema(cr.MustReadYAMLStr(schemaYAML), false, false)
	require.NoError(t, err)
	input := cr.MustReadYAMLStr(inputYAML)
	_, err = validateRuntimeTypes(input, schema, allResourcesMap, aggregators, transformers, false)
	require.NoError(t, err)
}

func TestValidateRuntimeTypes(t *testing.T) {

	// Replacements

	checkValidateRuntimeTypesEqual(t, `STRING`, `@c1`, `🌝🌝🌝🌝🌝c1`)

	checkValidateRuntimeTypesEqual(t, `STRING`, `@c2`, `🌝🌝🌝🌝🌝c2`)
	checkValidateRuntimeTypesEqual(t, `STRING|INT`, `@c2`, `🌝🌝🌝🌝🌝c2`)
	checkValidateRuntimeTypesError(t, `INT`, `@c2`)
	checkValidateRuntimeTypesError(t, `FLOAT`, `@c2`)

	checkValidateRuntimeTypesError(t, `STRING`, `@c3`)
	checkValidateRuntimeTypesNoError(t, `
    map: {INT: FLOAT}
    map2: {a: FLOAT, b: FLOAT, c: INT}
    str: STRING
    floats: [FLOAT]
    list:
      - STRING:
          lat: INT
          lon:
            a: [STRING]
    `, `@c3`)
	checkValidateRuntimeTypesNoError(t, `
    map: {2: FLOAT, 3: FLOAT}
    map2: {STRING: FLOAT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@c3`)
	checkValidateRuntimeTypesError(t, `
    map: {2: FLOAT, 3: INT}
    map2: {a: FLOAT, b: FLOAT, c: INT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@c3`)
	checkValidateRuntimeTypesError(t, `
    map: {INT: FLOAT}
    map2: {STRING: INT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@c3`)

	checkValidateRuntimeTypesError(t, `STRING`, `@c4`)
	checkValidateRuntimeTypesNoError(t, `
    map: {INT: FLOAT}
    map2: {a: FLOAT, b: FLOAT, c: INT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@c4`)
	checkValidateRuntimeTypesNoError(t, `
    map: {2: FLOAT, 3: FLOAT}
    map2: {STRING: FLOAT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@c4`)
	checkValidateRuntimeTypesNoError(t, `
    map: {2: FLOAT, 3: INT}
    map2: {STRING: INT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@c4`)

	checkValidateRuntimeTypesError(t, `INT`, `@c5`)
	checkValidateRuntimeTypesEqual(t, `FLOAT`, `@c5`, `🌝🌝🌝🌝🌝c5`)
	checkValidateRuntimeTypesEqual(t, `FLOAT|INT`, `@c5`, `🌝🌝🌝🌝🌝c5`)
	checkValidateRuntimeTypesError(t, `STRING`, `@c5`)
	checkValidateRuntimeTypesError(t, `BOOL`, `@c5`)

	checkValidateRuntimeTypesEqual(t, `INT`, `@c6`, `🌝🌝🌝🌝🌝c6`)
	checkValidateRuntimeTypesEqual(t, `FLOAT`, `@c6`, `🌝🌝🌝🌝🌝c6`)
	checkValidateRuntimeTypesEqual(t, `INT|FLOAT`, `@c6`, `🌝🌝🌝🌝🌝c6`)
	checkValidateRuntimeTypesError(t, `STRING`, `@c6`)
	checkValidateRuntimeTypesError(t, `BOOL`, `@c6`)

	checkValidateRuntimeTypesEqual(t, `INT`, `@c7`, `🌝🌝🌝🌝🌝c7`)
	checkValidateRuntimeTypesEqual(t, `FLOAT`, `@c7`, `🌝🌝🌝🌝🌝c7`)
	checkValidateRuntimeTypesEqual(t, `FLOAT|INT`, `@c7`, `🌝🌝🌝🌝🌝c7`)
	checkValidateRuntimeTypesError(t, `STRING`, `@c7`)
	checkValidateRuntimeTypesError(t, `BOOL`, `@c7`)

	checkValidateRuntimeTypesError(t, `[INT]`, `@cb`)
	checkValidateRuntimeTypesEqual(t, `[FLOAT]`, `@cb`, `🌝🌝🌝🌝🌝cb`)
	checkValidateRuntimeTypesEqual(t, `[FLOAT|INT]`, `@cb`, `🌝🌝🌝🌝🌝cb`)
	checkValidateRuntimeTypesError(t, `[STRING]`, `@cb`)
	checkValidateRuntimeTypesError(t, `[BOOL]`, `@cb`)

	checkValidateRuntimeTypesEqual(t, `[INT]`, `@cc`, `🌝🌝🌝🌝🌝cc`)
	checkValidateRuntimeTypesEqual(t, `[FLOAT]`, `@cc`, `🌝🌝🌝🌝🌝cc`)
	checkValidateRuntimeTypesEqual(t, `[INT|FLOAT]`, `@cc`, `🌝🌝🌝🌝🌝cc`)
	checkValidateRuntimeTypesError(t, `[STRING]`, `@cc`)
	checkValidateRuntimeTypesError(t, `[BOOL]`, `@cc`)

	checkValidateRuntimeTypesEqual(t, `[INT]`, `@cd`, `🌝🌝🌝🌝🌝cd`)
	checkValidateRuntimeTypesEqual(t, `[FLOAT]`, `@cd`, `🌝🌝🌝🌝🌝cd`)
	checkValidateRuntimeTypesEqual(t, `[FLOAT|INT]`, `@cd`, `🌝🌝🌝🌝🌝cd`)
	checkValidateRuntimeTypesError(t, `[STRING]`, `@cd`)
	checkValidateRuntimeTypesError(t, `[BOOL]`, `@cd`)

	checkValidateRuntimeTypesEqual(t, `{a: INT, b: INT}`, `@c8`, `🌝🌝🌝🌝🌝c8`)
	checkValidateRuntimeTypesEqual(t, `{STRING: INT}`, `@c8`, `🌝🌝🌝🌝🌝c8`)
	checkValidateRuntimeTypesEqual(t, `{a: FLOAT, b: INT}`, `@c8`, `🌝🌝🌝🌝🌝c8`)
	checkValidateRuntimeTypesEqual(t, `{STRING: FLOAT}`, `@c8`, `🌝🌝🌝🌝🌝c8`)
	checkValidateRuntimeTypesEqual(t, `{a: FLOAT}`, `@c8`, `🌝🌝🌝🌝🌝c8`)
	checkValidateRuntimeTypesError(t, `{a: FLOAT, b: INT, c: INT}`, `@c8`)

	checkValidateRuntimeTypesEqual(t, `{a: INT, b: INT}`, `@c9`, `🌝🌝🌝🌝🌝c9`)
	checkValidateRuntimeTypesEqual(t, `{STRING: INT}`, `@c9`, `🌝🌝🌝🌝🌝c9`)
	checkValidateRuntimeTypesEqual(t, `{a: FLOAT, b: INT}`, `@c9`, `🌝🌝🌝🌝🌝c9`)
	checkValidateRuntimeTypesEqual(t, `{STRING: FLOAT}`, `@c9`, `🌝🌝🌝🌝🌝c9`)
	checkValidateRuntimeTypesEqual(t, `{a: FLOAT}`, `@c9`, `🌝🌝🌝🌝🌝c9`)
	checkValidateRuntimeTypesError(t, `{a: FLOAT, b: INT, c: INT}`, `@c9`)

	checkValidateRuntimeTypesEqual(t, `{a: INT, b: INT}`, `@ca`, `🌝🌝🌝🌝🌝ca`)
	checkValidateRuntimeTypesEqual(t, `{STRING: INT}`, `@ca`, `🌝🌝🌝🌝🌝ca`)
	checkValidateRuntimeTypesEqual(t, `{a: FLOAT, b: INT}`, `@ca`, `🌝🌝🌝🌝🌝ca`)
	checkValidateRuntimeTypesEqual(t, `{STRING: FLOAT}`, `@ca`, `🌝🌝🌝🌝🌝ca`)
	checkValidateRuntimeTypesEqual(t, `{a: FLOAT}`, `@ca`, `🌝🌝🌝🌝🌝ca`)
	checkValidateRuntimeTypesError(t, `{a: FLOAT, b: INT, c: INT}`, `@ca`)

	checkValidateRuntimeTypesEqual(t, `INT_COLUMN`, `@rc1`, `🌝🌝🌝🌝🌝rc1`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_COLUMN`, `@rc1`, `🌝🌝🌝🌝🌝rc1`)
	checkValidateRuntimeTypesError(t, `STRING_COLUMN`, `@rc1`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_COLUMN|STRING_COLUMN`, `@rc1`, `🌝🌝🌝🌝🌝rc1`)
	checkValidateRuntimeTypesError(t, `INT_LIST_COLUMN`, `@rc1`)
	checkValidateRuntimeTypesError(t, `[INT_COLUMN]`, `@rc1`)
	checkValidateRuntimeTypesError(t, `{INT_COLUMN: INT}`, `@rc1`)
	checkValidateRuntimeTypesError(t, `{k: INT_COLUMN}`, `@rc1`)

	checkValidateRuntimeTypesEqual(t, `INT_COLUMN`, `@rc2`, `🌝🌝🌝🌝🌝rc2`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_COLUMN`, `@rc2`, `🌝🌝🌝🌝🌝rc2`)
	checkValidateRuntimeTypesEqual(t, `STRING_COLUMN`, `@rc2`, `🌝🌝🌝🌝🌝rc2`)
	checkValidateRuntimeTypesEqual(t, `INT_LIST_COLUMN`, `@rc2`, `🌝🌝🌝🌝🌝rc2`)
	checkValidateRuntimeTypesEqual(t, `STRING_LIST_COLUMN`, `@rc2`, `🌝🌝🌝🌝🌝rc2`)
	checkValidateRuntimeTypesError(t, `[INT_COLUMN]`, `@rc2`)
	checkValidateRuntimeTypesError(t, `[STRING_COLUMN]`, `@rc2`)
	checkValidateRuntimeTypesError(t, `{INT_COLUMN: INT}`, `@rc2`)
	checkValidateRuntimeTypesError(t, `{k: INT_COLUMN}`, `@rc2`)

	checkValidateRuntimeTypesError(t, `INT_COLUMN`, `@rc3`)
	checkValidateRuntimeTypesError(t, `FLOAT_COLUMN`, `@rc3`)
	checkValidateRuntimeTypesEqual(t, `STRING_COLUMN`, `@rc3`, `🌝🌝🌝🌝🌝rc3`)
	checkValidateRuntimeTypesError(t, `STRING_LIST_COLUMN`, `@rc3`)
	checkValidateRuntimeTypesError(t, `[STRING_COLUMN]`, `@rc3`)

	checkValidateRuntimeTypesError(t, `INT_COLUMN`, `@rc4`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_COLUMN`, `@rc4`, `🌝🌝🌝🌝🌝rc4`)
	checkValidateRuntimeTypesEqual(t, `INT_COLUMN|FLOAT_COLUMN`, `@rc4`, `🌝🌝🌝🌝🌝rc4`)
	checkValidateRuntimeTypesError(t, `STRING_COLUMN`, `@rc4`)
	checkValidateRuntimeTypesError(t, `FLOAT_LIST_COLUMN`, `@rc4`)
	checkValidateRuntimeTypesError(t, `[FLOAT_COLUMN]`, `@rc4`)

	checkValidateRuntimeTypesEqual(t, `STRING`, `@agg1`, `🌝🌝🌝🌝🌝agg1`)
	checkValidateRuntimeTypesEqual(t, `INT|STRING`, `@agg1`, `🌝🌝🌝🌝🌝agg1`)
	checkValidateRuntimeTypesError(t, `INT`, `@agg1`)

	checkValidateRuntimeTypesEqual(t, `STRING`, `@agg2`, `🌝🌝🌝🌝🌝agg2`)
	checkValidateRuntimeTypesEqual(t, `INT`, `@agg2`, `🌝🌝🌝🌝🌝agg2`)
	checkValidateRuntimeTypesEqual(t, `{INT: BOOL}`, `@agg2`, `🌝🌝🌝🌝🌝agg2`)

	checkValidateRuntimeTypesError(t, `STRING`, `@agg3`)
	checkValidateRuntimeTypesNoError(t, `
    map: {INT: FLOAT}
    map2: {a: FLOAT, b: FLOAT, c: INT}
    str: STRING
    floats: [FLOAT]
    list:
      - STRING:
          lat: INT
          lon:
            a: [STRING]
    `, `@agg3`)
	checkValidateRuntimeTypesError(t, `
    map: {INT: INT}
    map2: {a: FLOAT, b: FLOAT, c: INT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@agg3`)
	checkValidateRuntimeTypesNoError(t, `
    map: {FLOAT: FLOAT}
    map2: {a: FLOAT, b: FLOAT, c: FLOAT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@agg3`)
	checkValidateRuntimeTypesNoError(t, `
    map: {INT: FLOAT}
    map2: {STRING: FLOAT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@agg3`)
	checkValidateRuntimeTypesError(t, `
    map: {INT: FLOAT}
    map2: {STRING: INT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@agg3`)
	checkValidateRuntimeTypesError(t, `
    map: {2: FLOAT, 3: FLOAT}
    map2: {a: FLOAT, b: FLOAT, c: INT}
    str: STRING
    floats: [FLOAT]
    list: [{STRING: {lat: INT, lon: {a: [STRING]}}}]
    `, `@agg3`)

	checkValidateRuntimeTypesEqual(t, `INT`, `@agg4`, `🌝🌝🌝🌝🌝agg4`)
	checkValidateRuntimeTypesEqual(t, `FLOAT`, `@agg4`, `🌝🌝🌝🌝🌝agg4`)
	checkValidateRuntimeTypesEqual(t, `FLOAT|INT`, `@agg4`, `🌝🌝🌝🌝🌝agg4`)
	checkValidateRuntimeTypesError(t, `STRING`, `@agg4`)
	checkValidateRuntimeTypesError(t, `BOOL`, `@agg4`)

	checkValidateRuntimeTypesError(t, `INT`, `@agg5`)
	checkValidateRuntimeTypesEqual(t, `FLOAT`, `@agg5`, `🌝🌝🌝🌝🌝agg5`)
	checkValidateRuntimeTypesEqual(t, `INT|FLOAT`, `@agg5`, `🌝🌝🌝🌝🌝agg5`)
	checkValidateRuntimeTypesError(t, `STRING`, `@agg5`)
	checkValidateRuntimeTypesError(t, `BOOL`, `@agg5`)

	checkValidateRuntimeTypesEqual(t, `[INT]`, `@agg8`, `🌝🌝🌝🌝🌝agg8`)
	checkValidateRuntimeTypesEqual(t, `[FLOAT]`, `@agg8`, `🌝🌝🌝🌝🌝agg8`)
	checkValidateRuntimeTypesEqual(t, `[FLOAT|INT]`, `@agg8`, `🌝🌝🌝🌝🌝agg8`)
	checkValidateRuntimeTypesError(t, `[STRING]`, `@agg8`)
	checkValidateRuntimeTypesError(t, `[BOOL]`, `@agg8`)

	checkValidateRuntimeTypesError(t, `[INT]`, `@agg9`)
	checkValidateRuntimeTypesEqual(t, `[FLOAT]`, `@agg9`, `🌝🌝🌝🌝🌝agg9`)
	checkValidateRuntimeTypesEqual(t, `[INT|FLOAT]`, `@agg9`, `🌝🌝🌝🌝🌝agg9`)
	checkValidateRuntimeTypesError(t, `[STRING]`, `@agg9`)
	checkValidateRuntimeTypesError(t, `[BOOL]`, `@agg9`)

	checkValidateRuntimeTypesEqual(t, `{a: INT, b: INT}`, `@agg6`, `🌝🌝🌝🌝🌝agg6`)
	checkValidateRuntimeTypesEqual(t, `{STRING: INT}`, `@agg6`, `🌝🌝🌝🌝🌝agg6`)
	checkValidateRuntimeTypesEqual(t, `{a: FLOAT, b: INT}`, `@agg6`, `🌝🌝🌝🌝🌝agg6`)
	checkValidateRuntimeTypesEqual(t, `{STRING: FLOAT}`, `@agg6`, `🌝🌝🌝🌝🌝agg6`)
	checkValidateRuntimeTypesEqual(t, `{a: FLOAT}`, `@agg6`, `🌝🌝🌝🌝🌝agg6`)
	checkValidateRuntimeTypesError(t, `{a: FLOAT, b: INT, c: INT}`, `@agg6`)

	checkValidateRuntimeTypesError(t, `{a: INT, b: INT}`, `@agg7`)
	checkValidateRuntimeTypesEqual(t, `{STRING: INT}`, `@agg7`, `🌝🌝🌝🌝🌝agg7`)
	checkValidateRuntimeTypesEqual(t, `{STRING: FLOAT}`, `@agg7`, `🌝🌝🌝🌝🌝agg7`)

	checkValidateRuntimeTypesEqual(t, `STRING_COLUMN`, `@tc1`, `🌝🌝🌝🌝🌝tc1`)
	checkValidateRuntimeTypesError(t, `INT_COLUMN`, `@tc1`)
	checkValidateRuntimeTypesEqual(t, `INT_COLUMN|STRING_COLUMN`, `@tc1`, `🌝🌝🌝🌝🌝tc1`)
	checkValidateRuntimeTypesError(t, `FLOAT_COLUMN`, `@tc1`)
	checkValidateRuntimeTypesError(t, `STRING_LIST_COLUMN`, `@tc1`)
	checkValidateRuntimeTypesError(t, `INT_LIST_COLUMN`, `@tc1`)
	checkValidateRuntimeTypesError(t, `FLOAT_LIST_COLUMN`, `@tc1`)
	checkValidateRuntimeTypesError(t, `[STRING_COLUMN]`, `@tc1`)
	checkValidateRuntimeTypesError(t, `[INT_COLUMN]`, `@tc1`)
	checkValidateRuntimeTypesError(t, `{STRING_COLUMN: INT}`, `@tc1`)
	checkValidateRuntimeTypesError(t, `{k: STRING_COLUMN}`, `@tc1`)

	checkValidateRuntimeTypesEqual(t, `STRING_COLUMN`, `@tc2`, `🌝🌝🌝🌝🌝tc2`)
	checkValidateRuntimeTypesEqual(t, `INT_COLUMN`, `@tc2`, `🌝🌝🌝🌝🌝tc2`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_COLUMN`, `@tc2`, `🌝🌝🌝🌝🌝tc2`)
	checkValidateRuntimeTypesEqual(t, `STRING_LIST_COLUMN`, `@tc2`, `🌝🌝🌝🌝🌝tc2`)
	checkValidateRuntimeTypesEqual(t, `INT_LIST_COLUMN`, `@tc2`, `🌝🌝🌝🌝🌝tc2`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_LIST_COLUMN`, `@tc2`, `🌝🌝🌝🌝🌝tc2`)
	checkValidateRuntimeTypesError(t, `[STRING_COLUMN]`, `@tc2`)
	checkValidateRuntimeTypesError(t, `[INT_COLUMN]`, `@tc2`)
	checkValidateRuntimeTypesError(t, `{STRING_COLUMN: INT}`, `@tc2`)
	checkValidateRuntimeTypesError(t, `{k: STRING_COLUMN}`, `@tc2`)

	checkValidateRuntimeTypesError(t, `STRING_COLUMN`, `@tc3`)
	checkValidateRuntimeTypesEqual(t, `INT_COLUMN`, `@tc3`, `🌝🌝🌝🌝🌝tc3`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_COLUMN`, `@tc3`, `🌝🌝🌝🌝🌝tc3`)
	checkValidateRuntimeTypesError(t, `STRING_LIST_COLUMN`, `@tc3`)
	checkValidateRuntimeTypesError(t, `INT_LIST_COLUMN`, `@tc3`)
	checkValidateRuntimeTypesError(t, `FLOAT_LIST_COLUMN`, `@tc3`)
	checkValidateRuntimeTypesError(t, `[STRING_COLUMN]`, `@tc3`)
	checkValidateRuntimeTypesError(t, `[INT_COLUMN]`, `@tc3`)
	checkValidateRuntimeTypesError(t, `{STRING_COLUMN: INT}`, `@tc3`)
	checkValidateRuntimeTypesError(t, `{k: STRING_COLUMN}`, `@tc3`)

	checkValidateRuntimeTypesError(t, `STRING_COLUMN`, `@tc4`)
	checkValidateRuntimeTypesError(t, `INT_COLUMN`, `@tc4`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_COLUMN`, `@tc4`, `🌝🌝🌝🌝🌝tc4`)
	checkValidateRuntimeTypesEqual(t, `INT_COLUMN|FLOAT_COLUMN`, `@tc4`, `🌝🌝🌝🌝🌝tc4`)
	checkValidateRuntimeTypesError(t, `STRING_LIST_COLUMN`, `@tc4`)
	checkValidateRuntimeTypesError(t, `INT_LIST_COLUMN`, `@tc4`)
	checkValidateRuntimeTypesError(t, `FLOAT_LIST_COLUMN`, `@tc4`)
	checkValidateRuntimeTypesError(t, `[STRING_COLUMN]`, `@tc4`)
	checkValidateRuntimeTypesError(t, `[INT_COLUMN]`, `@tc4`)
	checkValidateRuntimeTypesError(t, `{STRING_COLUMN: INT}`, `@tc4`)
	checkValidateRuntimeTypesError(t, `{k: STRING_COLUMN}`, `@tc4`)

	checkValidateRuntimeTypesError(t, `STRING_COLUMN`, `@tc5`)
	checkValidateRuntimeTypesError(t, `INT_COLUMN`, `@tc5`)
	checkValidateRuntimeTypesError(t, `FLOAT_COLUMN`, `@tc5`)
	checkValidateRuntimeTypesEqual(t, `STRING_LIST_COLUMN`, `@tc5`, `🌝🌝🌝🌝🌝tc5`)
	checkValidateRuntimeTypesEqual(t, `STRING_LIST_COLUMN|INT_COLUMN`, `@tc5`, `🌝🌝🌝🌝🌝tc5`)
	checkValidateRuntimeTypesError(t, `INT_LIST_COLUMN`, `@tc5`)
	checkValidateRuntimeTypesError(t, `FLOAT_LIST_COLUMN`, `@tc5`)
	checkValidateRuntimeTypesError(t, `[STRING_COLUMN]`, `@tc5`)
	checkValidateRuntimeTypesError(t, `[INT_COLUMN]`, `@tc5`)
	checkValidateRuntimeTypesError(t, `{STRING_COLUMN: INT}`, `@tc5`)
	checkValidateRuntimeTypesError(t, `{k: STRING_COLUMN}`, `@tc5`)

	checkValidateRuntimeTypesError(t, `STRING_COLUMN`, `@tc6`)
	checkValidateRuntimeTypesError(t, `INT_COLUMN`, `@tc6`)
	checkValidateRuntimeTypesError(t, `FLOAT_COLUMN`, `@tc6`)
	checkValidateRuntimeTypesError(t, `STRING_LIST_COLUMN`, `@tc6`)
	checkValidateRuntimeTypesEqual(t, `INT_LIST_COLUMN`, `@tc6`, `🌝🌝🌝🌝🌝tc6`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_LIST_COLUMN`, `@tc6`, `🌝🌝🌝🌝🌝tc6`)
	checkValidateRuntimeTypesError(t, `[STRING_COLUMN]`, `@tc6`)
	checkValidateRuntimeTypesError(t, `[INT_COLUMN]`, `@tc6`)
	checkValidateRuntimeTypesError(t, `{STRING_COLUMN: INT}`, `@tc6`)
	checkValidateRuntimeTypesError(t, `{k: STRING_COLUMN}`, `@tc6`)

	checkValidateRuntimeTypesError(t, `STRING_COLUMN`, `@tc7`)
	checkValidateRuntimeTypesError(t, `INT_COLUMN`, `@tc7`)
	checkValidateRuntimeTypesError(t, `FLOAT_COLUMN`, `@tc7`)
	checkValidateRuntimeTypesError(t, `STRING_LIST_COLUMN`, `@tc7`)
	checkValidateRuntimeTypesError(t, `INT_LIST_COLUMN`, `@tc7`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_LIST_COLUMN`, `@tc7`, `🌝🌝🌝🌝🌝tc7`)
	checkValidateRuntimeTypesEqual(t, `FLOAT_LIST_COLUMN|INT_LIST_COLUMN`, `@tc7`, `🌝🌝🌝🌝🌝tc7`)
	checkValidateRuntimeTypesError(t, `[STRING_COLUMN]`, `@tc7`)
	checkValidateRuntimeTypesError(t, `[INT_COLUMN]`, `@tc7`)
	checkValidateRuntimeTypesError(t, `{STRING_COLUMN: INT}`, `@tc7`)
	checkValidateRuntimeTypesError(t, `{k: STRING_COLUMN}`, `@tc7`)

	checkValidateRuntimeTypesEqual(t,
		`[INT_COLUMN]`,
		`[@tc3, @rc1]`,
		[]interface{}{"🌝🌝🌝🌝🌝tc3", "🌝🌝🌝🌝🌝rc1"})

	checkValidateRuntimeTypesEqual(t,
		`[FLOAT_COLUMN]`,
		`[@tc3, @rc1, @tc4, @rc4]`,
		[]interface{}{"🌝🌝🌝🌝🌝tc3", "🌝🌝🌝🌝🌝rc1", "🌝🌝🌝🌝🌝tc4", "🌝🌝🌝🌝🌝rc4"})

	checkValidateRuntimeTypesEqual(t,
		`[FLOAT]`,
		`[@c5, @c6, 2, 2.2, @agg4, @agg5]`,
		[]interface{}{"🌝🌝🌝🌝🌝c5", "🌝🌝🌝🌝🌝c6", float64(2), float64(2.2), "🌝🌝🌝🌝🌝agg4", "🌝🌝🌝🌝🌝agg5"})

	checkValidateRuntimeTypesEqual(t,
		`[FLOAT]`,
		`@cb`,
		"🌝🌝🌝🌝🌝cb")

	checkValidateRuntimeTypesEqual(t,
		`[FLOAT]`,
		`@cc`,
		"🌝🌝🌝🌝🌝cc")

	checkValidateRuntimeTypesEqual(t,
		`[FLOAT|INT]`,
		`[@c5, @c6, 2, 2.2, @agg4, @agg5]`,
		[]interface{}{"🌝🌝🌝🌝🌝c5", "🌝🌝🌝🌝🌝c6", int64(2), float64(2.2), "🌝🌝🌝🌝🌝agg4", "🌝🌝🌝🌝🌝agg5"})

	checkValidateRuntimeTypesEqual(t,
		`{2: INT_COLUMN, 3: INT}`,
		`{2: @tc3, 3: @agg4}`,
		map[interface{}]interface{}{int64(2): "🌝🌝🌝🌝🌝tc3", int64(3): "🌝🌝🌝🌝🌝agg4"})

	checkValidateRuntimeTypesEqual(t,
		`{2: FLOAT_COLUMN, 3: FLOAT}`,
		`{2: @tc3, 3: @agg4}`,
		map[interface{}]interface{}{int64(2): "🌝🌝🌝🌝🌝tc3", int64(3): "🌝🌝🌝🌝🌝agg4"})

	checkValidateRuntimeTypesEqual(t,
		`{FLOAT: FLOAT_COLUMN}`,
		`{2: @tc3, 3: @tc4, @agg4: @rc1, @agg5: @rc2, @c5: @rc4, @c6: @tc2}`,
		map[interface{}]interface{}{
			float64(2):  "🌝🌝🌝🌝🌝tc3",
			float64(3):  "🌝🌝🌝🌝🌝tc4",
			"🌝🌝🌝🌝🌝agg4": "🌝🌝🌝🌝🌝rc1",
			"🌝🌝🌝🌝🌝agg5": "🌝🌝🌝🌝🌝rc2",
			"🌝🌝🌝🌝🌝c5":   "🌝🌝🌝🌝🌝rc4",
			"🌝🌝🌝🌝🌝c6":   "🌝🌝🌝🌝🌝tc2",
		})

	checkValidateRuntimeTypesNoError(t, `
    FLOAT:
      map: {INT: FLOAT}
      map2: {a: FLOAT, b: FLOAT, c: INT}
      str: STRING
      floats: [FLOAT]
      list:
        - STRING:
            lat: INT
            lon:
              a: [STRING]
    `, `
    @agg4: @agg3
    @agg5: @c3
    @c5: @c4
    @c6:
      map: {2: 2.2, 3: 3}
      map2: {a: 2.2, b: 3, c: 4}
      str: test
      floats: [1.1, 2.2, 3.3]
      list:
        - key_1:
            lat: 17
            lon:
              a: [test1, test2, test3]
          key_2:
            lat: 88
            lon:
              a: [test4, test5, test6]
        - key_a:
            lat: 12
            lon:
              a: [test7, test8, test9]
    2.2:
      map: {2: 2.2, @c6: @c6, @agg4: @agg5, 3: @c5, 4: @agg5}
      map2: {a: 2.2, b: @c5, c: @agg4}
      str: @c1
      floats: [@c5, @c6, 2, 2.2, @agg4, @agg5]
      list:
        - key_1:
            lat: @c6
            lon:
              a: [test1, @agg1, test3]
          @agg1:
            lat: 88
            lon:
              a: @ce
        - @c1:
            lat: @agg4
            lon:
              a: @agga
          key_2:
            lat: 17
            lon: @cf
          key_3:
            lat: 41
            lon: @aggb
    `)

	checkValidateRuntimeTypesError(t, `
    FLOAT:
      map: {INT: FLOAT}
      map2: {a: FLOAT, b: FLOAT, c: INT}
      str: STRING
      floats: [FLOAT]
      list:
        - STRING:
            lat: INT
            lon:
              a: [STRING]
    `, `
    2.2:
      map: {2: 2.2, @c6: @c6, @agg4: @agg5, 3: @c5, 4: @agg5}
      map2: {a: 2.2, b: @c5, c: @agg5}
      str: @c1
      floats: [@c5, @c6, 2, 2.2, @agg4, @agg5]
      list:
        - key_1:
            lat: @c6
            lon:
              a: [test1, @agg1, test3]
          @agg1:
            lat: 88
            lon:
              a: @ce
    `)

	checkValidateRuntimeTypesNoError(t, `
    - a: FLOAT_COLUMN
      b: INT_COLUMN|STRING_COLUMN
      c: {1: INT_COLUMN, 2: FLOAT_COLUMN, 3: BOOL, 4: STRING}
      d: {INT: INT_COLUMN}
      e: {FLOAT_COLUMN: FLOAT|STRING}
      f: {INT_LIST_COLUMN|STRING_COLUMN: FLOAT_COLUMN}
      g: [FLOAT]
    `, `
    - a: @tc4
      b: @rc3
      c: {1: @rc1, 2: @tc3, 3: true, 4: @agg1}
      d: {1: @rc1, 2: @tc3, @c6: @rc2, @agg4: @tc2}
      e: {@tc3: @agg4, @rc4: test, @tc2: 2.2, @rc1: @c1, @tc4: @agg5, @rc2: 2}
      f: {@tc6: @tc4, @tc1: @rc2, @rc3: @rc1}
      g: [@c5, @c6, 2, 2.2, @agg4, @agg5]
    `)

	checkValidateRuntimeTypesError(t, `
    - a: FLOAT_COLUMN
      b: INT_COLUMN|STRING_COLUMN
      c: {1: INT_COLUMN, 2: FLOAT_COLUMN, 3: BOOL, 4: STRING}
      d: {INT: INT_COLUMN}
      e: {FLOAT_COLUMN: FLOAT|STRING}
      f: {INT_LIST_COLUMN|STRING_COLUMN: FLOAT_COLUMN}
      g: [FLOAT]
    `, `
    - a: @tc4
      b: @rc3
      c: {1: @rc1, 2: @tc3, 3: true, 4: @agg1}
      d: {1: @rc1, 2: @tc3, @c6: @rc2, @agg4: @tc2}
      e: {@tc3: @agg4, @rc4: test, @tc2: 2.2, @rc1: @c1, @tc4: @agg5, @rc2: 2}
      f: {@tc7: @tc4, @tc1: @rc2, @rc3: @rc1}
      g: [@c5, @c6, 2, 2.2, @agg4, @agg5]
    `)

	// No replacements

	checkValidateRuntimeTypesEqual(t, `INT`, `2`, int64(2))
	checkValidateRuntimeTypesError(t, `INT`, `test`)
	checkValidateRuntimeTypesError(t, `INT`, `2.2`)
	checkValidateRuntimeTypesEqual(t, `FLOAT`, `2`, float64(2))
	checkValidateRuntimeTypesError(t, `FLOAT`, `test`)
	checkValidateRuntimeTypesEqual(t, `BOOL`, `true`, true)
	checkValidateRuntimeTypesEqual(t, `STRING`, `str`, "str")
	checkValidateRuntimeTypesError(t, `STRING`, `1`)

	checkValidateRuntimeTypesEqual(t, `{STRING: FLOAT}`, `{test: 2.2, test2: 4.4}`,
		map[interface{}]interface{}{"test": 2.2, "test2": 4.4})
	checkValidateRuntimeTypesError(t, `{STRING: FLOAT}`, `{test: test2}`)
	checkValidateRuntimeTypesEqual(t, `{STRING: FLOAT}`, `{test: 2}`,
		map[interface{}]interface{}{"test": float64(2)})
	checkValidateRuntimeTypesEqual(t, `{STRING: INT}`, `{test: 2}`,
		map[interface{}]interface{}{"test": int64(2)})
	checkValidateRuntimeTypesError(t, `{STRING: INT}`, `{test: 2.0}`)

	checkValidateRuntimeTypesEqual(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, sum: 4}`,
		map[interface{}]interface{}{"mean": float64(2.2), "sum": int64(4)})
	checkValidateRuntimeTypesError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, sum: test}`)
	checkValidateRuntimeTypesError(t, `{mean: FLOAT, sum: INT}`, `{mean: false, sum: 4}`)
	checkValidateRuntimeTypesError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, 2: 4}`)
	checkValidateRuntimeTypesError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, sum: Null}`)
	checkValidateRuntimeTypesError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2}`)
	checkValidateRuntimeTypesError(t, `{mean: FLOAT, sum: INT}`, `{mean: 2.2, sum: 4, stddev: 2}`)

	checkValidateRuntimeTypesEqual(t, `[INT]`, `[1, 2]`,
		[]interface{}{int64(1), int64(2)})
	checkValidateRuntimeTypesError(t, `[INT]`, `[1.0, 2]`)
	checkValidateRuntimeTypesEqual(t, `[FLOAT]`, `[1.0, 2]`,
		[]interface{}{float64(1), float64(2)})

	schemaYAML :=
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

	checkValidateRuntimeTypesNoError(t, schemaYAML, `
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

	checkValidateRuntimeTypesError(t, schemaYAML, `
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

	checkValidateRuntimeTypesError(t, schemaYAML, `
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

	checkValidateRuntimeTypesError(t, schemaYAML, `
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

	checkValidateRuntimeTypesError(t, schemaYAML, `
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

	checkValidateRuntimeTypesError(t, schemaYAML, `
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

	checkValidateRuntimeTypesError(t, schemaYAML, `
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

	checkValidateRuntimeTypesEqual(t, `FLOAT|INT`, `2`, int64(2))
	checkValidateRuntimeTypesEqual(t, `INT|FLOAT`, `2`, int64(2))
	checkValidateRuntimeTypesEqual(t, `FLOAT|INT`, `2.2`, float64(2.2))
	checkValidateRuntimeTypesEqual(t, `INT|FLOAT`, `2.2`, float64(2.2))
	checkValidateRuntimeTypesError(t, `STRING`, `2`)
	checkValidateRuntimeTypesEqual(t, `STRING|FLOAT`, `2`, float64(2))
	checkValidateRuntimeTypesEqual(t, `{_type: [INT], _max_count: 2}`, `[2]`, []interface{}{int64(2)})
	checkValidateRuntimeTypesError(t, `{_type: [INT], _max_count: 2}`, `[2, 3, 4]`)
	checkValidateRuntimeTypesEqual(t, `{_type: [INT], _min_count: 2}`, `[2, 3, 4]`, []interface{}{int64(2), int64(3), int64(4)})
	checkValidateRuntimeTypesError(t, `{_type: [INT], _min_count: 2}`, `[2]`)
	checkValidateRuntimeTypesError(t, `{_type: INT, _optional: true}`, `Null`)
	checkValidateRuntimeTypesError(t, `{_type: INT, _optional: true}`, ``)
	checkValidateRuntimeTypesEqual(t, `{_type: INT, _allow_null: true}`, `Null`, nil)
	checkValidateRuntimeTypesEqual(t, `{_type: INT, _allow_null: true}`, ``, nil)
	checkValidateRuntimeTypesError(t, `{_type: {a: INT}}`, `Null`)
	checkValidateRuntimeTypesError(t, `{_type: {a: INT}, _optional: true}`, `Null`)
	checkValidateRuntimeTypesEqual(t, `{_type: {a: INT}, _allow_null: true}`, `Null`, nil)
	checkValidateRuntimeTypesEqual(t, `{_type: {a: INT}}`, `{a: 2}`, map[interface{}]interface{}{"a": int64(2)})
	checkValidateRuntimeTypesError(t, `{_type: {a: INT}}`, `{a: Null}`)
	checkValidateRuntimeTypesError(t, `{a: {_type: INT, _optional: false}}`, `{a: Null}`)
	checkValidateRuntimeTypesError(t, `{a: {_type: INT, _optional: false}}`, `{}`)
	checkValidateRuntimeTypesError(t, `{a: {_type: INT, _optional: true}}`, `{a: Null}`)
	checkValidateRuntimeTypesEqual(t, `{a: {_type: INT, _optional: true}}`, `{}`, map[interface{}]interface{}{})
	checkValidateRuntimeTypesEqual(t, `{a: {_type: INT, _allow_null: true}}`, `{a: Null}`, map[interface{}]interface{}{"a": nil})
	checkValidateRuntimeTypesError(t, `{a: {_type: INT, _allow_null: true}}`, `{}`)
	checkValidateRuntimeTypesEqual(t, `{a: {_type: INT, _allow_null: true, _optional: true}}`, `{}`, map[interface{}]interface{}{})
}

func TestValidateResourceReferences(t *testing.T) {
	var input, replaced interface{}
	var err error

	input = cr.MustReadYAMLStr(`@bad`)
	_, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.Error(t, err)

	input = cr.MustReadYAMLStr(`@rc1?test`)
	_, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.Error(t, err)

	input = cr.MustReadYAMLStr(`@rc1 test`)
	_, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.Error(t, err)

	input = cr.MustReadYAMLStr(`[@rc1 test]`)
	_, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.Error(t, err)

	input = cr.MustReadYAMLStr(`@rc1 test: 2.2`)
	_, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.Error(t, err)

	input = cr.MustReadYAMLStr(`2.2: @rc1 test`)
	_, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.Error(t, err)

	input = cr.MustReadYAMLStr(`str`)
	replaced, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.NoError(t, err)
	require.Equal(t, "str", replaced)

	input = cr.MustReadYAMLStr(`@rc1`)
	replaced, err = validateResourceReferences(input, nil, rawColsMap, allResourceConfigsMap)
	require.NoError(t, err)
	require.Equal(t, "b_rc1", replaced)
	replaced, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.NoError(t, err)
	require.Equal(t, "b_rc1", replaced)
	_, err = validateResourceReferences(input, nil, transformedColsMap, allResourceConfigsMap)
	require.Error(t, err)
	_, err = validateResourceReferences(input, nil, nil, allResourceConfigsMap)
	require.Error(t, err)

	input = cr.MustReadYAMLStr(`[@tc1, rc2, @rc1]`)
	replaced, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.NoError(t, err)
	require.Equal(t, []interface{}{"e_tc1", "rc2", "b_rc1"}, replaced)
	_, err = validateResourceReferences(input, nil, rawColsMap, allResourceConfigsMap)
	require.Error(t, err)
	_, err = validateResourceReferences(input, nil, transformedColsMap, allResourceConfigsMap)
	require.Error(t, err)

	input = cr.MustReadYAMLStr(`{@c5: 1, @agg4: @c6, @c6: @agg5}`)
	replaced, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.NoError(t, err)
	require.Equal(t, map[interface{}]interface{}{"a_c5": int64(1), "c_agg4": "a_c6", "a_c6": "c_agg5"}, replaced)

	input = cr.MustReadYAMLStr(`[@tc1, @bad, @rc1]`)
	_, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.Error(t, err)

	input = cr.MustReadYAMLStr(
		`
     map: {@agg1: @c1}
     str: @rc1
     floats: [@tc2]
     map2:
       map3:
         lat: @c2
         lon:
           @c3: agg2
           b: [@tc1, @agg3]
    `)
	replaced, err = validateResourceReferences(input, nil, allResourcesMap, allResourceConfigsMap)
	require.NoError(t, err)
	expected := cr.MustReadYAMLStr(
		`
     map: {c_agg1: a_c1}
     str: b_rc1
     floats: [e_tc2]
     map2:
       map3:
         lat: a_c2
         lon:
           a_c3: agg2
           b: [e_tc1, c_agg3]
    `)
	require.Equal(t, expected, replaced)
}
