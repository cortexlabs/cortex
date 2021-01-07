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

package pointer

import (
	"reflect"
	"time"
)

func Int(val int) *int {
	return &val
}

func Int8(val int8) *int8 {
	return &val
}

func Int16(val int16) *int16 {
	return &val
}

func Int32(val int32) *int32 {
	return &val
}

func Int64(val int64) *int64 {
	return &val
}

func Float64(val float64) *float64 {
	return &val
}

func Float32(val float32) *float32 {
	return &val
}

func String(val string) *string {
	return &val
}

func Bool(val bool) *bool {
	return &val
}

func Time(val time.Time) *time.Time {
	return &val
}

func Duration(val time.Duration) *time.Duration {
	return &val
}

// IndirectSafe dereferences if obj is a pointer, otherwise no-op
func IndirectSafe(obj interface{}) interface{} {
	if obj == nil {
		return nil
	}
	return reflect.Indirect(reflect.ValueOf(obj)).Interface()
}
