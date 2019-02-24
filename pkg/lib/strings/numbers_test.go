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

	"github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/stretchr/testify/require"
)

func TestRound(t *testing.T) {
	require.Equal(t, strings.Round(1.111, 2, false), "1.11")
	require.Equal(t, strings.Round(1.111, 3, false), "1.111")
	require.Equal(t, strings.Round(1.111, 4, false), "1.111")
	require.Equal(t, strings.Round(1.555, 2, false), "1.56")
	require.Equal(t, strings.Round(1.555, 3, false), "1.555")
	require.Equal(t, strings.Round(1.555, 4, false), "1.555")
	require.Equal(t, strings.Round(1.100, 2, false), "1.1")

	require.Equal(t, strings.Round(1.111, 2, true), "1.11")
	require.Equal(t, strings.Round(1.111, 3, true), "1.111")
	require.Equal(t, strings.Round(1.111, 4, true), "1.1110")
	require.Equal(t, strings.Round(1.555, 2, true), "1.56")
	require.Equal(t, strings.Round(1.555, 3, true), "1.555")
	require.Equal(t, strings.Round(1.555, 4, true), "1.5550")
	require.Equal(t, strings.Round(1.100, 2, true), "1.10")

	require.Equal(t, strings.Round(30, 0, true), "30")
	require.Equal(t, strings.Round(2, 1, true), "2.0")
	require.Equal(t, strings.Round(1, 2, true), "1.00")
	require.Equal(t, strings.Round(20, 3, true), "20.000")

	require.Equal(t, strings.Round(30, 0, false), "30")
	require.Equal(t, strings.Round(2, 1, false), "2")
	require.Equal(t, strings.Round(1, 2, false), "1")
	require.Equal(t, strings.Round(20, 3, false), "20")
}
