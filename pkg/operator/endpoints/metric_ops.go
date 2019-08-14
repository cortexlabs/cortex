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

package endpoints

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

func SumInt(floats ...*float64) *int {
	if len(floats) == 0 {
		return nil
	}

	sum := 0
	for _, num := range floats {
		sum += int(*num)
	}
	return &sum
}

func SumFloat(floats ...*float64) *float64 {
	if len(floats) == 0 {
		return nil
	}

	sum := 0.0
	for _, num := range floats {
		sum += *num
	}
	return &sum
}

func Min(floats ...*float64) *float64 {
	if len(floats) == 0 {
		return nil
	}

	min := *floats[0]
	for _, num := range floats {
		if min > *num {
			min = *num
		}
	}
	return &min
}

func Max(floats ...*float64) *float64 {
	if len(floats) == 0 {
		return nil
	}

	max := *floats[0]
	for _, num := range floats {
		if max < *num {
			max = *num
		}
	}
	return &max
}

func Avg(values []*float64, counts []*float64) (*float64, error) {
	if len(values) == 0 || len(counts) == 0 {
		return nil, nil
	}

	if len(values) != len(counts) {
		return nil, errors.New("length of values is not equal to length of weights")
	}

	total := float64(*SumInt(counts...))
	avg := 0.0
	for idx, valPtr := range values {
		weight := *counts[idx]

		if weight <= 0 {
			return nil, errors.New(fmt.Sprintf("weight %g must be greater", weight))
		}

		value := *valPtr
		avg += value * weight / total
	}

	return &avg, nil
}
