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

package math

import (
	"math"
	"sort"
)

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

func IsDivisibleByInt64(num int64, divisor int64) bool {
	return num%divisor == 0
}

func FactorsInt64(num int64) []int64 {
	divisibleNumbers := []int64{}
	maxDivisor := int64(math.Sqrt(float64(num)))
	incrementer := int64(1)

	// Skip even numbers if num is odd
	if num%2 == 1 {
		incrementer = int64(2)
	}

	for divisor := int64(1); divisor <= maxDivisor; divisor += incrementer {
		if num%divisor == 0 {
			divisibleNumbers = append(divisibleNumbers, divisor)
			complementaryDivisor := num / divisor
			if divisor != complementaryDivisor {
				divisibleNumbers = append(divisibleNumbers, complementaryDivisor)
			}
		}
	}

	sort.Slice(divisibleNumbers, func(i, j int) bool { return divisibleNumbers[i] < divisibleNumbers[j] })
	return divisibleNumbers
}
