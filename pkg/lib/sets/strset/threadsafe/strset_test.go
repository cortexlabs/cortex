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

package threadsafe_test

import (
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset/threadsafe"
	"github.com/stretchr/testify/require"
)

// Also tests Add
func TestNew(t *testing.T) {
	set := threadsafe.New()
	require.Equal(t, 0, set.Len())

	set = threadsafe.New("a", "b", "a")
	require.Equal(t, 2, set.Len())
	if !set.Has("a") {
		require.FailNow(t, "a not found in set")
	}
	if !set.Has("b") {
		require.FailNow(t, "b not found in set")
	}
}

func TestAdd(t *testing.T) {
	set := threadsafe.New()
	set.Add("a")
	set.Add("b", "c")
	require.Equal(t, set, threadsafe.New("a", "b", "c"))
}

func TestRemove(t *testing.T) {
	set := threadsafe.New("a", "b")
	set.Remove("c")
	require.Equal(t, set, threadsafe.New("a", "b"))

	set.Remove()
	require.Equal(t, set, threadsafe.New("a", "b"))

	set.Remove("a")
	require.Equal(t, set, threadsafe.New("b"))

	set.Add("a")
	set.Remove("a", "b")
	require.Equal(t, set, threadsafe.New())
}

func TestPop(t *testing.T) {
	set := threadsafe.New("a", "b")
	p := set.Pop()
	require.Contains(t, []string{"a", "b"}, p)
	require.Equal(t, 1, set.Len())
	p = set.Pop()
	require.Contains(t, []string{"a", "b"}, p)
	require.Equal(t, 0, set.Len())
	p = set.Pop()
	require.Equal(t, "", p)
	require.Equal(t, 0, set.Len())
}

func TestPop2(t *testing.T) {
	set := threadsafe.New("a", "b")
	p, ok := set.Pop2()
	require.Contains(t, []string{"a", "b"}, p)
	require.Equal(t, 1, set.Len())
	require.True(t, ok)
	p, ok = set.Pop2()
	require.Contains(t, []string{"a", "b"}, p)
	require.Equal(t, 0, set.Len())
	require.True(t, ok)
	p, ok = set.Pop2()
	require.Equal(t, "", p)
	require.Equal(t, 0, set.Len())
	require.False(t, ok)
}

func TestHas(t *testing.T) {
	set := threadsafe.New("a", "b", "c")
	require.True(t, set.Has("a"))
	require.True(t, set.Has("a", "b"))
	require.False(t, set.Has("z"))
	require.False(t, set.Has("a", "z"))
}

func TestHasAny(t *testing.T) {
	set := threadsafe.New("a", "b", "c")
	require.True(t, set.HasAny("a"))
	require.True(t, set.HasAny("a", "b"))
	require.False(t, set.HasAny("z"))
	require.True(t, set.HasAny("a", "z"))
}

func TestClear(t *testing.T) {
	set := threadsafe.New("a", "b", "c")
	require.Equal(t, 3, set.Len())
	set.Clear()
	require.Equal(t, 0, set.Len())
}

func TestIsEqual(t *testing.T) {
	set1 := threadsafe.New("a", "b", "c")
	set2 := strset.New("a", "b", "c")
	set3 := strset.New("a", "b")
	set4 := strset.New("a", "b", "z")
	require.True(t, set1.IsEqual(set2))
	require.False(t, set1.IsEqual(set3))
	require.False(t, set1.IsEqual(set4))
}

func IsEqualThreadsafe(t *testing.T) {
	set1 := threadsafe.New("a", "b", "c")
	set2 := threadsafe.New("a", "b", "c")
	set3 := threadsafe.New("a", "b")
	set4 := threadsafe.New("a", "b", "z")
	require.True(t, set1.IsEqualThreadsafe(set2))
	require.False(t, set1.IsEqualThreadsafe(set3))
	require.False(t, set1.IsEqualThreadsafe(set4))
}

