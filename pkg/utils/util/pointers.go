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
	"reflect"
)

func IntPtr(val int) *int {
	return &val
}

func Int8Ptr(val int8) *int8 {
	return &val
}

func Int16Ptr(val int16) *int16 {
	return &val
}

func Int32Ptr(val int32) *int32 {
	return &val
}

func Int64Ptr(val int64) *int64 {
	return &val
}

func Float64Ptr(val float64) *float64 {
	return &val
}

func Float32Ptr(val float32) *float32 {
	return &val
}

func StrPtr(val string) *string {
	return &val
}

func BoolPtr(val bool) *bool {
	return &val
}

// Dereference if obj is a pointer, otherwise no-op
func IndirectSafe(obj interface{}) interface{} {
	if obj == nil {
		return nil
	}
	return reflect.Indirect(reflect.ValueOf(obj)).Interface()
}
