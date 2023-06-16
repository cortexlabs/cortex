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

func generateRecommendationTimeline(t *testing.T, recs []int32, interval time.Duration) *recommendations {
	t.Helper()

	startTime := time.Now()
	recsTimeline := map[time.Time]int32{}

	for i := range recs {
		timestamp := startTime.Add(time.Duration(i) * interval)
		recsTimeline[timestamp] = recs[i]
	}

	return &recommendations{
		timeline: recsTimeline,
	}
}

func TestAutoscaler_AutoscaleFn(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	interval := 250 * time.Millisecond

	cases := []struct {
		name                   string
		autoscalingSpec        userconfig.Autoscaling
		inFlight               float64
		currentReplicas        int32
		recommendationTimeline []int32
		expectedRequest        *int32
	}{
		{
			name: "no scale below zero within stabilization period",
			autoscalingSpec: userconfig.Autoscaling{
				MinReplicas:                  0,
				MaxReplicas:                  5,
				InitReplicas:                 0,
				TargetInFlight:               pointer.Float64(1),
				Window:                       4 * interval,
				DownscaleStabilizationPeriod: time.Second,
				UpscaleStabilizationPeriod:   time.Second,
				MaxDownscaleFactor:           0.75,
				MaxUpscaleFactor:             1.5,
				DownscaleTolerance:           0.05,
				UpscaleTolerance:             0.05,
			},
			inFlight:        0,
			currentReplicas: 1,
			expectedRequest: nil,
		},
		{
			name: "downscale no stabilization",
			autoscalingSpec: userconfig.Autoscaling{
				MinReplicas:                  1,
				MaxReplicas:                  5,
				InitReplicas:                 1,
				TargetInFlight:               pointer.Float64(1),
				Window:                       4 * interval,
				DownscaleStabilizationPeriod: 0,
				UpscaleStabilizationPeriod:   0,
				MaxDownscaleFactor:           0.5,
				MaxUpscaleFactor:             1.5,
				DownscaleTolerance:           0.05,
				UpscaleTolerance:             0.05,
			},
			inFlight:               2,
			currentReplicas:        5,
			recommendationTimeline: []int32{5, 2, 2, 2},
			expectedRequest:        pointer.Int32(3),
		},
		{
			name: "downscale with stabilization",
			autoscalingSpec: userconfig.Autoscaling{
				MinReplicas:                  1,
				MaxReplicas:                  5,
				InitReplicas:                 1,
				TargetInFlight:               pointer.Float64(1),
				Window:                       4 * interval,
				DownscaleStabilizationPeriod: time.Second,
				UpscaleStabilizationPeriod:   time.Second,
				MaxDownscaleFactor:           0.5,
				MaxUpscaleFactor:             1.5,
				DownscaleTolerance:           0.05,
				UpscaleTolerance:             0.05,
			},
			inFlight:               5,
			currentReplicas:        1,
			recommendationTimeline: []int32{5, 5, 2, 2},
			expectedRequest:        nil,
		},
		{
			name: "upscale no stabilization",
			autoscalingSpec: userconfig.Autoscaling{
				MinReplicas:                  1,
				MaxReplicas:                  5,
				InitReplicas:                 1,
				TargetInFlight:               pointer.Float64(1),
				Window:                       4 * interval,
				DownscaleStabilizationPeriod: 0,
				UpscaleStabilizationPeriod:   0,
				MaxDownscaleFactor:           0.5,
				MaxUpscaleFactor:             1.5,
				DownscaleTolerance:           0.05,
				UpscaleTolerance:             0.05,
			},
			inFlight:        3,
			currentReplicas: 1,
			expectedRequest: pointer.Int32(2),
		},
		{
			name: "upscale with stabilization",
			autoscalingSpec: userconfig.Autoscaling{
				MinReplicas:                  1,
				MaxReplicas:                  5,
				InitReplicas:                 1,
				TargetInFlight:               pointer.Float64(1),
				Window:                       4 * interval,
				DownscaleStabilizationPeriod: time.Second,
				UpscaleStabilizationPeriod:   time.Second,
				MaxDownscaleFactor:           0.5,
				MaxUpscaleFactor:             1.5,
				DownscaleTolerance:           0.05,
				UpscaleTolerance:             0.05,
			},
			inFlight:               5,
			currentReplicas:        2,
			recommendationTimeline: []int32{2, 2, 2, 5},
			expectedRequest:        nil,
		},
		{
			name: "no upscale below current replicas",
			autoscalingSpec: userconfig.Autoscaling{
				MinReplicas:                  0,
				MaxReplicas:                  5,
				InitReplicas:                 0,
				TargetInFlight:               pointer.Float64(1),
				Window:                       4 * interval,
				DownscaleStabilizationPeriod: time.Second,
				UpscaleStabilizationPeriod:   time.Second,
				MaxDownscaleFactor:           0.75,
				MaxUpscaleFactor:             1.5,
				DownscaleTolerance:           0.05,
				UpscaleTolerance:             0.05,
			},
			inFlight:               3,
			currentReplicas:        2,
			recommendationTimeline: []int32{0, 1, 2, 3},
			expectedRequest:        nil,
		},
	}

	for i, tt := range cases {
		localTT := tt
		localI := i
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var latestRequest *int32

			scalerMock := &ScalerFunc{
				ScaleFunc: func(apiName string, request int32) error {
					latestRequest = pointer.Int32(request)
					return nil
				},
				GetInFlightRequestsFunc: func(apiName string, window time.Duration) (*float64, error) {
					return pointer.Float64(localTT.inFlight), nil
				},
				GetAutoscalingSpecFunc: func(apiName string) (*userconfig.Autoscaling, error) {
					return &cases[localI].autoscalingSpec, nil
				},
				CurrentRequestedReplicasFunc: func(apiName string) (int32, error) {
					return localTT.currentReplicas, nil
				},
			}

			autoScaler := &Autoscaler{
				logger:  log,
				crons:   make(map[string]cron.Cron),
				scalers: make(map[userconfig.Kind]Scaler),
				recs:    make(map[string]*recommendations),
			}
			autoScaler.AddScaler(scalerMock, userconfig.RealtimeAPIKind)

			apiName := "test"
			api := userconfig.Resource{
				Name: apiName,
				Kind: userconfig.RealtimeAPIKind,
			}

			autoScaler.recs[apiName] = generateRecommendationTimeline(t, localTT.recommendationTimeline, interval)
			autoscaleFn, err := autoScaler.autoscaleFn(api)
			require.NoError(t, err)

			time.Sleep(interval)

			err = autoscaleFn()
			require.NoError(t, err)

			require.Equal(t, localTT.expectedRequest, latestRequest)
		})
	}

}

