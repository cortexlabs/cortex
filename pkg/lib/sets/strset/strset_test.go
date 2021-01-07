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

package strset_test

import (
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/stretchr/testify/require"
)

// Also tests Add
func TestNew(t *testing.T) {
	set := strset.New()
	require.Equal(t, 0, len(set))

	set = strset.New("a", "b", "a")
	require.Equal(t, 2, len(set))
	if _, ok := set["a"]; !ok {
		require.FailNow(t, "a not found in set")
	}
	if _, ok := set["b"]; !ok {
		require.FailNow(t, "b not found in set")
	}
}

func TestAdd(t *testing.T) {
	set := strset.New()
	set.Add("a")
	set.Add("b", "c")
	require.Equal(t, set, strset.New("a", "b", "c"))
}

func TestRemove(t *testing.T) {
	set := strset.New("a", "b")
	set.Remove("c")
	require.Equal(t, set, strset.New("a", "b"))

	set.Remove()
	require.Equal(t, set, strset.New("a", "b"))

	set.Remove("a")
	require.Equal(t, set, strset.New("b"))

	set.Add("a")
	set.Remove("a", "b")
	require.Equal(t, set, strset.New())
}

func TestPop(t *testing.T) {
	set := strset.New("a", "b")
	p := set.Pop()
	require.Contains(t, []string{"a", "b"}, p)
	require.Equal(t, 1, len(set))
	p = set.Pop()
	require.Contains(t, []string{"a", "b"}, p)
	require.Equal(t, 0, len(set))
	p = set.Pop()
	require.Equal(t, "", p)
	require.Equal(t, 0, len(set))
}

func TestPop2(t *testing.T) {
	set := strset.New("a", "b")
	p, ok := set.Pop2()
	require.Contains(t, []string{"a", "b"}, p)
	require.Equal(t, 1, len(set))
	require.True(t, ok)
	p, ok = set.Pop2()
	require.Contains(t, []string{"a", "b"}, p)
	require.Equal(t, 0, len(set))
	require.True(t, ok)
	p, ok = set.Pop2()
	require.Equal(t, "", p)
	require.Equal(t, 0, len(set))
	require.False(t, ok)
}

func TestHas(t *testing.T) {
	set := strset.New("a", "b", "c")
	require.True(t, set.Has("a"))
	require.True(t, set.Has("a", "b"))
	require.False(t, set.Has("z"))
	require.False(t, set.Has("a", "z"))
}

func TestHasAny(t *testing.T) {
	set := strset.New("a", "b", "c")
	require.True(t, set.HasAny("a"))
	require.True(t, set.HasAny("a", "b"))
	require.False(t, set.HasAny("z"))
	require.True(t, set.HasAny("a", "z"))
}

func TestClear(t *testing.T) {
	set := strset.New("a", "b", "c")
	require.Equal(t, 3, len(set))
	set.Clear()
	require.Equal(t, 0, len(set))
}

func TestIsEqual(t *testing.T) {
	set1 := strset.New("a", "b", "c")
	set2 := strset.New("a", "b", "c")
	set3 := strset.New("a", "b")
	set4 := strset.New("a", "b", "z")
	require.True(t, set1.IsEqual(set2))
	require.False(t, set1.IsEqual(set3))
	require.False(t, set1.IsEqual(set4))
}

func TestIsSubset(t *testing.T) {
	set1 := strset.New("a", "b", "c")
	set2 := strset.New("a", "b", "c")
	set3 := strset.New("a", "b")
	set4 := strset.New("a", "b", "z")
	set5 := strset.New("a", "b", "c", "d")
	require.True(t, set1.IsSubset(set2))
	require.True(t, set1.IsSubset(set3))
	require.False(t, set1.IsSubset(set4))
	require.False(t, set1.IsSubset(set5))
}

func TestIsSuperset(t *testing.T) {
	set1 := strset.New("a", "b", "c")
	set2 := strset.New("a", "b", "c")
	set3 := strset.New("a", "b")
	set4 := strset.New("a", "b", "z")
	set5 := strset.New("a", "b", "c", "d")
	require.True(t, set1.IsSuperset(set2))
	require.False(t, set1.IsSuperset(set3))
	require.False(t, set1.IsSuperset(set4))
	require.True(t, set1.IsSuperset(set5))
}

func TestCopy(t *testing.T) {
	set := strset.New()
	cset := set.Copy()
	require.Equal(t, 0, len(cset))

	set = strset.New("a", "b")
	cset = set.Copy()
	require.Equal(t, 2, len(cset))
	if _, ok := set["a"]; !ok {
		require.FailNow(t, "a not found in set")
	}
	if _, ok := set["b"]; !ok {
		require.FailNow(t, "b not found in set")
	}
}

