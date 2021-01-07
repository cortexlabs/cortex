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

package configreader

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlattenAllStrValues(t *testing.T) {
	var input interface{}
	var expected []string

	input = "test"
	expected = []string{"test"}
	CheckFlattenAllStrValues(input, expected, t)

	input = []interface{}{"test1", "test2", "test3"}
	expected = []string{"test1", "test2", "test3"}
	CheckFlattenAllStrValues(input, expected, t)

	input = map[interface{}]interface{}{
		"k1": "test1",
		"k2": "test2",
		"k3": "test3",
	}
	expected = []string{"test1", "test2", "test3"}
	CheckFlattenAllStrValues(input, expected, t)

	input = map[interface{}]interface{}{
		"k1": []interface{}{"test1"},
		"k2": "test2",
		"k3": []interface{}{"test3", "test4"},
	}
	expected = []string{"test1", "test2", "test3", "test4"}
	CheckFlattenAllStrValues(input, expected, t)

	input = map[string]interface{}{
		"k1": []interface{}{"test1"},
		"k2": "test2",
		"k3": []interface{}{"test3", "test4"},
		"k4": map[string]interface{}{
			"k2": "test6",
			"k1": []interface{}{"test5"},
			"k3": []interface{}{"test7", "test8"},
		},
	}
	expected = []string{"test1", "test2", "test3", "test4", "test5", "test6", "test7", "test8"}
	CheckFlattenAllStrValues(input, expected, t)

	input = MustReadYAMLStr(
		`
    test:
      key1: [test5]
      key2: test6
      key3: [test7, test8]
      a2:
        key1: [test1]
        key2: test2
        key3: [test3, test4]
      z:
        - key1: [test9]
          key2: test10
          key3: [test11, test12]
        - key1: [test13]
          key2: test14
          key3: [test15, test16]
  `)
	expected = []string{"test1", "test2", "test3", "test4", "test5", "test6", "test7", "test8", "test9", "test10", "test11", "test12", "test13", "test14", "test15", "test16"}
	CheckFlattenAllStrValues(input, expected, t)
}

func CheckFlattenAllStrValues(obj interface{}, expected []string, t *testing.T) {
	flattened, err := FlattenAllStrValues(obj)
	require.NoError(t, err)
	require.Equal(t, expected, flattened)
}
