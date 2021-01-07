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
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

func HasInt64(list []int64, query int64) bool {
	for _, elem := range list {
		if elem == query {
			return true
		}
	}
	return false
}

func CopyInt64s(vals []int64) []int64 {
	return append(vals[:0:0], vals...)
}

func UniqueInt64(vals []int64) []int64 {
	keys := make(map[int64]bool)
	list := []int64{}
	for _, entry := range vals {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func Int64ToString(vals []int64) []string {
	stringSlice := []string{}
	for _, elem := range vals {
		stringSlice = append(stringSlice, s.Int64(elem))
	}
	return stringSlice
}
