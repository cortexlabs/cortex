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

package strings_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
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
	require.Equal(t, "<null>", s.Obj(a))
	var b []string
	require.Equal(t, "<null>", s.Obj(b))
	var c *string
	require.Equal(t, "<null>", s.Obj(c))

	require.Equal(t, "true", s.Obj(true))
	require.Equal(t, "2.2", s.Obj(float32(2.2)))
	require.Equal(t, "2.0", s.Obj(float32(2)))
	require.Equal(t, "2.0", s.Obj(float64(2)))
	require.Equal(t, "3", s.Obj(int(3)))
	require.Equal(t, "3", s.Obj(pointer.Int(3)))
	require.Equal(t, "-3", s.Obj(int8(-3)))
	require.Equal(t, "3", s.Obj(int16(3)))
	require.Equal(t, "-3", s.Obj(int32(-3)))
	require.Equal(t, "3", s.Obj(int64(3)))
	require.Equal(t, "4", s.Obj(int(4)))
	require.Equal(t, "4", s.Obj(int8(4)))
	require.Equal(t, "4", s.Obj(int16(4)))
	require.Equal(t, "4", s.Obj(int32(4)))
	require.Equal(t, "4", s.Obj(int64(4)))
	require.Equal(t, `""`, s.Obj(""))
	require.Equal(t, `"test"`, s.Obj("test"))
	require.Equal(t, `"test"`, s.Obj(pointer.String("test")))

	var myFloat MyFloat = 2
	require.Equal(t, "2.0", s.Obj(myFloat))
	var myString MyString = "test"
	require.Equal(t, `"test"`, s.Obj(myString))

	strSlice := []string{"a", "b", "c"}
	require.Equal(t, `["a", "b", "c"]`, s.ObjFlat(strSlice))
	require.Equal(t, `["a", "b", "c"]`, s.ObjFlat(&strSlice))
	intSlice := []int8{1, 2, 3}
	require.Equal(t, `[1, 2, 3]`, s.ObjFlat(intSlice))
	mixedSlice := []interface{}{int(1), float64(2), "three", ""}
	require.Equal(t, `[1, 2.0, "three", ""]`, s.ObjFlat(mixedSlice))
	nestedSlice := []interface{}{int(1), "three", strSlice, []interface{}{"a", []float64{1, 2.2}}}
	require.Equal(t, `[1, "three", ["a", "b", "c"], ["a", [1.0, 2.2]]]`, s.ObjFlat(nestedSlice))

	strMap := map[string]string{"b": "y", "a": "x"}
	require.Equal(t, `{"a": "x", "b": "y"}`, s.ObjFlat(strMap))
	require.Equal(t, `{"a": "x", "b": "y"}`, s.ObjFlat(&strMap))
	mixedMap := map[interface{}]interface{}{"1": "a", true: strMap, int(3): strSlice}
	require.Equal(t, `{"1": "a", 3: ["a", "b", "c"], true: {"a": "x", "b": "y"}}`, s.ObjFlat(mixedMap))

	myNested := MyNested{myFloat: []MyString{myString, myString}}
	require.Equal(t, `{2.0: ["test", "test"]}`, s.ObjFlat(myNested))

	emptyMap := map[interface{}]interface{}{}
	require.Equal(t, `{}`, s.Obj(emptyMap))
	require.Equal(t, `{}`, s.ObjFlat(emptyMap))

	emptyCollectionsInMap := map[interface{}]interface{}{"empty_map": map[interface{}]interface{}{}, "a": "b", "empty_slice": []interface{}{}}
	require.Equal(t, `{"a": "b", "empty_map": {}, "empty_slice": []}`, s.ObjFlat(emptyCollectionsInMap))
	require.Equal(t, `{
  "a": "b",
  "empty_map": {},
  "empty_slice": []
}`, s.Obj(emptyCollectionsInMap))

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

	require.Equal(t, testStructStr, s.ObjFlat(testStruct))
	require.Equal(t, testStructStr, s.ObjFlat(&testStruct))
	ptr := &testStruct
	require.Equal(t, testStructStr, s.ObjFlat(&ptr))

	var testInterface TestInterface
	testInterface = testStruct.Test2Ptr
	require.Equal(t, test2SubStr, s.ObjFlat(testInterface))
	require.Equal(t, test2SubStr, s.ObjFlat(&testInterface))
	testInterface = testStruct.Test3
	require.Equal(t, test3SubStr, s.ObjFlat(testInterface))
	require.Equal(t, test3SubStr, s.ObjFlat(&testInterface))

	require.Equal(t, testStructStrMultiline, s.Obj(testStruct))
	require.Equal(t, testStructStrMultiline, s.Obj(&testStruct))
	ptr = &testStruct
	require.Equal(t, testStructStrMultiline, s.Obj(&ptr))

	testInterface = testStruct.Test2Ptr
	require.Equal(t, test2SubStrMultiline, s.Obj(testInterface))
	require.Equal(t, test2SubStrMultiline, s.Obj(&testInterface))
	testInterface = testStruct.Test3
	require.Equal(t, test3SubStrMultiline, s.Obj(testInterface))
	require.Equal(t, test3SubStrMultiline, s.Obj(&testInterface))

}

func TestRound(t *testing.T) {
	require.Equal(t, strings.Round(1.111, 2, false), "1.11")
	require.Equal(t, strings.Round(1.111, 3, false), "1.111")
	require.Equal(t, strings.Round(1.111, 4, false), "1.111")
	require.Equal(t, strings.Round(1.555, 2, false), "1.56")
	require.Equal(t, strings.Round(1.555, 3, false), "1.555")
	require.Equal(t, strings.Round(1.555, 4, false), "1.555")
	require.Equal(t, strings.Round(1.100, 2, false), "1.1")

	require.Equal(t, strings.Round(1.111, 2, true), "1.11")
	require.Equal(t, strings.Round(1.111, 3, true), "1.111")
	require.Equal(t, strings.Round(1.111, 4, true), "1.1110")
	require.Equal(t, strings.Round(1.555, 2, true), "1.56")
	require.Equal(t, strings.Round(1.555, 3, true), "1.555")
	require.Equal(t, strings.Round(1.555, 4, true), "1.5550")
	require.Equal(t, strings.Round(1.100, 2, true), "1.10")

	require.Equal(t, strings.Round(30, 0, true), "30")
	require.Equal(t, strings.Round(2, 1, true), "2.0")
	require.Equal(t, strings.Round(1, 2, true), "1.00")
	require.Equal(t, strings.Round(20, 3, true), "20.000")

	require.Equal(t, strings.Round(30, 0, false), "30")
	require.Equal(t, strings.Round(2, 1, false), "2")
	require.Equal(t, strings.Round(1, 2, false), "1")
	require.Equal(t, strings.Round(20, 3, false), "20")
}
