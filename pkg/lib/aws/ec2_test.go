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

package aws

import (
	"fmt"
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/stretchr/testify/require"
)

func TestParseInstanceType(t *testing.T) {
	var testcases = []struct {
		instanceType string
		expected     ParsedInstanceType
	}{
		{"t3.small", ParsedInstanceType{"t", 3, strset.New(), "small"}},
		{"g4dn.xlarge", ParsedInstanceType{"g", 4, strset.New("d", "n"), "xlarge"}},
		{"inf1.24xlarge", ParsedInstanceType{"inf", 1, strset.New(), "24xlarge"}},
		{"u-9tb1.metal", ParsedInstanceType{"u-9tb", 1, strset.New(), "metal"}},
	}

	invalidTypes := []string{
		"badtype",
		"badtype.large",
		"badtype1.large",
		"badtype2ad.large",
	}

	for _, testcase := range testcases {
		parsed, err := ParseInstanceType(testcase.instanceType)
		require.NoError(t, err)
		require.Equal(t, testcase.expected.Family, parsed.Family, fmt.Sprintf("unexpected family for input: %s", testcase.instanceType))
		require.Equal(t, testcase.expected.Generation, parsed.Generation, fmt.Sprintf("unexpected generation for input: %s", testcase.instanceType))
		require.ElementsMatch(t, testcase.expected.Capabilities.Slice(), parsed.Capabilities.Slice(), fmt.Sprintf("unexpected capabilities for input: %s", testcase.instanceType))
		require.Equal(t, testcase.expected.Size, parsed.Size, fmt.Sprintf("unexpected size for input: %s", testcase.instanceType))
	}

	for _, instanceType := range invalidTypes {
		_, err := ParseInstanceType(instanceType)
		require.Error(t, err)
	}

	for instanceType := range AllInstanceTypes {
		_, err := ParseInstanceType(instanceType)
		require.NoError(t, err)
	}
}
