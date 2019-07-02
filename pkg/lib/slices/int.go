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

func HasInt(list []int, query int) bool {
	for _, elem := range list {
		if elem == query {
			return true
		}
	}
	return false
}

func CopyInts(vals []int) []int {
	return append(vals[:0:0], vals...)
}

func MinInt(vals ...int) int {
	min := vals[0]
	for _, val := range vals {
		if val < min {
			min = val
		}
	}
	return min
}

func MaxInt(vals ...int) int {
	max := vals[0]
	for _, val := range vals {
		if val > max {
			max = val
		}
	}
	return max
}

func AreNGreaterThanZero(minCount int, vals ...int) bool {
	count := 0
	for _, val := range vals {
		if val > 0 {
			count++
			if count >= minCount {
				return true
			}
		}
	}
	return false
}
