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

package slices

import (
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/stretchr/testify/require"
)

var _float64NilPtr = (*float64)(nil)

func TestFloat64PtrSumInt(t *testing.T) {
	require.Equal(t, 0, Float64PtrSumInt(nil))
	require.Equal(t, 1, Float64PtrSumInt(pointer.Float64(1)))
	require.Equal(t, 2, Float64PtrSumInt(pointer.Float64(1), pointer.Float64(1.5)))
}

func TestFloat64PtrMin(t *testing.T) {
	require.Equal(t, _float64NilPtr, Float64PtrMin())
	require.Equal(t, _float64NilPtr, Float64PtrMin(nil))
	require.Equal(t, pointer.Float64(1), Float64PtrMin(pointer.Float64(1)))
	require.Equal(t, pointer.Float64(-1), Float64PtrMin(_float64NilPtr, pointer.Float64(1), pointer.Float64(-1)))
}

func TestFloat64PtrMax(t *testing.T) {
	require.Equal(t, _float64NilPtr, Float64PtrMax())
	require.Equal(t, _float64NilPtr, Float64PtrMax(nil))
	require.Equal(t, pointer.Float64(1), Float64PtrMax(pointer.Float64(1)))
	require.Equal(t, pointer.Float64(1.5), Float64PtrMax(pointer.Float64(1), pointer.Float64(1.5), _float64NilPtr))
}

func TestFloat64PtrAvg(t *testing.T) {
	var err error
	var avg *float64

	avg, err = Float64PtrAvg([]*float64{pointer.Float64(1)}, []*float64{pointer.Float64(10)})
	require.Equal(t, pointer.Float64(1), avg)
	require.NoError(t, err)

	avg, err = Float64PtrAvg([]*float64{pointer.Float64(10)}, []*float64{pointer.Float64(1)})
	require.Equal(t, pointer.Float64(10), avg)
	require.NoError(t, err)

	avg, err = Float64PtrAvg([]*float64{pointer.Float64(1), pointer.Float64(4), _float64NilPtr}, []*float64{pointer.Float64(2), pointer.Float64(1), pointer.Float64(1)})
	require.Equal(t, pointer.Float64(2), avg)
	require.NoError(t, err)

	avg, err = Float64PtrAvg([]*float64{pointer.Float64(1), pointer.Float64(4), pointer.Float64(1)}, []*float64{pointer.Float64(2), pointer.Float64(1), _float64NilPtr})
	require.Equal(t, pointer.Float64(2), avg)
	require.NoError(t, err)

	avg, err = Float64PtrAvg([]*float64{pointer.Float64(1), pointer.Float64(4), _float64NilPtr}, []*float64{pointer.Float64(2), pointer.Float64(1), _float64NilPtr})
	require.Equal(t, pointer.Float64(2), avg)
	require.NoError(t, err)

	avg, err = Float64PtrAvg([]*float64{pointer.Float64(1)}, []*float64{pointer.Float64(2), pointer.Float64(1)})
	require.Equal(t, _float64NilPtr, avg)
	require.Error(t, err)

	avg, err = Float64PtrAvg([]*float64{pointer.Float64(2)}, []*float64{pointer.Float64(0)})
	require.Equal(t, _float64NilPtr, avg)
	require.NoError(t, err)

	avg, err = Float64PtrAvg([]*float64{pointer.Float64(0)}, []*float64{pointer.Float64(2)})
	require.Equal(t, pointer.Float64(0), avg)
	require.NoError(t, err)

	avg, err = Float64PtrAvg([]*float64{nil}, []*float64{pointer.Float64(2)})
	require.Equal(t, _float64NilPtr, avg)
	require.NoError(t, err)

}
