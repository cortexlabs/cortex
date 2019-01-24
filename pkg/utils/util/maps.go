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

package util

import (
	"reflect"
	"sort"
)

// interface{}

func InterfaceMapKeys(myMap map[string]interface{}) []string {
	keys := make([]string, len(myMap))
	i := 0
	for key := range myMap {
		keys[i] = key
		i++
	}
	return keys
}

func InterfaceMapSortedKeys(myMap map[string]interface{}) []string {
	keys := InterfaceMapKeys(myMap)
	sort.Strings(keys)
	return keys
}

func InterfaceMapKeysUnsafe(myMap interface{}) []string {
	keyValues := reflect.ValueOf(myMap).MapKeys()
	keys := make([]string, len(keyValues))
	for i := range keyValues {
		keys[i] = keyValues[i].String()
	}
	return keys
}

func InterfaceMapsKeysMatch(map1 map[string]interface{}, map2 map[string]interface{}) bool {
	if len(map1) != len(map2) {
		return false
	}
	for key, _ := range map1 {
		if _, ok := map2[key]; !ok {
			return false
		}
	}
	return true
}

// string

func StrMapKeys(myMap map[string]string) []string {
	keys := make([]string, len(myMap))
	i := 0
	for key := range myMap {
		keys[i] = key
		i++
	}
	return keys
}

func StrMapValues(myMap map[string]string) []string {
	values := make([]string, len(myMap))
	i := 0
	for _, value := range myMap {
		values[i] = value
		i++
	}
	return values
}

func MergeStrMaps(maps ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}
