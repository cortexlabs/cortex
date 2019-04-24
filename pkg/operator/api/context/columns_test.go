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

package context_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func TestGetColumnRuntimeTypes(t *testing.T) {
	var columnInputValues map[string]interface{}
	var expected map[string]interface{}

	rawColumns := context.RawColumns{
		"rfInt": &context.RawIntColumn{
			RawIntColumn: &userconfig.RawIntColumn{
				Type: userconfig.IntegerColumnType,
			},
		},
		"rfFloat": &context.RawFloatColumn{
			RawFloatColumn: &userconfig.RawFloatColumn{
				Type: userconfig.FloatColumnType,
			},
		},
		"rfStr": &context.RawStringColumn{
			RawStringColumn: &userconfig.RawStringColumn{
				Type: userconfig.StringColumnType,
			},
		},
	}

	columnInputValues = cr.MustReadYAMLStrMap("in: rfInt")
	expected = map[string]interface{}{"in": userconfig.IntegerColumnType}
	checkTestGetColumnRuntimeTypes(columnInputValues, rawColumns, expected, t)

	columnInputValues = cr.MustReadYAMLStrMap("in: rfStr")
	expected = map[string]interface{}{"in": userconfig.StringColumnType}
	checkTestGetColumnRuntimeTypes(columnInputValues, rawColumns, expected, t)

	columnInputValues = cr.MustReadYAMLStrMap("in: [rfFloat]")
	expected = map[string]interface{}{"in": []userconfig.ColumnType{userconfig.FloatColumnType}}
	checkTestGetColumnRuntimeTypes(columnInputValues, rawColumns, expected, t)

	columnInputValues = cr.MustReadYAMLStrMap("in: [rfInt, rfFloat, rfStr, rfInt]")
	expected = map[string]interface{}{"in": []userconfig.ColumnType{userconfig.IntegerColumnType, userconfig.FloatColumnType, userconfig.StringColumnType, userconfig.IntegerColumnType}}
	checkTestGetColumnRuntimeTypes(columnInputValues, rawColumns, expected, t)

	columnInputValues = cr.MustReadYAMLStrMap("in1: [rfInt, rfFloat]\nin2: rfStr")
	expected = map[string]interface{}{"in1": []userconfig.ColumnType{userconfig.IntegerColumnType, userconfig.FloatColumnType}, "in2": userconfig.StringColumnType}
	checkTestGetColumnRuntimeTypes(columnInputValues, rawColumns, expected, t)

	columnInputValues = cr.MustReadYAMLStrMap("in: 1")
	checkErrTestGetColumnRuntimeTypes(columnInputValues, rawColumns, t)

	columnInputValues = cr.MustReadYAMLStrMap("in: [1, 2, 3]")
	checkErrTestGetColumnRuntimeTypes(columnInputValues, rawColumns, t)

	columnInputValues = cr.MustReadYAMLStrMap("in: {in: rfInt}")
	checkErrTestGetColumnRuntimeTypes(columnInputValues, rawColumns, t)

	columnInputValues = cr.MustReadYAMLStrMap("in: rfMissing")
	checkErrTestGetColumnRuntimeTypes(columnInputValues, rawColumns, t)

	columnInputValues = cr.MustReadYAMLStrMap("in: [rfMissing]")
	checkErrTestGetColumnRuntimeTypes(columnInputValues, rawColumns, t)
}

func checkTestGetColumnRuntimeTypes(columnInputValues map[string]interface{}, rawColumns context.RawColumns, expected map[string]interface{}, t *testing.T) {
	runtimeTypes, err := context.GetColumnRuntimeTypes(columnInputValues, rawColumns)
	require.NoError(t, err)
	require.Equal(t, expected, runtimeTypes)
}

func checkErrTestGetColumnRuntimeTypes(columnInputValues map[string]interface{}, rawColumns context.RawColumns, t *testing.T) {
	_, err := context.GetColumnRuntimeTypes(columnInputValues, rawColumns)
	require.Error(t, err)
}
