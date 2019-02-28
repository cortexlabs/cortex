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

package cast_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cortexlabs/cortex/pkg/lib/cast"
)

func TestInterfaceToInterfaceSlice(t *testing.T) {
	slice1 := []string{"test1", "test2", "test3"}
	slice2 := []interface{}{"test1", "test2", "test3"}
	var ok bool

	_, ok = cast.InterfaceToInterfaceSlice(slice1)
	require.True(t, ok)

	_, ok = cast.InterfaceToInterfaceSlice(slice2)
	require.True(t, ok)

	_, ok = cast.InterfaceToStrSlice(slice1)
	require.True(t, ok)

	_, ok = cast.InterfaceToStrSlice(slice2)
	require.True(t, ok)
}

func TestInterfaceToFloat64(t *testing.T) {
	var out float64
	var ok bool

	out, ok = cast.InterfaceToFloat64(float64(1.1))
	require.True(t, ok)
	require.Equal(t, float64(1.1), out)

	out, ok = cast.InterfaceToFloat64(float32(2.2))
	require.True(t, ok)
	require.Equal(t, float64(float32(2.2)), out)

	out, ok = cast.InterfaceToFloat64(int(3))
	require.True(t, ok)
	require.Equal(t, float64(3), out)

	_, ok = cast.InterfaceToFloat64("test")
	require.False(t, ok)
}

func TestInterfaceToIntDowncast(t *testing.T) {
	var out int
	var ok bool

	out, ok = cast.InterfaceToIntDowncast(int(1))
	require.True(t, ok)
	require.Equal(t, int(1), out)

	out, ok = cast.InterfaceToIntDowncast(float32(2))
	require.True(t, ok)
	require.Equal(t, int(2), out)

	out, ok = cast.InterfaceToIntDowncast(float32(2.0))
	require.True(t, ok)
	require.Equal(t, int(2), out)

	_, ok = cast.InterfaceToIntDowncast(float32(2.2))
	require.False(t, ok)

	out, ok = cast.InterfaceToIntDowncast(float64(3))
	require.True(t, ok)
	require.Equal(t, int(3), out)

	out, ok = cast.InterfaceToIntDowncast(float64(3.0))
	require.True(t, ok)
	require.Equal(t, int(3), out)

	_, ok = cast.InterfaceToIntDowncast(float64(3.3))
	require.False(t, ok)

	_, ok = cast.InterfaceToIntDowncast("test")
	require.False(t, ok)
}

func TestInterfaceToInt(t *testing.T) {
	var out int
	var ok bool

	out, ok = cast.InterfaceToInt(int(1))
	require.True(t, ok)
	require.Equal(t, int(1), out)

	_, ok = cast.InterfaceToInt(float32(2))
	require.False(t, ok)

	_, ok = cast.InterfaceToInt("test")
	require.False(t, ok)
}

func TestInterfaceToInt8Downcast(t *testing.T) {
	var out int8
	var ok bool

	out, ok = cast.InterfaceToInt8Downcast(int(1))
	require.True(t, ok)
	require.Equal(t, int8(1), out)

	out, ok = cast.InterfaceToInt8Downcast(float32(2))
	require.True(t, ok)
	require.Equal(t, int8(2), out)

	out, ok = cast.InterfaceToInt8Downcast(float32(2.0))
	require.True(t, ok)
	require.Equal(t, int8(2), out)

	_, ok = cast.InterfaceToInt8Downcast(float32(2.2))
	require.False(t, ok)

	out, ok = cast.InterfaceToInt8Downcast(float64(3))
	require.True(t, ok)
	require.Equal(t, int8(3), out)

	out, ok = cast.InterfaceToInt8Downcast(float64(3.0))
	require.True(t, ok)
	require.Equal(t, int8(3), out)

	_, ok = cast.InterfaceToInt8Downcast(float64(3.3))
	require.False(t, ok)

	_, ok = cast.InterfaceToInt8Downcast("test")
	require.False(t, ok)

	_, ok = cast.InterfaceToInt8Downcast(int(999999))
	require.False(t, ok)
}

func TestInterfaceToInt8(t *testing.T) {
	var out int8
	var ok bool

	out, ok = cast.InterfaceToInt8(int(1))
	require.True(t, ok)
	require.Equal(t, int8(1), out)

	_, ok = cast.InterfaceToInt8(float32(2))
	require.False(t, ok)

	_, ok = cast.InterfaceToInt8("test")
	require.False(t, ok)
}

func TestInterfaceToInterfaceInterfaceMap(t *testing.T) {
	var ok bool
	var in interface{}
	var casted map[interface{}]interface{}
	var expected map[interface{}]interface{}

	in = map[string]string{"test": "str"}
	expected = map[interface{}]interface{}{"test": "str"}
	casted, ok = cast.InterfaceToInterfaceInterfaceMap(in)
	require.True(t, ok)
	require.Equal(t, expected, casted)

	in = map[int]bool{2: true}
	expected = map[interface{}]interface{}{int(2): true}
	casted, ok = cast.InterfaceToInterfaceInterfaceMap(in)
	require.True(t, ok)
	require.Equal(t, expected, casted)

	in = map[interface{}]float32{"test": float32(2.2)}
	expected = map[interface{}]interface{}{"test": float32(2.2)}
	casted, ok = cast.InterfaceToInterfaceInterfaceMap(in)
	require.True(t, ok)
	require.Equal(t, expected, casted)
}