func TestIsSubset(t *testing.T) {
	set1 := threadsafe.New("a", "b", "c")
	set2 := strset.New("a", "b", "c")
	set3 := strset.New("a", "b")
	set4 := strset.New("a", "b", "z")
	set5 := strset.New("a", "b", "c", "d")
	require.True(t, set1.IsSubset(set2))
	require.True(t, set1.IsSubset(set3))
	require.False(t, set1.IsSubset(set4))
	require.False(t, set1.IsSubset(set5))
}

func IsSubsetThreadsafe(t *testing.T) {
	set1 := threadsafe.New("a", "b", "c")
	set2 := threadsafe.New("a", "b", "c")
	set3 := threadsafe.New("a", "b")
	set4 := threadsafe.New("a", "b", "z")
	set5 := threadsafe.New("a", "b", "c", "d")
	require.True(t, set1.IsSubsetThreadsafe(set2))
	require.True(t, set1.IsSubsetThreadsafe(set3))
	require.False(t, set1.IsSubsetThreadsafe(set4))
	require.False(t, set1.IsSubsetThreadsafe(set5))
}

func TestIsSuperset(t *testing.T) {
	set1 := threadsafe.New("a", "b", "c")
	set2 := strset.New("a", "b", "c")
	set3 := strset.New("a", "b")
	set4 := strset.New("a", "b", "z")
	set5 := strset.New("a", "b", "c", "d")
	require.True(t, set1.IsSuperset(set2))
	require.False(t, set1.IsSuperset(set3))
	require.False(t, set1.IsSuperset(set4))
	require.True(t, set1.IsSuperset(set5))
}

func IsSupersetThreadsafe(t *testing.T) {
	set1 := threadsafe.New("a", "b", "c")
	set2 := threadsafe.New("a", "b", "c")
	set3 := threadsafe.New("a", "b")
	set4 := threadsafe.New("a", "b", "z")
	set5 := threadsafe.New("a", "b", "c", "d")
	require.True(t, set1.IsSupersetThreadsafe(set2))
	require.False(t, set1.IsSupersetThreadsafe(set3))
	require.False(t, set1.IsSupersetThreadsafe(set4))
	require.True(t, set1.IsSupersetThreadsafe(set5))
}

func TestCopy(t *testing.T) {
	set := threadsafe.New()
	cset := set.Copy()
	require.Equal(t, 0, len(cset))

	set = threadsafe.New("a", "b")
	cset = set.Copy()
	require.Equal(t, 2, len(cset))
	if !set.Has("a") {
		require.FailNow(t, "a not found in set")
	}
	if !set.Has("b") {
		require.FailNow(t, "b not found in set")
	}
}

func TestCopyToThreadsafe(t *testing.T) {
	set := threadsafe.New()
	cset := set.CopyToThreadsafe()
	require.Equal(t, 0, cset.Len())

	set = threadsafe.New("a", "b")
	cset = set.CopyToThreadsafe()
	require.Equal(t, 2, cset.Len())
	if !set.Has("a") {
		require.FailNow(t, "a not found in set")
	}
	if !set.Has("b") {
		require.FailNow(t, "b not found in set")
	}
}

func TestSlice(t *testing.T) {
	set := threadsafe.New()
	require.Equal(t, set.Slice(), []string{})

	set.Add("a")
	require.Equal(t, set.Slice(), []string{"a"})

	set.Add("a", "b")
	require.ElementsMatch(t, set.Slice(), []string{"a", "b"})
}

func TestMerge(t *testing.T) {
	set := threadsafe.New()
	emptySet := strset.New()
	set.Merge(emptySet)
	require.Equal(t, 0, set.Len())

	set.Merge(strset.New("a"))
	require.Equal(t, 1, set.Len())

	set.Merge(emptySet)
	require.Equal(t, 1, set.Len())

	set.Add("a", "b", "c")
	set.Merge(strset.New("e", "e", "d"))
	require.Equal(t, set, threadsafe.New("a", "b", "c", "e", "d"))

	set.Merge(strset.New("a", "e"))
	require.Equal(t, set, threadsafe.New("a", "b", "c", "e", "d"))
	set.Merge(strset.New("a", "e", "i"), strset.New("o", "u"), strset.New("sometimes y"))
	require.Equal(t, set, threadsafe.New("a", "b", "c", "e", "d", "i", "o", "u", "sometimes y"))
}

