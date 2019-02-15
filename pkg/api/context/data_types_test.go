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

	"github.com/cortexlabs/cortex/pkg/api/context"
)

func TestDataTypeID(t *testing.T) {
	var type1 interface{}
	var type2 interface{}

	type1 = "STRING"
	type2 = "STRING"
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = "INT"
	checkDataTypeIDsNotEqual(type1, type2, t)

	type1 = "INT|FLOAT"
	type2 = "INT|FLOAT"
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = "FLOAT|INT"
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = "INT|STRING"
	checkDataTypeIDsNotEqual(type1, type2, t)
	type2 = "INT"
	checkDataTypeIDsNotEqual(type1, type2, t)

	type1 = "INT|FLOAT|BOOL|STRING"
	type2 = "INT|FLOAT|BOOL|STRING"
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = "FLOAT|STRING|INT|BOOL"
	checkDataTypeIDsEqual(type1, type2, t)

	type1 = []string{"INT|FLOAT|BOOL|STRING"}
	type2 = []string{"INT|FLOAT|BOOL|STRING"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = []string{"FLOAT|STRING|INT|BOOL"}
	checkDataTypeIDsEqual(type1, type2, t)

	type1 = []string{"INT|FLOAT"}
	type2 = []string{"INT|FLOAT"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = []string{"FLOAT|INT"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = []string{"INT|STRING"}
	checkDataTypeIDsNotEqual(type1, type2, t)
	type2 = "INT|FLOAT"
	checkDataTypeIDsNotEqual(type1, type2, t)

	type1 = map[string]string{"INT|FLOAT": "STRING|BOOL"}
	type2 = map[string]string{"INT|FLOAT": "STRING|BOOL"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = map[string]string{"FLOAT|INT": "BOOL|STRING"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = map[string]string{"INT|FLOAT": "STRING|INT"}
	checkDataTypeIDsNotEqual(type1, type2, t)

	type1 = map[string]string{"a": "INT|BOOL", "b": "STRING|FLOAT"}
	type2 = map[string]string{"a": "INT|BOOL", "b": "STRING|FLOAT"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = map[string]string{"b": "STRING|FLOAT", "a": "INT|BOOL"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = map[string]string{"a": "BOOL|INT", "b": "FLOAT|STRING"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = map[string]string{"b": "FLOAT|STRING", "a": "BOOL|INT"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = map[string]string{"a": "INT|FLOAT", "b": "STRING|FLOAT"}
	checkDataTypeIDsNotEqual(type1, type2, t)
	type2 = map[string]string{"a": "INT|BOOL", "b": "STRING|INT"}
	checkDataTypeIDsNotEqual(type1, type2, t)

	type1 = "STRING_COLUMN"
	type2 = "STRING_COLUMN"
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = "INT_COLUMN"
	checkDataTypeIDsNotEqual(type1, type2, t)

	type1 = "INT_COLUMN|FLOAT_COLUMN|BOOL_COLUMN|STRING_COLUMN"
	type2 = "INT_COLUMN|FLOAT_COLUMN|BOOL_COLUMN|STRING_COLUMN"
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = "FLOAT_COLUMN|STRING_COLUMN|INT_COLUMN|BOOL_COLUMN"
	checkDataTypeIDsEqual(type1, type2, t)

	type1 = []string{"INT_COLUMN|FLOAT_COLUMN"}
	type2 = []string{"INT_COLUMN|FLOAT_COLUMN"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = []string{"FLOAT_COLUMN|INT_COLUMN"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = []string{"INT_COLUMN|STRING_COLUMN"}
	checkDataTypeIDsNotEqual(type1, type2, t)
	type2 = "INT_COLUMN|FLOAT_COLUMN"
	checkDataTypeIDsNotEqual(type1, type2, t)

	type1 = map[string]string{"INT_COLUMN|FLOAT_COLUMN": "STRING_COLUMN|BOOL_COLUMN"}
	type2 = map[string]string{"INT_COLUMN|FLOAT_COLUMN": "STRING_COLUMN|BOOL_COLUMN"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = map[string]string{"FLOAT_COLUMN|INT_COLUMN": "BOOL_COLUMN|STRING_COLUMN"}
	checkDataTypeIDsEqual(type1, type2, t)
	type2 = map[string]string{"INT_COLUMN|FLOAT_COLUMN": "STRING_COLUMN|INT_COLUMN"}
	checkDataTypeIDsNotEqual(type1, type2, t)
}

func checkDataTypeIDsEqual(type1 interface{}, type2 interface{}, t *testing.T) {
	id1 := context.DataTypeID(type1)
	id2 := context.DataTypeID(type2)
	require.Equal(t, id1, id2)
}

func checkDataTypeIDsNotEqual(type1 interface{}, type2 interface{}, t *testing.T) {
	id1 := context.DataTypeID(type1)
	id2 := context.DataTypeID(type2)
	require.NotEqual(t, id1, id2)
}
