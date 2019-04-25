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

package strings_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	s "github.com/cortexlabs/cortex/pkg/operator/api/strings"
)

func TestLongestCommonPrefix(t *testing.T) {
	var strs []string
	var expected string

	strs = []string{
		"12345",
	}
	expected = "12345"
	require.Equal(t, expected, s.LongestCommonPrefix(strs...))

	strs = []string{
		"12345",
		"12345678",
	}
	expected = "12345"
	require.Equal(t, expected, s.LongestCommonPrefix(strs...))

	strs = []string{
		"12345",
		"12345678",
		"1239",
	}
	expected = "123"
	require.Equal(t, expected, s.LongestCommonPrefix(strs...))

	strs = []string{
		"123",
		"456",
	}
	expected = ""
	require.Equal(t, expected, s.LongestCommonPrefix(strs...))
}