func TestMergeThreadsafe(t *testing.T) {
	set := threadsafe.New()
	emptySet := threadsafe.New()
	set.MergeThreadsafe(emptySet)
	require.Equal(t, 0, set.Len())

	set.MergeThreadsafe(threadsafe.New("a"))
	require.Equal(t, 1, set.Len())

	set.MergeThreadsafe(emptySet)
	require.Equal(t, 1, set.Len())

	set.Add("a", "b", "c")
	set.MergeThreadsafe(threadsafe.New("e", "e", "d"))
	require.Equal(t, set, threadsafe.New("a", "b", "c", "e", "d"))

	set.MergeThreadsafe(threadsafe.New("a", "e"))
	require.Equal(t, set, threadsafe.New("a", "b", "c", "e", "d"))
	set.MergeThreadsafe(threadsafe.New("a", "e", "i"), threadsafe.New("o", "u"), threadsafe.New("sometimes y"))
	require.Equal(t, set, threadsafe.New("a", "b", "c", "e", "d", "i", "o", "u", "sometimes y"))
}

func TestSubtract(t *testing.T) {
	set := threadsafe.New("a", "b", "c")
	set.Subtract(strset.New("z", "a", "b"))
	require.Equal(t, set, threadsafe.New("c"))
	set.Subtract(strset.New("x"))
	require.Equal(t, set, threadsafe.New("c"))
}

func SubtractThreadsafe(t *testing.T) {
	set := threadsafe.New("a", "b", "c")
	set.SubtractThreadsafe(threadsafe.New("z", "a", "b"))
	require.Equal(t, set, threadsafe.New("c"))
	set.SubtractThreadsafe(threadsafe.New("x"))
	require.Equal(t, set, threadsafe.New("c"))
}

func TestShrink(t *testing.T) {
	set := threadsafe.New("a", "b", "c", "d")
	set.Shrink(2)
	require.Equal(t, set.Len(), 2)

	set = threadsafe.New("g", "f", "e", "d", "c", "b", "a")
	set.ShrinkSorted(3)
	require.Equal(t, set.Len(), 3)

	set = threadsafe.New("a", "b")
	set.Shrink(2)
	require.Equal(t, set, threadsafe.New("a", "b"))

	set = threadsafe.New("a")
	set.Shrink(2)
	require.Equal(t, set, threadsafe.New("a"))

	set = threadsafe.New()
	set.Shrink(2)
	require.Equal(t, set.Len(), 0)
}

func TestShrinkSorted(t *testing.T) {
	for i := 0; i < 10; i++ {
		set := threadsafe.New("g", "f", "e", "d", "c", "b", "a")
		set.ShrinkSorted(2)
		require.Equal(t, set, threadsafe.New("a", "b"))

		set = threadsafe.New("g", "f", "e", "d", "c", "b", "a")
		set.ShrinkSorted(3)
		require.Equal(t, set, threadsafe.New("a", "b", "c"))
	}

	set := threadsafe.New("a", "b")
	set.ShrinkSorted(2)
	require.Equal(t, set, threadsafe.New("a", "b"))

	set = threadsafe.New("a")
	set.ShrinkSorted(2)
	require.Equal(t, set, threadsafe.New("a"))

	set = threadsafe.New()
	set.ShrinkSorted(2)
	require.Equal(t, set.Len(), 0)
}

