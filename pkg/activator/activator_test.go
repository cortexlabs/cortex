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

package activator

import (
	"context"
	"errors"
	"testing"

	"github.com/cortexlabs/cortex/pkg/proxy"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	logger, err := config.Build()
	require.NoError(t, err)

	logr := logger.Sugar()

	return logr
}

func TestActivator_Try(t *testing.T) {
	log := newLogger(t)

	apiName := "test"
	act := &activator{
		apiActivators: map[string]*apiActivator{
			apiName: newAPIActivator(apiName, 1, 1),
		},
		logger: log,
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, ApiNameCtxKey, apiName)

	errCh := make(chan error)
	waitCh := make(chan struct{})

	for i := 0; i < 3; i++ {
		go func() {
			errCh <- act.Try(ctx, func() error {
				<-waitCh
				return nil
			})
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
