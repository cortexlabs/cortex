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

package structs

import (
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/stretchr/testify/require"
)

type SampleStruct struct {
	AField string
	BField *string
	CField *bool
}

func TestDeepCopy(t *testing.T) {
	t.Parallel()

	var a SampleStruct
	b := SampleStruct{
		AField: "fox",
		BField: pointer.String("bull"),
	}

	err := DeepCopy(&a, &b)
	require.NoError(t, err)

	require.EqualValues(t, b.AField, "fox")
	require.EqualValues(t, b.AField, a.AField)

	require.True(t, a.BField != nil)
	require.True(t, b.BField != nil)
	require.True(t, a.BField != b.BField)
	require.True(t, a.CField == nil)
	require.True(t, b.CField == nil)

	require.EqualValues(t, *a.BField, "bull")
	require.EqualValues(t, *a.BField, *b.BField)
}
