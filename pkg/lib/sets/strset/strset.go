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

package strset

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type Set map[string]struct{}

var _keyExists = struct{}{}

// New creates and initializes a new Set.
func New(ts ...string) Set {
	s := make(Set)
	s.Add(ts...)
	return s
}

func FromSlice(items []string) Set {
	return New(items...)
}

// NewWithSize creates a new Set and gives make map a size hint.
func NewWithSize(size int) Set {
	return make(Set, size)
}

// Add includes the specified items (one or more) to the Set. The underlying
// Set s is modified. If passed nothing it silently returns.
func (s Set) Add(items ...string) {
	for _, item := range items {
		s[item] = _keyExists
	}
}

// Remove deletes the specified items from the Set. The underlying Set s is
// modified. If passed nothing it silently returns.
func (s Set) Remove(items ...string) {
	for _, item := range items {
		delete(s, item)
	}
}

// GetOne returns an item from the set or "" if the set is empty.
func (s Set) GetOne() string {
	for item := range s {
		return item
	}
	return ""
}

// GetOne2 returns an item from the set. The second value is a bool that is
// true if an item exists in the set, or false if the set is empty.
func (s Set) GetOne2() (string, bool) {
	for item := range s {
		return item, true
	}
	return "", false
}

// Pop deletes and returns an item from the Set. The underlying Set s is
// modified. If Set is empty, the zero value is returned.
func (s Set) Pop() string {
	for item := range s {
		delete(s, item)
		return item
	}
	return ""
}

// Pop2 tries to delete and return an item from the Set. The underlying Set s
// is modified. The second value is a bool that is true if the item existed in
// the set, and false if not. If Set is empty, the zero value and false are
// returned.
func (s Set) Pop2() (string, bool) {
	for item := range s {
		delete(s, item)
		return item, true
	}
	return "", false
}

// Has looks for the existence of items passed. It returns false if nothing is
// passed. For multiple items it returns true only if all of the items exist.
func (s Set) Has(items ...string) bool {
	has := false
	for _, item := range items {
		if _, has = s[item]; !has {
			break
		}
	}
	return has
}

// HasWithPrefix checks if at least one element of the set is the prefix of any of the passed items.
// It returns false if nothing is passed.
func (s Set) HasWithPrefix(items ...string) bool {
	for _, prefix := range items {
		for k := range s {
			if strings.HasPrefix(prefix, k) {
				return true
			}
		}
	}
	return false
}

// HasAny looks for the existence of any of the items passed.
// It returns false if nothing is passed.
// For multiple items it returns true if any of the items exist.
func (s Set) HasAny(items ...string) bool {
	has := false
	for _, item := range items {
		if _, has = s[item]; has {
			break
		}
	}
	return has
}

// Clear removes all items from the Set.
func (s *Set) Clear() {
	*s = make(Set)
}

// IsEqual test whether s and t are the same in size and have the same items.
func (s Set) IsEqual(t Set) bool {
	if len(s) != len(t) {
		return false
	}

	for item := range s {
		if !t.Has(item) {
			return false
		}
	}

	return true
}

// IsSubset tests whether t is a subset of s.
func (s Set) IsSubset(t Set) bool {
	if len(s) < len(t) {
		return false
	}

	for item := range t {
		if !s.Has(item) {
			return false
		}
	}

	return true
}

// IsSuperset tests whether t is a superset of s.
func (s Set) IsSuperset(t Set) bool {
	return t.IsSubset(s)
}

// Copy returns a new Set with a copy of s.
func (s Set) Copy() Set {
	u := make(Set, len(s))
	for item := range s {
		u[item] = _keyExists
	}
	return u
}

// String returns a string representation of s
func (s Set) String() string {
	v := make([]string, 0, len(s))
	for item := range s {
		v = append(v, fmt.Sprintf("%v", item))
	}
	return fmt.Sprintf("[%s]", strings.Join(v, ", "))
}

// List returns a slice of all items.
func (s Set) Slice() []string {
	v := make([]string, 0, len(s))
	for item := range s {
		v = append(v, item)
	}
	return v
}

// List returns a sorted slice of all items (a to z).
func (s Set) SliceSorted() []string {
	v := s.Slice()
	sort.Strings(v)
	return v
}

// Merge is like Union, however it modifies the current Set it's applied on
// with the given t Set.
func (s Set) Merge(sets ...Set) {
	for _, set := range sets {
		for item := range set {
			s[item] = _keyExists
		}
	}
}

// Subtract removes the Set items contained in sets from Set s
func (s Set) Subtract(sets ...Set) {
	for _, set := range sets {
		for item := range set {
			delete(s, item)
		}
	}
}

// Remove items until len(s) <= targetLen
func (s Set) Shrink(targetLen int) {
	for len(s) > targetLen {
		s.Pop()
	}
}

// remove items alphabetically until len(s) <= targetLen
func (s Set) ShrinkSorted(targetLen int) {
	if len(s) <= targetLen {
		return
	}

	sorted := s.SliceSorted()
	extras := sorted[targetLen:]
	s.Remove(extras...)
}

// Union is the merger of multiple sets. It returns a new set with all the
// elements present in all the sets that are passed.
func Union(sets ...Set) Set {
	maxPos := -1
	maxSize := 0

	// find which set is the largest and its size
	for i, set := range sets {
		if l := len(set); l > maxSize {
			maxSize = l
			maxPos = i
		}
	}
	if maxSize == 0 {
		return make(Set)
	}

	u := sets[maxPos].Copy()
	for i, set := range sets {
		if i == maxPos {
			continue
		}
		for item := range set {
			u[item] = _keyExists
		}
	}
	return u
}

// Difference returns a new set which contains items which are in the first
// set but not in the others.
func Difference(set1 Set, sets ...Set) Set {
	s := set1.Copy()
	for _, set := range sets {
		s.Subtract(set)
	}
	return s
}

// Intersection returns a new set which contains items that only exist in all
// given sets.
func Intersection(sets ...Set) Set {
	minPos := -1
	minSize := math.MaxInt64
	for i, set := range sets {
		if l := len(set); l < minSize {
			minSize = l
			minPos = i
		}
	}
	if minSize == math.MaxInt64 || minSize == 0 {
		return make(Set)
	}

	t := sets[minPos].Copy()
	for i, set := range sets {
		if i == minPos {
			continue
		}
		for item := range t {
			if _, has := set[item]; !has {
				delete(t, item)
			}
		}
	}
	return t
}

// SymmetricDifference returns a new set which s is the difference of items
// which are in one of either, but not in both.
func SymmetricDifference(s Set, t Set) Set {
	u := Difference(s, t)
	v := Difference(t, s)
	return Union(u, v)
}