func TestAutoscaler_Awake(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	mux := sync.RWMutex{}
	var latestRequest int32

	downscaleStabilizationPeriod := 3 * time.Second
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
				MinReplicas:                  0,
				MaxReplicas:                  1,
				InitReplicas:                 1,
				TargetInFlight:               pointer.Float64(1),
				Window:                       500 * time.Millisecond,
				DownscaleStabilizationPeriod: downscaleStabilizationPeriod,
				MaxDownscaleFactor:           0.75,
				MaxUpscaleFactor:             1.5,
			}, nil
		},
		CurrentRequestedReplicasFunc: func(apiName string) (int32, error) {
			return 0, nil
		},
	}

	autoScaler := &Autoscaler{
		logger:  log,
		crons:   make(map[string]cron.Cron),
		scalers: make(map[userconfig.Kind]Scaler),
		recs:    make(map[string]*recommendations),
	}
	autoScaler.AddScaler(scalerMock, userconfig.RealtimeAPIKind)

	apiName := "test"
	api := userconfig.Resource{
		Name: apiName,
		Kind: userconfig.RealtimeAPIKind,
	}

	autoscaleFn, err := autoScaler.autoscaleFn(api)
	require.NoError(t, err)

	ticker := time.NewTicker(250 * time.Millisecond)
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

	require.Never(t, func() bool {
		mux.RLock()
		defer mux.RUnlock()
		return latestRequest != 1
	}, downscaleStabilizationPeriod, time.Second)
}

