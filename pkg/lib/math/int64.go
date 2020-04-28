/*
Copyright 2020 Cortex Labs, Inc.

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

package math

func MinInt64(val int64, vals ...int64) int64 {
	min := val
	for _, v := range vals {
		if v < min {
			min = v
		}
	}
	return min
}

func MaxInt64(val int64, vals ...int64) int64 {
	max := val
	for _, v := range vals {
		if v > max {
			max = v
		}
	}
	return max
}

func CheckDivisibleByInt64(num int64, divisor int64) bool {
	if num%divisor == 0 {
		return true
	}
	return false
}

func FindDivisibleNumbersOfInt64(num int64) []int64 {
	divisibleNumbers := []int64{}
	for divisor := int64(1); divisor <= num; divisor++ {
		if CheckDivisibleByInt64(num, divisor) {
			divisibleNumbers = append(divisibleNumbers, divisor)
		}
	}
	return divisibleNumbers
}
