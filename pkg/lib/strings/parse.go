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

package strings

import (
	"strconv"
)

func ParseBool(valStr string) (bool, bool) {
	casted, err := strconv.ParseBool(valStr)
	if err != nil {
		return false, false
	}
	return casted, true
}

func ParseFloat32(valStr string) (float32, bool) {
	casted, err := strconv.ParseFloat(valStr, 32)
	if err != nil {
		return 0, false
	}
	return float32(casted), true
}

func ParseFloat64(valStr string) (float64, bool) {
	casted, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, false
	}
	return casted, true
}

func ParseInt(valStr string) (int, bool) {
	casted, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, false
	}
	return casted, true
}

func ParseInt64(valStr string) (int64, bool) {
	casted, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return 0, false
	}
	return casted, true
}

func ParseInt32(valStr string) (int32, bool) {
	casted, err := strconv.ParseInt(valStr, 10, 32)
	if err != nil {
		return 0, false
	}
	return int32(casted), true
}

func ParseInt16(valStr string) (int16, bool) {
	casted, err := strconv.ParseInt(valStr, 10, 16)
	if err != nil {
		return 0, false
	}
	return int16(casted), true
}

func ParseInt8(valStr string) (int8, bool) {
	casted, err := strconv.ParseInt(valStr, 10, 8)
	if err != nil {
		return 0, false
	}
	return int8(casted), true
}
