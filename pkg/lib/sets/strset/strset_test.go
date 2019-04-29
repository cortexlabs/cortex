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

package strset

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Also tests Add
func TestNew(t *testing.T) {
	set := New()
	require.Equal(t, 0, len(set))

	set = New("a", "b", "a")
	require.Equal(t, 2, len(set))
	if _, ok := set["a"]; !ok {
		require.FailNow(t, "a not found in set")
	}
	if _, ok := set["b"]; !ok {
		require.FailNow(t, "b not found in set")
	}
}

func TestAdd(t *testing.T) {
	set := New()
	set.Add("a")
	set.Add("b", "c")
	require.Equal(t, set, New("a", "b", "c"))
}

func TestRemove(t *testing.T) {
	set := New("a", "b")
	set.Remove("c")
	require.Equal(t, set, New("a", "b"))

	set.Remove()
	require.Equal(t, set, New("a", "b"))

	set.Remove("a")
	require.Equal(t, set, New("b"))

	set.Add("a")
	set.Remove("a", "b")
	require.Equal(t, set, New())
}

func TestPop(t *testing.T) {
	set := New("a", "b")
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
	set := New("a", "b")
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
	set := New("a", "b", "c")
	require.True(t, set.Has("a"))
	require.True(t, set.Has("a", "b"))
	require.False(t, set.Has("z"))
	require.False(t, set.Has("a", "z"))
}

func TestHasAny(t *testing.T) {
	set := New("a", "b", "c")
	require.True(t, set.HasAny("a"))
	require.True(t, set.HasAny("a", "b"))
	require.False(t, set.HasAny("z"))
	require.True(t, set.HasAny("a", "z"))
}

func TestClear(t *testing.T) {
	set := New("a", "b", "c")
	require.Equal(t, 3, len(set))
	set.Clear()
	require.Equal(t, 0, len(set))
}

func TestIsEqual(t *testing.T) {
	set1 := New("a", "b", "c")
	set2 := New("a", "b", "c")
	set3 := New("a", "b")
	set4 := New("a", "b", "z")
	require.True(t, set1.IsEqual(set2))
	require.False(t, set1.IsEqual(set3))
	require.False(t, set1.IsEqual(set4))
}

func TestIsSubset(t *testing.T) {
	set1 := New("a", "b", "c")
	set2 := New("a", "b", "c")
	set3 := New("a", "b")
	set4 := New("a", "b", "z")
	set5 := New("a", "b", "c", "d")
	require.True(t, set1.IsSubset(set2))
	require.True(t, set1.IsSubset(set3))
	require.False(t, set1.IsSubset(set4))
	require.False(t, set1.IsSubset(set5))
}

func TestIsSuperset(t *testing.T) {
	set1 := New("a", "b", "c")
	set2 := New("a", "b", "c")
	set3 := New("a", "b")
	set4 := New("a", "b", "z")
	set5 := New("a", "b", "c", "d")
	require.True(t, set1.IsSuperset(set2))
	require.False(t, set1.IsSuperset(set3))
	require.False(t, set1.IsSuperset(set4))
	require.True(t, set1.IsSuperset(set5))
}

func TestCopy(t *testing.T) {
	set := New()
	cset := set.Copy()
	require.Equal(t, 0, len(cset))

	set = New("a", "b")
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
	set := New()
	require.Equal(t, set.Slice(), []string{})

	set.Add("a")
	require.Equal(t, set.Slice(), []string{"a"})

	set.Add("a", "b")
	require.ElementsMatch(t, set.Slice(), []string{"a", "b"})
}

func TestMerge(t *testing.T) {
	set := New()
	emptySet := New()
	set.Merge(emptySet)
	require.Equal(t, 0, len(set))

	set.Merge(New("a"))
	require.Equal(t, 1, len(set))

	set.Merge(emptySet)
	require.Equal(t, 1, len(set))

	set.Add("a", "b", "c")
	set.Merge(New("e", "e", "d"))
	require.Equal(t, set, New("a", "b", "c", "e", "d"))

	set.Merge(New("a", "e"))
	require.Equal(t, set, New("a", "b", "c", "e", "d"))
	set.Merge(New("a", "e", "i"), New("o", "u"), New("sometimes y"))
	require.Equal(t, set, New("a", "b", "c", "e", "d", "i", "o", "u", "sometimes y"))
}

func TestSubtract(t *testing.T) {
	set := New("a", "b", "c")
	set.Subtract(New("z", "a", "b"))
	require.Equal(t, set, New("c"))
	set.Subtract(New("x"))
	require.Equal(t, set, New("c"))
}

func TestUnion(t *testing.T) {
	require.Equal(t, len(Union(New(), New())), 0)
	require.Equal(t, Union(New("a", "b"), New()), New("a", "b"))
	require.Equal(t, Union(New(), New("a", "b")), New("a", "b"))
	require.Equal(t, Union(New("a", "a"), New("a", "b")), New("a", "b"))
	require.Equal(t, Union(New("a"), New("b")), New("a", "b"))
}

func TestDifference(t *testing.T) {
	set1 := New("a", "b", "c")
	set2 := New("z", "a", "b")

	d := Difference(set1, set2)
	require.Equal(t, d, New("c"))
	d = Difference(set2, set1)
	require.Equal(t, d, New("z"))
	d = Difference(set1, set1)
	require.Equal(t, d, New())
}

func TestIntersection(t *testing.T) {
	set1 := New("a", "b", "c")
	set2 := New("a", "x", "y")
	set3 := New("z", "b", "c")
	set4 := New("z", "b", "w")

	d := Intersection(set1, set2)
	require.Equal(t, d, New("a"))
	d = Intersection(set1, set3)
	require.Equal(t, d, New("b", "c"))
	d = Intersection(set2, set3)
	require.Equal(t, d, New())
	d = Intersection(set1, set3, set4)
	require.Equal(t, d, New("b"))
}
