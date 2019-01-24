package util

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
	aCopy := CopyStrSlice(a)
	sort.Strings(aCopy)
	return aCopy
}

func SortIntsCopy(a []int) []int {
	aCopy := CopyIntSlice(a)
	sort.Ints(aCopy)
	return aCopy
}

func SortInt32sCopy(a []int32) []int32 {
	aCopy := CopyInt32Slice(a)
	SortInt32s(aCopy)
	return aCopy
}

func SortInt64sCopy(a []int64) []int64 {
	aCopy := CopyInt64Slice(a)
	SortInt64s(aCopy)
	return aCopy
}

func SortFloat32sCopy(a []float32) []float32 {
	aCopy := CopyFloat32Slice(a)
	SortFloat32s(aCopy)
	return aCopy
}

func SortFloat64sCopy(a []float64) []float64 {
	aCopy := CopyFloat64Slice(a)
	sort.Float64s(aCopy)
	return aCopy
}
