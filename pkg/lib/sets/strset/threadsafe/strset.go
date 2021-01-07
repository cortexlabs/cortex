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

package threadsafe

import (
	"sync"

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

// New creates and initializes a new Set.
type Set struct {
	sync.RWMutex
	s strset.Set
}

func New(ts ...string) *Set {
	set := Set{}
	set.s = strset.New(ts...)
	return &set
}

func FromSlice(items []string) *Set {
	return New(items...)
}

// NewWithSize creates a new Set and gives make map a size hint.
func NewWithSize(size int) *Set {
	set := Set{}
	set.s = strset.NewWithSize(size)
	return &set
}

func (s *Set) Len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.s)
}

func (s *Set) ToStrset() strset.Set {
	s.RLock()
	defer s.RUnlock()
	return s.s.Copy()
}

// Add includes the specified items (one or more) to the Set. The underlying
// Set s is modified. If passed nothing it silently returns.
func (s *Set) Add(items ...string) {
	s.Lock()
	defer s.Unlock()
	s.s.Add(items...)
}

// Remove deletes the specified items from the Set. The underlying Set s is
// modified. If passed nothing it silently returns.
func (s *Set) Remove(items ...string) {
	s.Lock()
	defer s.Unlock()
	s.s.Remove(items...)
}

// GetOne returns an item from the set or "" if the set is empty.
func (s *Set) GetOne() string {
	s.RLock()
	defer s.RUnlock()
	return s.s.GetOne()
}

// GetOne2 returns an item from the set. The second value is a bool that is
// true if an item exists in the set, or false if the set is empty.
func (s *Set) GetOne2() (string, bool) {
	s.RLock()
	defer s.RUnlock()
	return s.s.GetOne2()
}

// Pop deletes and returns an item from the Set. The underlying Set s is
// modified. If Set is empty, the zero value is returned.
func (s *Set) Pop() string {
	s.Lock()
	defer s.Unlock()
	return s.s.Pop()
}

// Pop2 tries to delete and return an item from the Set. The underlying Set s
// is modified. The second value is a bool that is true if the item existed in
// the set, and false if not. If Set is empty, the zero value and false are
// returned.
func (s *Set) Pop2() (string, bool) {
	s.Lock()
	defer s.Unlock()
	return s.s.Pop2()
}

// Has looks for the existence of items passed. It returns false if nothing is
// passed. For multiple items it returns true only if all of the items exist.
func (s *Set) Has(items ...string) bool {
	s.RLock()
	defer s.RUnlock()
	return s.s.Has(items...)
}

// HasAny looks for the existence of any of the items passed.
// It returns false if nothing is passed.
// For multiple items it returns true if any of the items exist.
func (s *Set) HasAny(items ...string) bool {
	s.RLock()
	defer s.RUnlock()
	return s.s.HasAny(items...)
}

// Clear removes all items from the Set.
func (s *Set) Clear() {
	s.Lock()
	defer s.Unlock()
	s.s.Clear()
}

// IsEqual test whether s and t are the same in size and have the same items.
func (s *Set) IsEqual(t strset.Set) bool {
	s.RLock()
	defer s.RUnlock()
	return s.s.IsEqual(t)
}

// IsEqualThreadsafe test whether s and t are the same in size and have the same items.
func (s *Set) IsEqualThreadsafe(t *Set) bool {
	s.RLock()
	defer s.RUnlock()
	t.RLock()
	defer t.RUnlock()
	return s.s.IsEqual(t.s)
}

// IsSubset tests whether t is a subset of s.
func (s *Set) IsSubset(t strset.Set) bool {
	s.RLock()
	defer s.RUnlock()
	return s.s.IsSubset(t)
}

// IsSubsetThreadsafe tests whether t is a subset of s.
func (s *Set) IsSubsetThreadsafe(t *Set) bool {
	s.RLock()
	defer s.RUnlock()
	t.RLock()
	defer t.RUnlock()
	return s.s.IsSubset(t.s)
}

// IsSuperset tests whether t is a superset of s.
func (s *Set) IsSuperset(t strset.Set) bool {
	s.RLock()
	defer s.RUnlock()
	return s.s.IsSuperset(t)
}

// IsSupersetThreadsafe tests whether t is a superset of s.
func (s *Set) IsSupersetThreadsafe(t *Set) bool {
	s.RLock()
	defer s.RUnlock()
	t.RLock()
	defer t.RUnlock()
	return s.s.IsSuperset(t.s)
}

// Copy returns a new Set with a copy of s.
func (s *Set) Copy() strset.Set {
	s.RLock()
	defer s.RUnlock()
	return s.s.Copy()
}

