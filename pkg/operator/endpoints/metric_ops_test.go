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

package endpoints

import (
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/stretchr/testify/require"
)

func TestSumInt(t *testing.T) {
	intNilPtr := (*int)(nil)

	require.Equal(t, intNilPtr, SumInt([]*float64{}...))
	require.Equal(t, pointer.Int(1), SumInt(pointer.Float64(1)))
	require.Equal(t, pointer.Int(2), SumInt(pointer.Float64(1), pointer.Float64(1.5)))
}

func TestSumFloat(t *testing.T) {
	floatNilPtr := (*float64)(nil)

	require.Equal(t, floatNilPtr, SumFloat([]*float64{}...))
	require.Equal(t, pointer.Float64(1), SumFloat(pointer.Float64(1)))
	require.Equal(t, pointer.Float64(2.5), SumFloat(pointer.Float64(1), pointer.Float64(1.5)))
}

func TestMin(t *testing.T) {
	floatNilPtr := (*float64)(nil)

	require.Equal(t, floatNilPtr, Min([]*float64{}...))
	require.Equal(t, pointer.Float64(1), Min(pointer.Float64(1)))
	require.Equal(t, pointer.Float64(-1), Min(pointer.Float64(1), pointer.Float64(1.5), pointer.Float64(-1)))
}

func TestMax(t *testing.T) {
	floatNilPtr := (*float64)(nil)

	require.Equal(t, floatNilPtr, Max([]*float64{}...))
	require.Equal(t, pointer.Float64(1), Max(pointer.Float64(1)))
	require.Equal(t, pointer.Float64(1.5), Max(pointer.Float64(1), pointer.Float64(1.5), pointer.Float64(-1)))
}

func TestAvg(t *testing.T) {
	var err error
	var avg *float64

	floatNilPtr := (*float64)(nil)
	avg, err = Avg([]*float64{}, []*float64{})
	require.Equal(t, floatNilPtr, avg)
	require.NoError(t, err)

	avg, err = Avg([]*float64{pointer.Float64(1)}, []*float64{pointer.Float64(10)})
	require.Equal(t, pointer.Float64(1), avg)
	require.NoError(t, err)

	avg, err = Avg([]*float64{pointer.Float64(10)}, []*float64{pointer.Float64(1)})
	require.Equal(t, pointer.Float64(10), avg)
	require.NoError(t, err)

	avg, err = Avg([]*float64{pointer.Float64(1), pointer.Float64(4)}, []*float64{pointer.Float64(2), pointer.Float64(1)})
	require.Equal(t, pointer.Float64(2), avg)
	require.NoError(t, err)

	avg, err = Avg([]*float64{pointer.Float64(1)}, []*float64{pointer.Float64(2), pointer.Float64(1)})
	require.Equal(t, floatNilPtr, avg)
	require.Error(t, err)

	avg, err = Avg([]*float64{pointer.Float64(2)}, []*float64{pointer.Float64(0)})
	require.Equal(t, floatNilPtr, avg)
	require.Error(t, err)
}
