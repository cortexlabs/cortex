/*
Copyright 2022 Cortex Labs, Inc.

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

package slices_test

import (
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/stretchr/testify/require"
)

func TestStrSliceElementsMatch(t *testing.T) {
	var strs1 []string
	var strs2 []string

	strs1 = []string{}
	strs2 = []string{}
	require.True(t, slices.StrSliceElementsMatch(strs1, strs2))

	strs1 = []string{"1"}
	strs2 = []string{"1"}
	require.True(t, slices.StrSliceElementsMatch(strs1, strs2))

	strs1 = []string{"1", "2", "3"}
	strs2 = []string{"1", "2", "3"}
	require.True(t, slices.StrSliceElementsMatch(strs1, strs2))

	strs1 = []string{"1", "2", "3"}
	strs2 = []string{"1", "2", "3", "4"}
	require.False(t, slices.StrSliceElementsMatch(strs1, strs2))

	strs1 = []string{"1", "2", "3"}
	strs2 = []string{"1", "4", "3"}
	require.False(t, slices.StrSliceElementsMatch(strs1, strs2))

	strs1 = []string{"1", "2", "3"}
	strs2 = []string{"3", "2", "1"}
	require.True(t, slices.StrSliceElementsMatch(strs1, strs2))

	strs1 = []string{"2", "1", "2", "3"}
	strs2 = []string{"3", "2", "1", "2"}
	require.True(t, slices.StrSliceElementsMatch(strs1, strs2))
	require.Equal(t, []string{"2", "1", "2", "3"}, strs1) // ensure sort didn't get applied
	require.Equal(t, []string{"3", "2", "1", "2"}, strs2) // ensure sort didn't get applied

	strs1 = []string{"2", "1", "2", "3"}
	strs2 = []string{"3", "2", "1"}
	require.False(t, slices.StrSliceElementsMatch(strs1, strs2))
}

func TestHasString(t *testing.T) {
	cases := []struct {
		name     string
		slice    []string
		target   string
		expected bool
	}{
		{
			name:     "exists",
			slice:    []string{"a", "b", "c"},
			target:   "a",
			expected: true,
		},
		{
			name:     "doesn't exist",
			slice:    []string{"a", "b", "c"},
			target:   "d",
			expected: false,
		},
		{
			name:     "nil",
			slice:    nil,
			target:   "a",
			expected: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := slices.HasString(tt.slice, tt.target)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveString(t *testing.T) {
	cases := []struct {
		name     string
		slice    []string
		target   string
		expected []string
	}{
		{
			name:     "simple",
			slice:    []string{"a", "b", "c"},
			target:   "a",
			expected: []string{"b", "c"},
		},
		{
			name:     "repeated",
			slice:    []string{"a", "b", "c", "c"},
			target:   "c",
			expected: []string{"a", "b"},
		},
		{
			name:     "nil",
			slice:    nil,
			target:   "a",
			expected: nil,
		},
		{
			name:     "unchanged",
			slice:    []string{"a", "b", "c", "c"},
			target:   "d",
			expected: []string{"a", "b", "c", "c"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := slices.RemoveString(tt.slice, tt.target)
			require.Equal(t, tt.expected, result)
		})
	}
}
