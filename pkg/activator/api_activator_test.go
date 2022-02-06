/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Modifications Copyright 2022 Cortex Labs, Inc.
*/

package activator

import (
	"context"
	"testing"

	"github.com/cortexlabs/cortex/pkg/proxy"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestApiActivator_Try(t *testing.T) {
	t.Parallel()

	act := newAPIActivator(1, 1)

	errCh := make(chan error)
	waitCh := make(chan struct{})

	for i := 0; i < 3; i++ {

		go func() {
			errCh <- act.try(context.Background(), func() error {
				<-waitCh
				return nil
			}, &readinessTracker{ready: true})
		}()
	}

	err := <-errCh
	require.Error(t, err)
	require.True(t, errors.Is(err, proxy.ErrRequestQueueFull))

	for i := 0; i < 2; i++ {
		waitCh <- struct{}{}
		require.NoError(t, <-errCh)
	}
}