func TestAutoscaler_MinReplicas(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	mux := sync.RWMutex{}
	var latestRequest int32

	minReplicas := int32(5)
	maxReplicas := int32(10)

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
				MinReplicas:        minReplicas,
				MaxReplicas:        maxReplicas,
				InitReplicas:       minReplicas,
				TargetInFlight:     pointer.Float64(1),
				Window:             500 * time.Millisecond,
				MaxDownscaleFactor: 0.75,
				MaxUpscaleFactor:   1.5,
			}, nil
		},
		CurrentRequestedReplicasFunc: func(apiName string) (int32, error) {
			return minReplicas + 1, nil
		},
	}

	autoScaler := &Autoscaler{
		logger:  log,
		crons:   make(map[string]cron.Cron),
		scalers: make(map[userconfig.Kind]Scaler),
		recs:    make(map[string]*recommendations),
	}
	autoScaler.AddScaler(scalerMock, userconfig.RealtimeAPIKind)

	apiName := "test"
	api := userconfig.Resource{
		Name: apiName,
		Kind: userconfig.RealtimeAPIKind,
	}

	autoscaleFn, err := autoScaler.autoscaleFn(api)
	require.NoError(t, err)

	ticker := time.NewTicker(250 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				err := autoscaleFn()
				require.NoError(t, err)
			}
		}
	}()

	require.Never(t, func() bool {
		mux.RLock()
		defer mux.RUnlock()
		return latestRequest < minReplicas
	}, 3*time.Second, time.Second)
}

func TestAutoscaler_MaxReplicas(t *testing.T) {
	t.Parallel()
	log := newLogger(t)

	mux := sync.RWMutex{}
	var latestRequest int32

	minReplicas := int32(5)
	maxReplicas := int32(10)

	scalerMock := &ScalerFunc{
		ScaleFunc: func(apiName string, request int32) error {
			mux.Lock()
			defer mux.Unlock()

			latestRequest = request
			return nil
		},
		GetInFlightRequestsFunc: func(apiName string, window time.Duration) (*float64, error) {
			return pointer.Float64(20), nil
		},
		GetAutoscalingSpecFunc: func(apiName string) (*userconfig.Autoscaling, error) {
			return &userconfig.Autoscaling{
				MinReplicas:        minReplicas,
				MaxReplicas:        maxReplicas,
				InitReplicas:       minReplicas,
				TargetInFlight:     pointer.Float64(1),
				Window:             500 * time.Millisecond,
				MaxDownscaleFactor: 0.75,
				MaxUpscaleFactor:   1.5,
			}, nil
		},
		CurrentRequestedReplicasFunc: func(apiName string) (int32, error) {
			return minReplicas, nil
		},
	}

	autoScaler := &Autoscaler{
		logger:  log,
		crons:   make(map[string]cron.Cron),
		scalers: make(map[userconfig.Kind]Scaler),
		recs:    make(map[string]*recommendations),
	}
	autoScaler.AddScaler(scalerMock, userconfig.RealtimeAPIKind)

	apiName := "test"
	api := userconfig.Resource{
		Name: apiName,
		Kind: userconfig.RealtimeAPIKind,
	}

	autoscaleFn, err := autoScaler.autoscaleFn(api)
	require.NoError(t, err)

	ticker := time.NewTicker(250 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				err := autoscaleFn()
				require.NoError(t, err)
			}
		}
	}()

	require.Never(t, func() bool {
		mux.RLock()
		defer mux.RUnlock()
		return latestRequest > maxReplicas
	}, 3*time.Second, time.Second)
}
