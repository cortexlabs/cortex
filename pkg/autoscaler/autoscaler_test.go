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

package autoscaler

import (
	"sync"
	"testing"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
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

func TestAutoscaler_Awake(t *testing.T) {
	log := newLogger(t)

	mux := sync.RWMutex{}
	var latestRequest int32

	scalerMock := &ScalerFunc{
		ScaleFunc: func(apiName string, request int32) error {
			mux.Lock()
			defer mux.Unlock()

			latestRequest = request
			return nil
		},
		GetInFlightRequestsFunc: func(apiName string, window time.Duration) (*float64, error) {
			return pointer.Float64(0), nil
		},
		GetAutoscalingSpecFunc: func(apiName string) (*userconfig.Autoscaling, error) {
			return &userconfig.Autoscaling{
				MinReplicas:    0,
				MaxReplicas:    1,
				InitReplicas:   1,
				TargetInFlight: pointer.Float64(1),
				Window:         time.Second,
			}, nil
		},
		CurrentReplicasFunc: func(apiName string) (int32, error) {
			return 1, nil
		},
	}

	awakenStabilizationPeriod := 3 * time.Second
	autoScaler := &Autoscaler{
		logger:                    log,
		crons:                     make(map[string]cron.Cron),
		scalers:                   make(map[userconfig.Kind]Scaler),
		awakenMap:                 make(map[string]time.Time),
		awakenStabilizationPeriod: awakenStabilizationPeriod,
	}
	autoScaler.AddScaler(scalerMock, userconfig.RealtimeAPIKind)

	api := userconfig.Resource{
		Name: "test",
		Kind: userconfig.RealtimeAPIKind,
	}

	autoscaleFn, err := autoScaler.autoscaleFn(api)
	require.NoError(t, err)

	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				err := autoscaleFn()
				require.NoError(t, err)
			}
		}
	}()

	err = autoScaler.Awaken(api)
	require.NoError(t, err)

	_, ok := autoScaler.awakenMap[api.Name]
	require.True(t, ok)

	require.Never(t, func() bool {
		mux.RLock()
		defer mux.RUnlock()
		return latestRequest == 0
	}, awakenStabilizationPeriod, time.Second)
}
