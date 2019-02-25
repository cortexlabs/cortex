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

package slices

import libmath "github.com/cortexlabs/cortex/pkg/lib/math"

func HasString(query string, list []string) bool {
	for _, elem := range list {
		if elem == query {
			return true
		}
	}
	return false
}

func HasAnyStrings(queries []string, list []string) bool {
	keys := make(map[string]bool)
	for _, elem := range queries {
		keys[elem] = true
	}
	for _, elem := range list {
		if _, ok := keys[elem]; ok {
			return true
		}
	}
	return false
}

func HasAllStrings(queries []string, list []string) bool {
	keys := make(map[string]bool)
	for _, elem := range list {
		keys[elem] = true
	}
	for _, elem := range queries {
		if _, ok := keys[elem]; !ok {
			return false
		}
	}
	return true
}

func CopyStrings(vals []string) []string {
	return append(vals[:0:0], vals...)
}

func UniqueStrings(strs []string) []string {
	keys := make(map[string]bool)
	out := []string{}
	for _, elem := range strs {
		if _, ok := keys[elem]; !ok {
			keys[elem] = true
			out = append(out, elem)
		}
	}
	return out
}

func RemoveEmpties(strs []string) []string {
	cleanStrs := []string{}
	for _, str := range strs {
		if str != "" {
			cleanStrs = append(cleanStrs, str)
		}
	}
	return cleanStrs
}

func RemoveEmptiesAndUnique(strs []string) []string {
	keys := make(map[string]bool)
	out := []string{}
	for _, elem := range strs {
		if elem != "" {
			if _, ok := keys[elem]; !ok {
				keys[elem] = true
				out = append(out, elem)
			}
		}
	}
	return out
}

func HasDuplicateStr(in []string) bool {
	keys := make(map[string]bool)
	for _, elem := range in {
		if _, ok := keys[elem]; ok {
			return true
		}
		keys[elem] = true
	}
	return false
}

func FindDuplicateStrs(in []string) []string {
	dups := []string{}
	keys := map[string]bool{}
	for _, elem := range in {
		if _, ok := keys[elem]; ok {
			dups = append(dups, elem)
		}
		keys[elem] = true
	}
	return dups
}

func SubtractStrSlice(slice1 []string, slice2 []string) []string {
	result := []string{}
	for _, elem := range slice1 {
		if !HasString(elem, slice2) {
			result = append(result, elem)
		}
	}
	return result
}

func StrSliceElementsMatch(strs1 []string, strs2 []string) bool {
	if len(strs1) == 0 && len(strs2) == 0 {
		return true
	}
	if len(strs1) != len(strs2) {
		return false
	}
	return StrSlicesEqual(SortStrsCopy(strs1), SortStrsCopy(strs2))
}

func StrSlicesEqual(strs1 []string, strs2 []string) bool {
	if len(strs1) == 0 && len(strs2) == 0 {
		return true
	}
	if len(strs1) != len(strs2) {
		return false
	}
	for i := range strs1 {
		if strs1[i] != strs2[i] {
			return false
		}
	}
	return true
}

func FilterStrs(strs []string, filterFn func(string) bool) []string {
	out := []string{}
	for _, elem := range strs {
		if filterFn(elem) {
			out = append(out, elem)
		}
	}
	return out
}

func MapStrs(strs []string, mapFn func(string) string) []string {
	out := make([]string, len(strs))
	for i, elem := range strs {
		out[i] = mapFn(elem)
	}
	return out
}

func MergeStrSlices(slices ...[]string) []string {
	if slices == nil || len(slices) == 0 {
		return nil
	}

	var totalLen int
	for _, s := range slices {
		totalLen += len(s)
	}

	result := make([]string, totalLen)
	var i int
	for _, s := range slices {
		i += copy(result[i:], s)
	}
	return result
}

func ZipStrsToMap(strs1 []string, strs2 []string) map[string]string {
	strMap := map[string]string{}
	length := libmath.MinInt(len(strs1), len(strs2))
	for i := 0; i < length; i++ {
		strMap[strs1[i]] = strs2[i]
	}
	return strMap
}

func StrSliceToSet(strs []string) map[string]bool {
	if strs == nil {
		return nil
	}
	set := make(map[string]bool)
	for _, str := range strs {
		set[str] = true
	}
	return set
}
