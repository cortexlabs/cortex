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

package context

import (
	"testing"

	"github.com/stretchr/testify/require"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

func TestExtractCortexResources(t *testing.T) {
	var resources []Resource

	resources = ExtractCortexResources(cr.MustReadYAMLStr(`@rc1`), rawCols)
	require.Equal(t, []Resource{rc1}, resources)

	resources = ExtractCortexResources(cr.MustReadYAMLStr(`@rc1`), nil)
	require.Equal(t, []Resource{}, resources)

	resources = ExtractCortexResources(cr.MustReadYAMLStr(`@rc1`), rawCols, resource.TransformedColumnType)
	require.Equal(t, []Resource{}, resources)

	resources = ExtractCortexResources(cr.MustReadYAMLStr(`[@rc1, rc2, @tc1]`), allResources)
	require.Equal(t, []Resource{rc1, tc1}, resources)

	resources = ExtractCortexResources(cr.MustReadYAMLStr(`[@rc1, rc2, @tc1]`), allResources, resource.RawColumnType)
	require.Equal(t, []Resource{rc1}, resources)

	resources = ExtractCortexResources(cr.MustReadYAMLStr(`[@rc1, rc2, @tc1]`), allResources, resource.TransformedColumnType)
	require.Equal(t, []Resource{tc1}, resources)

	// Check sorted by ID
	resources = ExtractCortexResources(cr.MustReadYAMLStr(`[@tc1, rc2, @rc1]`), allResources)
	require.Equal(t, []Resource{rc1, tc1}, resources)

	// Check duplicates
	resources = ExtractCortexResources(cr.MustReadYAMLStr(`[@rc1, rc2, @tc1, @rc1]`), allResources)
	require.Equal(t, []Resource{rc1, tc1}, resources)

	mixedInput := cr.MustReadYAMLStr(
		`
     map: {@agg1: @c1}
     str: @rc1
     floats: [@tc2]
     map2:
       map3:
         lat: @c2
         lon:
           @c3: agg2
           b: [@tc1, @agg3]
    `)

	resources = ExtractCortexResources(mixedInput, allResources)
	require.Equal(t, []Resource{c1, c2, c3, rc1, agg1, agg3, tc1, tc2}, resources)

	resources = ExtractCortexResources(mixedInput, allResources, resource.AggregateType)
	require.Equal(t, []Resource{agg1, agg3}, resources)

	resources = ExtractCortexResources(mixedInput, allResources, resource.AggregateType, resource.TransformedColumnType)
	require.Equal(t, []Resource{agg1, agg3, tc1, tc2}, resources)
}