func TestSlice(t *testing.T) {
	set := strset.New()
	require.Equal(t, set.Slice(), []string{})

	set.Add("a")
	require.Equal(t, set.Slice(), []string{"a"})

	set.Add("a", "b")
	require.ElementsMatch(t, set.Slice(), []string{"a", "b"})
}

func TestMerge(t *testing.T) {
	set := strset.New()
	emptySet := strset.New()
	set.Merge(emptySet)
	require.Equal(t, 0, len(set))

	set.Merge(strset.New("a"))
	require.Equal(t, 1, len(set))

	set.Merge(emptySet)
	require.Equal(t, 1, len(set))

	set.Add("a", "b", "c")
	set.Merge(strset.New("e", "e", "d"))
	require.Equal(t, set, strset.New("a", "b", "c", "e", "d"))

	set.Merge(strset.New("a", "e"))
	require.Equal(t, set, strset.New("a", "b", "c", "e", "d"))
	set.Merge(strset.New("a", "e", "i"), strset.New("o", "u"), strset.New("sometimes y"))
	require.Equal(t, set, strset.New("a", "b", "c", "e", "d", "i", "o", "u", "sometimes y"))
}

func TestSubtract(t *testing.T) {
	set := strset.New("a", "b", "c")
	set.Subtract(strset.New("z", "a", "b"))
	require.Equal(t, set, strset.New("c"))
	set.Subtract(strset.New("x"))
	require.Equal(t, set, strset.New("c"))
}

func TestShrink(t *testing.T) {
	set := strset.New("a", "b", "c", "d")
	set.Shrink(2)
	require.Len(t, set, 2)

	set = strset.New("g", "f", "e", "d", "c", "b", "a")
	set.ShrinkSorted(3)
	require.Len(t, set, 3)

	set = strset.New("a", "b")
	set.Shrink(2)
	require.Equal(t, set, strset.New("a", "b"))

	set = strset.New("a")
	set.Shrink(2)
	require.Equal(t, set, strset.New("a"))

	set = strset.New()
	set.Shrink(2)
	require.Len(t, set, 0)
}

func TestShrinkSorted(t *testing.T) {
	for i := 0; i < 10; i++ {
		set := strset.New("g", "f", "e", "d", "c", "b", "a")
		set.ShrinkSorted(2)
		require.Equal(t, set, strset.New("a", "b"))

		set = strset.New("g", "f", "e", "d", "c", "b", "a")
		set.ShrinkSorted(3)
		require.Equal(t, set, strset.New("a", "b", "c"))
	}

	set := strset.New("a", "b")
	set.ShrinkSorted(2)
	require.Equal(t, set, strset.New("a", "b"))

	set = strset.New("a")
	set.ShrinkSorted(2)
	require.Equal(t, set, strset.New("a"))

	set = strset.New()
	set.ShrinkSorted(2)
	require.Len(t, set, 0)
}

func TestUnion(t *testing.T) {
	require.Equal(t, len(strset.Union(strset.New(), strset.New())), 0)
	require.Equal(t, strset.Union(strset.New("a", "b"), strset.New()), strset.New("a", "b"))
	require.Equal(t, strset.Union(strset.New(), strset.New("a", "b")), strset.New("a", "b"))
	require.Equal(t, strset.Union(strset.New("a", "a"), strset.New("a", "b")), strset.New("a", "b"))
	require.Equal(t, strset.Union(strset.New("a"), strset.New("b")), strset.New("a", "b"))
}

func TestDifference(t *testing.T) {
	set1 := strset.New("a", "b", "c")
	set2 := strset.New("z", "a", "b")

	d := strset.Difference(set1, set2)
	require.Equal(t, d, strset.New("c"))
	d = strset.Difference(set2, set1)
	require.Equal(t, d, strset.New("z"))
	d = strset.Difference(set1, set1)
	require.Equal(t, d, strset.New())
}

func TestIntersection(t *testing.T) {
	set1 := strset.New("a", "b", "c")
	set2 := strset.New("a", "x", "y")
	set3 := strset.New("z", "b", "c")
	set4 := strset.New("z", "b", "w")

	d := strset.Intersection(set1, set2)
	require.Equal(t, d, strset.New("a"))
	d = strset.Intersection(set1, set3)
	require.Equal(t, d, strset.New("b", "c"))
	d = strset.Intersection(set2, set3)
	require.Equal(t, d, strset.New())
	d = strset.Intersection(set1, set3, set4)
	require.Equal(t, d, strset.New("b"))
}
