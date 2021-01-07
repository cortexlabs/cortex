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

import (
	"math"
	"sort"
)

// Inspiration: https://golang.org/src/sort/sort.go

type Int32Slice []int32

func (p Int32Slice) Len() int           { return len(p) }
func (p Int32Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int32Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func SortInt32s(a []int32)              { sort.Sort(Int32Slice(a)) }

type Int64Slice []int64

func (p Int64Slice) Len() int           { return len(p) }
func (p Int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func SortInt64s(a []int64)              { sort.Sort(Int64Slice(a)) }

type Float32Slice []float32

func (p Float32Slice) Len() int { return len(p) }
func (p Float32Slice) Less(i, j int) bool {
	return p[i] < p[j] || math.IsNaN(float64(p[i])) && !math.IsNaN(float64(p[j]))
}
func (p Float32Slice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func SortFloat32s(a []float32)       { sort.Sort(Float32Slice(a)) }

// Sort with copying

func SortStrsCopy(a []string) []string {
	aCopy := CopyStrings(a)
	sort.Strings(aCopy)
	return aCopy
}

func SortIntsCopy(a []int) []int {
	aCopy := CopyInts(a)
	sort.Ints(aCopy)
	return aCopy
}

func SortInt32sCopy(a []int32) []int32 {
	aCopy := CopyInt32s(a)
	SortInt32s(aCopy)
	return aCopy
}

func SortInt64sCopy(a []int64) []int64 {
	aCopy := CopyInt64s(a)
	SortInt64s(aCopy)
	return aCopy
}

func SortFloat32sCopy(a []float32) []float32 {
	aCopy := CopyFloat32s(a)
	SortFloat32s(aCopy)
	return aCopy
}

func SortFloat64sCopy(a []float64) []float64 {
	aCopy := CopyFloat64s(a)
	sort.Float64s(aCopy)
	return aCopy
}
