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

package slices

import (
	"strconv"

	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

func HasString(list []string, query string) bool {
	for _, elem := range list {
		if elem == query {
			return true
		}
	}
	return false
}

func HasAnyStrings(queries []string, list []string) bool {
	keys := strset.New()
	for _, elem := range queries {
		keys.Add(elem)
	}
	for _, elem := range list {
		if keys.Has(elem) {
			return true
		}
	}
	return false
}

func HasAllStrings(queries []string, list []string) bool {
	keys := strset.New()
	for _, elem := range list {
		keys.Add(elem)
	}
	for _, elem := range queries {
		if !keys.Has(elem) {
			return false
		}
	}
	return true
}

func CopyStrings(vals []string) []string {
	return append(vals[:0:0], vals...)
}

func UniqueStrings(strs []string) []string {
	keys := strset.New()
	out := []string{}
	for _, elem := range strs {
		if !keys.Has(elem) {
			keys.Add(elem)
			out = append(out, elem)
		}
	}
	return out
}

func RemoveEmpties(strs []string) []string {
	var cleanStrs []string
	for _, str := range strs {
		if str != "" {
			cleanStrs = append(cleanStrs, str)
		}
	}
	return cleanStrs
}

func RemoveEmptiesAndUnique(strs []string) []string {
	keys := strset.New()
	out := []string{}
	for _, elem := range strs {
		if elem != "" {
			if !keys.Has(elem) {
				keys.Add(elem)
				out = append(out, elem)
			}
		}
	}
	return out
}

func HasDuplicateStr(in []string) bool {
	keys := strset.New()
	for _, elem := range in {
		if keys.Has(elem) {
			return true
		}
		keys.Add(elem)
	}
	return false
}

func FindDuplicateStrs(in []string) []string {
	dups := []string{}
	keys := strset.New()
	for _, elem := range in {
		if keys.Has(elem) {
			dups = append(dups, elem)
		}
		keys.Add(elem)
	}
	return dups
}

func SubtractStrSlice(slice1 []string, slice2 []string) []string {
	result := []string{}
	for _, elem := range slice1 {
		if !HasString(slice2, elem) {
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

func StringToInt(vals []string) ([]int, error) {
	intSlice := []int{}
	for _, elem := range vals {
		i, err := strconv.Atoi(elem)
		if err != nil {
			return nil, err
		}
		intSlice = append(intSlice, i)
	}
	return intSlice, nil
}

func StringToInt32(vals []string) ([]int32, error) {
	intSlice := []int32{}
	for _, elem := range vals {
		i, err := strconv.ParseInt(elem, 10, 32)
		if err != nil {
			return nil, err
		}
		intSlice = append(intSlice, int32(i))
	}
	return intSlice, nil
}

func StringToInt64(vals []string) ([]int64, error) {
	intSlice := []int64{}
	for _, elem := range vals {
		i, err := strconv.ParseInt(elem, 10, 64)
		if err != nil {
			return nil, err
		}
		intSlice = append(intSlice, i)
	}
	return intSlice, nil
}