func TestUnion(t *testing.T) {
	require.Equal(t, threadsafe.Union(threadsafe.New(), strset.New()).Len(), 0)
	require.Equal(t, threadsafe.Union(threadsafe.New("a", "b"), strset.New()), threadsafe.New("a", "b"))
	require.Equal(t, threadsafe.Union(threadsafe.New(), strset.New("a", "b")), threadsafe.New("a", "b"))
	require.Equal(t, threadsafe.Union(threadsafe.New("a", "a"), strset.New("a", "b")), threadsafe.New("a", "b"))
	require.Equal(t, threadsafe.Union(threadsafe.New("a"), strset.New("b")), threadsafe.New("a", "b"))
}

func TestUnionThreadsafe(t *testing.T) {
	require.Equal(t, threadsafe.UnionThreadsafe(threadsafe.New(), threadsafe.New()).Len(), 0)
	require.Equal(t, threadsafe.UnionThreadsafe(threadsafe.New("a", "b"), threadsafe.New()), threadsafe.New("a", "b"))
	require.Equal(t, threadsafe.UnionThreadsafe(threadsafe.New(), threadsafe.New("a", "b")), threadsafe.New("a", "b"))
	require.Equal(t, threadsafe.UnionThreadsafe(threadsafe.New("a", "a"), threadsafe.New("a", "b")), threadsafe.New("a", "b"))
	require.Equal(t, threadsafe.UnionThreadsafe(threadsafe.New("a"), threadsafe.New("b")), threadsafe.New("a", "b"))
}

func TestDifference(t *testing.T) {
	set1 := threadsafe.New("a", "b", "c")
	set1Strset := strset.New("a", "b", "c")
	set2 := threadsafe.New("z", "a", "b")
	set2Strset := strset.New("z", "a", "b")

	d := threadsafe.Difference(set1, set2Strset)
	require.Equal(t, d, threadsafe.New("c"))
	d = threadsafe.Difference(set2, set1Strset)
	require.Equal(t, d, threadsafe.New("z"))
	d = threadsafe.Difference(set1, set1Strset)
	require.Equal(t, d, threadsafe.New())
}

func TestDifferenceThreadsafe(t *testing.T) {
	set1 := threadsafe.New("a", "b", "c")
	set2 := threadsafe.New("z", "a", "b")

	d := threadsafe.DifferenceThreadsafe(set1, set2)
	require.Equal(t, d, threadsafe.New("c"))
	d = threadsafe.DifferenceThreadsafe(set2, set1)
	require.Equal(t, d, threadsafe.New("z"))
	d = threadsafe.DifferenceThreadsafe(set1, set1)
	require.Equal(t, d, threadsafe.New())
}

func TestIntersection(t *testing.T) {
	set1 := threadsafe.New("a", "b", "c")
	set2 := threadsafe.New("a", "x", "y")
	set2Strset := strset.New("a", "x", "y")
	set3Strset := strset.New("z", "b", "c")
	set4Strset := strset.New("z", "b", "w")

	d := threadsafe.Intersection(set1, set2Strset)
	require.Equal(t, d, threadsafe.New("a"))
	d = threadsafe.Intersection(set1, set3Strset)
	require.Equal(t, d, threadsafe.New("b", "c"))
	d = threadsafe.Intersection(set2, set3Strset)
	require.Equal(t, d, threadsafe.New())
	d = threadsafe.Intersection(set1, set3Strset, set4Strset)
	require.Equal(t, d, threadsafe.New("b"))
}

func TestIntersectionThreadsafe(t *testing.T) {
	set1 := threadsafe.New("a", "b", "c")
	set2 := threadsafe.New("a", "x", "y")
	set3 := threadsafe.New("z", "b", "c")
	set4 := threadsafe.New("z", "b", "w")

	d := threadsafe.IntersectionThreadsafe(set1, set2)
	require.Equal(t, d, threadsafe.New("a"))
	d = threadsafe.IntersectionThreadsafe(set1, set3)
	require.Equal(t, d, threadsafe.New("b", "c"))
	d = threadsafe.IntersectionThreadsafe(set2, set3)
	require.Equal(t, d, threadsafe.New())
	d = threadsafe.IntersectionThreadsafe(set1, set3, set4)
	require.Equal(t, d, threadsafe.New("b"))
}
