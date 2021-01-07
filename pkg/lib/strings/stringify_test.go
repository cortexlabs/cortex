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

package strings

import (
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/stretchr/testify/require"
)

type MyFloat float64
type MyString string
type MyNested map[MyFloat][]MyString

type Test struct {
	Str      MyString
	Float    float64 `json:"float"`
	Test2Ptr *Test2
	Test3    Test3
}

type Test2 struct {
	Bool     bool `json:"bool"`
	Float    *float64
	Strs     *[]string
	Test3Ptr *Test3 `yaml:"test3"`
}

type Test3 struct {
	Strs []string
	Map  map[interface{}]interface{} `json:"map"`
}

type TestInterface interface {
	Test()
}

func (t *Test2) Test() {}
func (t Test3) Test()  {}

func TestObj(t *testing.T) {
	var a interface{}
	require.Equal(t, "<null>", Obj(a))
	var b []string
	require.Equal(t, "<null>", Obj(b))
	var c *string
	require.Equal(t, "<null>", Obj(c))

	require.Equal(t, "true", Obj(true))
	require.Equal(t, "2.2", Obj(float32(2.2)))
	require.Equal(t, "2.0", Obj(float32(2)))
	require.Equal(t, "2.0", Obj(float64(2)))
	require.Equal(t, "3", Obj(int(3)))
	require.Equal(t, "3", Obj(pointer.Int(3)))
	require.Equal(t, "-3", Obj(int8(-3)))
	require.Equal(t, "3", Obj(int16(3)))
	require.Equal(t, "-3", Obj(int32(-3)))
	require.Equal(t, "3", Obj(int64(3)))
	require.Equal(t, "4", Obj(int(4)))
	require.Equal(t, "4", Obj(int8(4)))
	require.Equal(t, "4", Obj(int16(4)))
	require.Equal(t, "4", Obj(int32(4)))
	require.Equal(t, "4", Obj(int64(4)))
	require.Equal(t, `""`, Obj(""))
	require.Equal(t, `"test"`, Obj("test"))
	require.Equal(t, `"test"`, Obj(pointer.String("test")))

	var myFloat MyFloat = 2
	require.Equal(t, "2.0", Obj(myFloat))
	var myString MyString = "test"
	require.Equal(t, `"test"`, Obj(myString))

	strSlice := []string{"a", "b", "c"}
	require.Equal(t, `["a", "b", "c"]`, ObjFlat(strSlice))
	require.Equal(t, `["a", "b", "c"]`, ObjFlat(&strSlice))
	intSlice := []int8{1, 2, 3}
	require.Equal(t, `[1, 2, 3]`, ObjFlat(intSlice))
	mixedSlice := []interface{}{int(1), float64(2), "three", ""}
	require.Equal(t, `[1, 2.0, "three", ""]`, ObjFlat(mixedSlice))
	nestedSlice := []interface{}{int(1), "three", strSlice, []interface{}{"a", []float64{1, 2.2}}}
	require.Equal(t, `[1, "three", ["a", "b", "c"], ["a", [1.0, 2.2]]]`, ObjFlat(nestedSlice))

	strMap := map[string]string{"b": "y", "a": "x"}
	require.Equal(t, `{"a": "x", "b": "y"}`, ObjFlat(strMap))
	require.Equal(t, `{"a": "x", "b": "y"}`, ObjFlat(&strMap))
	mixedMap := map[interface{}]interface{}{"1": "a", true: strMap, int(3): strSlice}
	require.Equal(t, `{"1": "a", 3: ["a", "b", "c"], true: {"a": "x", "b": "y"}}`, ObjFlat(mixedMap))

	myNested := MyNested{myFloat: []MyString{myString, myString}}
	require.Equal(t, `{2.0: ["test", "test"]}`, ObjFlat(myNested))

	emptyMap := map[interface{}]interface{}{}
	require.Equal(t, `{}`, Obj(emptyMap))
	require.Equal(t, `{}`, ObjFlat(emptyMap))

	emptyCollectionsInMap := map[interface{}]interface{}{"empty_map": map[interface{}]interface{}{}, "a": "b", "empty_slice": []interface{}{}}
	require.Equal(t, `{"a": "b", "empty_map": {}, "empty_slice": []}`, ObjFlat(emptyCollectionsInMap))
	require.Equal(t, `{
  "a": "b",
  "empty_map": {},
  "empty_slice": []
}`, Obj(emptyCollectionsInMap))

	testStruct := Test{
		Str:   myString,
		Float: 2,
		Test2Ptr: &Test2{
			Bool:  false,
			Float: pointer.Float64(1.7),
			Strs:  &strSlice,
			Test3Ptr: &Test3{
				Strs: []string{"a", "b", "c"},
				Map:  map[interface{}]interface{}{"1": "a", true: strMap, int(3): intSlice},
			},
		},
		Test3: Test3{
			Strs: nil,
			Map:  map[interface{}]interface{}{"1": nil, true: strMap, int(3): intSlice},
		},
	}
	testStructStr := `{"Str": "test", "float": 2.0, "Test2Ptr": {"bool": false, "Float": 1.7, "Strs": ["a", "b", "c"], "test3": {"Strs": ["a", "b", "c"], "map": {"1": "a", 3: [1, 2, 3], true: {"a": "x", "b": "y"}}}}, "Test3": {"Strs": <null>, "map": {"1": <null>, 3: [1, 2, 3], true: {"a": "x", "b": "y"}}}}`

	testStructStrMultiline := `{
  "Str": "test",
  "float": 2.0,
  "Test2Ptr": {
    "bool": false,
    "Float": 1.7,
    "Strs": [
      "a",
      "b",
      "c"
    ],
    "test3": {
      "Strs": [
        "a",
        "b",
        "c"
      ],
      "map": {
        "1": "a",
        3: [
          1,
          2,
          3
        ],
        true: {
          "a": "x",
          "b": "y"
        }
      }
    }
  },
  "Test3": {
    "Strs": <null>,
    "map": {
      "1": <null>,
      3: [
        1,
        2,
        3
      ],
      true: {
        "a": "x",
        "b": "y"
      }
    }
  }
}`
	test2SubStrMultiline := `{
  "bool": false,
  "Float": 1.7,
  "Strs": [
    "a",
    "b",
    "c"
  ],
  "test3": {
    "Strs": [
      "a",
      "b",
      "c"
    ],
    "map": {
      "1": "a",
      3: [
        1,
        2,
        3
      ],
      true: {
        "a": "x",
        "b": "y"
      }
    }
  }
}`
	test2SubStr := `{"bool": false, "Float": 1.7, "Strs": ["a", "b", "c"], "test3": {"Strs": ["a", "b", "c"], "map": {"1": "a", 3: [1, 2, 3], true: {"a": "x", "b": "y"}}}}`

	test3SubStrMultiline := `{
  "Strs": <null>,
  "map": {
    "1": <null>,
    3: [
      1,
      2,
      3
    ],
    true: {
      "a": "x",
      "b": "y"
    }
  }
}`
	test3SubStr := `{"Strs": <null>, "map": {"1": <null>, 3: [1, 2, 3], true: {"a": "x", "b": "y"}}}`

	require.Equal(t, testStructStr, ObjFlat(testStruct))
	require.Equal(t, testStructStr, ObjFlat(&testStruct))
	ptr := &testStruct
	require.Equal(t, testStructStr, ObjFlat(&ptr))

	var testInterface TestInterface
	testInterface = testStruct.Test2Ptr
	require.Equal(t, test2SubStr, ObjFlat(testInterface))
	require.Equal(t, test2SubStr, ObjFlat(&testInterface))
	testInterface = testStruct.Test3
	require.Equal(t, test3SubStr, ObjFlat(testInterface))
	require.Equal(t, test3SubStr, ObjFlat(&testInterface))

	require.Equal(t, testStructStrMultiline, Obj(testStruct))
	require.Equal(t, testStructStrMultiline, Obj(&testStruct))
	ptr = &testStruct
	require.Equal(t, testStructStrMultiline, Obj(&ptr))

	testInterface = testStruct.Test2Ptr
	require.Equal(t, test2SubStrMultiline, Obj(testInterface))
	require.Equal(t, test2SubStrMultiline, Obj(&testInterface))
	testInterface = testStruct.Test3
	require.Equal(t, test3SubStrMultiline, Obj(testInterface))
	require.Equal(t, test3SubStrMultiline, Obj(&testInterface))

}

