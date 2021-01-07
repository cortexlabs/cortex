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

func IsDivisibleByInt(num int, divisor int) bool {
	return num%divisor == 0
}

func FactorsInt(num int) []int {
	divisibleNumbers := []int{}
	maxDivisor := int(math.Sqrt(float64(num)))
	incrementer := 1

	// Skip even numbers if num is odd
	if num%2 == 1 {
		incrementer = 2
	}

	for divisor := 1; divisor <= maxDivisor; divisor += incrementer {
		if num%divisor == 0 {
			divisibleNumbers = append(divisibleNumbers, divisor)
			complementaryDivisor := num / divisor
			if divisor != complementaryDivisor {
				divisibleNumbers = append(divisibleNumbers, complementaryDivisor)
			}
		}
	}

	sort.Ints(divisibleNumbers)
	return divisibleNumbers
}
