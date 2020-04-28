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

func MinInt(val int, vals ...int) int {
	min := val
	for _, v := range vals {
		if v < min {
			min = v
		}
	}
	return min
}

func MaxInt(val int, vals ...int) int {
	max := val
	for _, v := range vals {
		if v > max {
			max = v
		}
	}
	return max
}

func CheckDivisibleByInt(num int, divisor int) bool {
	if num%divisor == 0 {
		return true
	}
	return false
}

func FindDivisibleNumbersOfInt(num int) []int {
	divisibleNumbers := []int{}
	for divisor := 1; divisor <= num; divisor++ {
		if CheckDivisibleByInt(num, divisor) {
			divisibleNumbers = append(divisibleNumbers, divisor)
		}
	}
	return divisibleNumbers
}