func TestRound(t *testing.T) {
	require.Equal(t, Round(1.111, 2, 0), "1.11")
	require.Equal(t, Round(1.111, 3, 0), "1.111")
	require.Equal(t, Round(1.111, 4, 0), "1.111")
	require.Equal(t, Round(1.555, 2, 0), "1.56")
	require.Equal(t, Round(1.555, 3, 0), "1.555")
	require.Equal(t, Round(1.555, 4, 0), "1.555")
	require.Equal(t, Round(1.100, 2, 0), "1.1")

	require.Equal(t, Round(1.111, 2, 2), "1.11")
	require.Equal(t, Round(1.111, 3, 3), "1.111")
	require.Equal(t, Round(1.111, 4, 4), "1.1110")
	require.Equal(t, Round(1.555, 2, 2), "1.56")
	require.Equal(t, Round(1.555, 3, 3), "1.555")
	require.Equal(t, Round(1.555, 4, 4), "1.5550")
	require.Equal(t, Round(1.100, 2, 2), "1.10")

	require.Equal(t, Round(30, 0, 0), "30")
	require.Equal(t, Round(2, 1, 1), "2.0")
	require.Equal(t, Round(1, 2, 2), "1.00")
	require.Equal(t, Round(20, 3, 3), "20.000")

	require.Equal(t, Round(1.5555, 3, 2), "1.556")
	require.Equal(t, Round(1.5, 3, 2), "1.50")
	require.Equal(t, Round(1, 3, 2), "1.00")

	require.Equal(t, Round(1.5555, 3, 1), "1.556")
	require.Equal(t, Round(1.5, 3, 1), "1.5")
	require.Equal(t, Round(1, 3, 1), "1.0")

	require.Equal(t, Round(1.5555, 3, 4), "1.5560")
	require.Equal(t, Round(1.5, 3, 4), "1.5000")
	require.Equal(t, Round(1, 3, 4), "1.0000")

	require.Equal(t, Round(30, 0, 0), "30")
	require.Equal(t, Round(2, 1, 0), "2")
	require.Equal(t, Round(1, 2, 0), "1")
	require.Equal(t, Round(20, 3, 0), "20")
}