// CopyToThreadsafe returns a new Set with a copy of s.
func (s *Set) CopyToThreadsafe() *Set {
	s.RLock()
	defer s.RUnlock()

	newSet := Set{}
	newSet.s = s.s.Copy()
	return &newSet
}

// String returns a string representation of s
func (s *Set) String() string {
	s.RLock()
	defer s.RUnlock()
	return s.s.String()
}

// List returns a slice of all items.
func (s *Set) Slice() []string {
	s.RLock()
	defer s.RUnlock()
	return s.s.Slice()
}

// List returns a sorted slice of all items (a to z).
func (s *Set) SliceSorted() []string {
	s.RLock()
	defer s.RUnlock()
	return s.s.SliceSorted()
}

// Merge is like Union, however it modifies the current Set it's applied on
// with the given t Set.
func (s *Set) Merge(sets ...strset.Set) {
	s.Lock()
	defer s.Unlock()
	s.s.Merge(sets...)
}

// MergeThreadsafe is like UnionThreadsafe, however it modifies the current Set it's applied on
// with the given t Set.
func (s *Set) MergeThreadsafe(sets ...*Set) {
	s.Lock()
	defer s.Unlock()

	for _, set := range sets {
		set.RLock()
		s.s.Merge(set.s)
		set.RUnlock()
	}
}

// Subtract removes the Set items contained in sets from Set s
func (s *Set) Subtract(sets ...strset.Set) {
	s.Lock()
	defer s.Unlock()
	s.s.Subtract(sets...)
}

// SubtractThreadsafe removes the Set items contained in sets from Set s
func (s *Set) SubtractThreadsafe(sets ...*Set) {
	s.Lock()
	defer s.Unlock()

	for _, set := range sets {
		set.RLock()
		s.s.Subtract(set.s)
		set.RUnlock()
	}
}

// Remove items until len(s) <= targetLen
func (s *Set) Shrink(targetLen int) {
	s.Lock()
	defer s.Unlock()
	s.s.Shrink(targetLen)
}

// remove items alphabetically until len(s) <= targetLen
func (s *Set) ShrinkSorted(targetLen int) {
	s.Lock()
	defer s.Unlock()
	s.s.ShrinkSorted(targetLen)
}

// Union is the merger of multiple sets. It returns a new set with all the
// elements present in all the sets that are passed.
func Union(set1 *Set, sets ...strset.Set) *Set {
	finalSet := set1.CopyToThreadsafe()
	for _, set := range sets {
		finalSet.s.Merge(set)
	}
	return finalSet
}

// UnionThreadsafe is the merger of multiple sets. It returns a new set with all the
// elements present in all the sets that are passed.
func UnionThreadsafe(sets ...*Set) *Set {
	finalSet := New()
	for _, set := range sets {
		set.RLock()
		finalSet.s.Merge(set.s)
		set.RUnlock()
	}
	return finalSet
}

// Difference returns a new set which contains items which are in the first
// set but not in the others.
func Difference(set1 *Set, sets ...strset.Set) *Set {
	s := set1.CopyToThreadsafe()
	for _, set := range sets {
		s.s.Subtract(set)
	}
	return s
}

// DifferenceThreadsafe returns a new set which contains items which are in in the first
// set but not in the others.
func DifferenceThreadsafe(set1 *Set, sets ...*Set) *Set {
	s := set1.CopyToThreadsafe()
	for _, set := range sets {
		set.RLock()
		s.s.Subtract(set.s)
		set.RUnlock()
	}
	return s
}

// Intersection returns a new set which contains items that only exist in all
// given sets.
func Intersection(set1 *Set, sets ...strset.Set) *Set {
	t := set1.CopyToThreadsafe()
	for _, set := range sets {
		for item := range t.s {
			if _, has := set[item]; !has {
				delete(t.s, item)
			}
		}
	}
	return t
}

// IntersectionThreadsafe returns a new set which contains items that only exist in all
// given sets.
func IntersectionThreadsafe(set1 *Set, sets ...*Set) *Set {
	t := set1.CopyToThreadsafe()
	for _, set := range sets {
		set.RLock()
		for item := range t.s {
			if _, has := set.s[item]; !has {
				delete(t.s, item)
			}
		}
		set.RUnlock()
	}
	return t
}

// SymmetricDifferenceThreadsafe returns a new set which s is the difference of items
// which are in one of either, but not in both.
func SymmetricDifferenceThreadsafe(s *Set, t *Set) *Set {
	u := DifferenceThreadsafe(s, t)
	v := DifferenceThreadsafe(t, s)
	return UnionThreadsafe(u, v)
}
