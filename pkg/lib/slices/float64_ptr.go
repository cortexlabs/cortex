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

// For adding integers stored in floats
func Float64PtrSumInt(floats ...*float64) int {
	sum := 0
	for _, num := range floats {
		if num != nil {
			sum += int(*num)
		}
	}
	return sum
}

func Float64PtrMin(floats ...*float64) *float64 {
	var min *float64

	for _, num := range floats {
		switch {
		case num != nil && min != nil && *num < *min:
			min = num
		case num != nil && min == nil:
			min = num
		}
	}
	return min
}

func Float64PtrMax(floats ...*float64) *float64 {
	var max *float64

	for _, num := range floats {
		switch {
		case num != nil && max != nil && *num > *max:
			max = num
		case num != nil && max == nil:
			max = num
		}
	}
	return max
}

func Float64PtrAvg(values []*float64, weights []*float64) (*float64, error) {
	if len(values) != len(weights) {
		return nil, ErrorLenValuesWeightsMismatch()
	}

	totalWeight := 0.0
	for i, valPtr := range values {
		if valPtr != nil && weights[i] != nil && *weights[i] > 0 {
			totalWeight += *weights[i]
		}
	}

	if totalWeight == 0.0 {
		return nil, nil
	}

	avg := 0.0
	for i, valPtr := range values {
		if valPtr != nil && weights[i] != nil && *weights[i] > 0 {
			avg += (*valPtr) * (*weights[i]) / float64(totalWeight)
		}
	}

	return &avg, nil
}
