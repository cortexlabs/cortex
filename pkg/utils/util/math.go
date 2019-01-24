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

import (
	"math"
	"strconv"
	"strings"
)

func Round(val float64, decimalPlaces int, pad bool) string {
	rounded := math.Round(val*math.Pow10(decimalPlaces)) / math.Pow10(decimalPlaces)
	str := strconv.FormatFloat(rounded, 'f', -1, 64)
	if !pad || decimalPlaces == 0 {
		return str
	}
	split := strings.Split(str, ".")
	intVal := split[0]
	decVal := ""
	if len(split) > 1 {
		decVal = split[1]
	}
	numZeros := decimalPlaces - len(decVal)
	return intVal + "." + decVal + strings.Repeat("0", numZeros)
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
