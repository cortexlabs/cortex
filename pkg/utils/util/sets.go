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

import "sort"

// string

func MergeStrSets(sets ...map[string]bool) map[string]bool {
	merged := make(map[string]bool)
	for _, set := range sets {
		for str, v := range set {
			if v == true {
				merged[str] = true
			}
		}
	}
	return merged
}

func SubtractStrSets(set1 map[string]bool, set2 map[string]bool) map[string]bool {
	subtraction := make(map[string]bool)
	for str := range set1 {
		if set2[str] == false {
			subtraction[str] = true
		}
	}
	return subtraction
}

func IntersectStrSets(set1 map[string]bool, set2 map[string]bool) map[string]bool {
	intersection := make(map[string]bool)
	for str := range set1 {
		if set2[str] == true {
			intersection[str] = true
		}
	}
	return intersection
}

func StrSetToSlice(set map[string]bool) []string {
	var strs []string
	for str, isPresent := range set {
		if isPresent {
			strs = append(strs, str)
		}
	}
	sort.Strings(strs)
	return strs
}

func CopyStrSet(set map[string]bool) map[string]bool {
	copiedSet := make(map[string]bool, len(set))
	for str := range set {
		copiedSet[str] = true
	}
	return copiedSet
}

func DoStrSetsOverlap(set1 map[string]bool, set2 map[string]bool) bool {
	if set1 == nil || set2 == nil {
		return false
	}
	for str := range set1 {
		if set2[str] == true {
			return true
		}
	}
	return false
}
