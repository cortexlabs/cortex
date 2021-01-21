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

package maps

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

func StrMapsEqual(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}

	if len(m1) == 0 && len(m2) == 0 {
		return true
	}

	if len(m1) == 0 || len(m2) == 0 {
		return false
	}

	for k, v1 := range m1 {
		if v2, ok := m2[k]; !ok || v2 != v1 {
			return false
		}
	}

	return true
}
