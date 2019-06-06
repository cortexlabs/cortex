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

func checkValueCastValueEqual(t *testing.T, typeStr string, valueYAML string, expected interface{}) {
	valueType := ValueTypeFromString(typeStr)
	require.NotEqual(t, valueType, UnknownValueType)
	casted, err := valueType.CastValue(cr.MustReadYAMLStr(valueYAML))
	require.NoError(t, err)
	require.Equal(t, casted, expected)
}

func checkValueCastValueError(t *testing.T, typeStr string, valueYAML string) {
	valueType := ValueTypeFromString(typeStr)
	require.NotEqual(t, valueType, UnknownValueType)
	_, err := valueType.CastValue(cr.MustReadYAMLStr(valueYAML))
	require.Error(t, err)
}

func TestValueCastValue(t *testing.T) {
	checkValueCastValueEqual(t, `INT`, `2`, int64(2))
	checkValueCastValueError(t, `STRING`, `2`)
	checkValueCastValueEqual(t, `STRING`, `test`, "test")
	checkValueCastValueEqual(t, `FLOAT`, `2`, float64(2))
	checkValueCastValueError(t, `BOOL`, `2`)
	checkValueCastValueEqual(t, `BOOL`, `true`, true)
}
