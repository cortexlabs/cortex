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
	"fmt"
	"math"
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	math2 "github.com/cortexlabs/cortex/pkg/lib/math"
	time2 "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kapps "k8s.io/api/apps/v1"
)

// GetInFlightFunc is the function signature used by the autoscaler to retrieve
// the number of in-flight requests / messages
type GetInFlightFunc func(apiName string, window time.Duration) (*float64, error)

type recommendations map[time.Time]int32

func (recs recommendations) add(rec int32) {
	recs[time.Now()] = rec
}

func (recs recommendations) deleteOlderThan(period time.Duration) {
	for t := range recs {
		if time.Since(t) > period {
			delete(recs, t)
		}
	}
}

// Returns nil if no recommendations in the period
func (recs recommendations) maxSince(period time.Duration) *int32 {
	max := int32(math.MinInt32)
	foundRecommendation := false

	for t, rec := range recs {
		if time.Since(t) <= period && rec > max {
			max = rec
			foundRecommendation = true
		}
	}

	if !foundRecommendation {
		return nil
	}

	return &max
}

// Returns nil if no recommendations in the period
func (recs recommendations) minSince(period time.Duration) *int32 {
	min := int32(math.MaxInt32)
	foundRecommendation := false

	for t, rec := range recs {
		if time.Since(t) <= period && rec < min {
			min = rec
			foundRecommendation = true
		}
	}

	if !foundRecommendation {
		return nil
	}

	return &min
}

// AutoscaleFn returns the autoscaler function
func AutoscaleFn(initialDeployment *kapps.Deployment, apiSpec *spec.API, getInFlightFn GetInFlightFunc) (func() error, error) {
	if initialDeployment == nil {
		if apiSpec != nil {
			return nil, errors.ErrorUnexpected("unable to find api deployment", apiSpec.Name)
		}
		return nil, errors.ErrorUnexpected("unable to find api deployment")
	}

	if apiSpec == nil {
		apiName := initialDeployment.Labels["apiName"]
		return nil, errors.ErrorUnexpected("unable to find api spec", apiName)
	}

	autoscalingSpec, err := userconfig.AutoscalingFromAnnotations(initialDeployment)
	if err != nil {
		return nil, err
	}

	apiName := apiSpec.Name
	currentRequestedReplicas := *initialDeployment.Spec.Replicas

	apiLogger, err := operator.GetRealtimeAPILoggerFromSpec(apiSpec)
	if err != nil {
		return nil, err
	}

	apiLogger.Infof("%s autoscaler init", apiName)

	var startTime time.Time
	recs := make(recommendations)

	return func() error {
		if startTime.IsZero() {
			startTime = time.Now()
		}

		avgInFlight, err := getInFlightFn(apiName, autoscalingSpec.Window)
		if err != nil {
			return err
		}
		if avgInFlight == nil {
			apiLogger.Debugf("%s autoscaler tick: metrics not available yet", apiName)
			return nil
		}

		rawRecommendation := *avgInFlight / *autoscalingSpec.TargetInFlight
		recommendation := int32(math.Ceil(rawRecommendation))

		if rawRecommendation < float64(currentRequestedReplicas) && rawRecommendation > float64(currentRequestedReplicas)*(1-autoscalingSpec.DownscaleTolerance) {
			recommendation = currentRequestedReplicas
		}

		if rawRecommendation > float64(currentRequestedReplicas) && rawRecommendation < float64(currentRequestedReplicas)*(1+autoscalingSpec.UpscaleTolerance) {
			recommendation = currentRequestedReplicas
		}

		// always allow subtraction of 1
		downscaleFactorFloor := math2.MinInt32(currentRequestedReplicas-1, int32(math.Ceil(float64(currentRequestedReplicas)*autoscalingSpec.MaxDownscaleFactor)))
		if recommendation < downscaleFactorFloor {
			recommendation = downscaleFactorFloor
		}

		// always allow addition of 1
		upscaleFactorCeil := math2.MaxInt32(currentRequestedReplicas+1, int32(math.Ceil(float64(currentRequestedReplicas)*autoscalingSpec.MaxUpscaleFactor)))
		if recommendation > upscaleFactorCeil {
			recommendation = upscaleFactorCeil
		}

		if recommendation < autoscalingSpec.MinReplicas {
			recommendation = autoscalingSpec.MinReplicas
		}

		if recommendation > autoscalingSpec.MaxReplicas {
			recommendation = autoscalingSpec.MaxReplicas
		}

		// Rule of thumb: any modifications that don't consider historical recommendations should be performed before
		// recording the recommendation, any modifications that use historical recommendations should be performed after
		recs.add(recommendation)

		// This is just for garbage collection
		recs.deleteOlderThan(time2.MaxDuration(autoscalingSpec.DownscaleStabilizationPeriod, autoscalingSpec.UpscaleStabilizationPeriod))

		request := recommendation
		var downscaleStabilizationFloor *int32
		var upscaleStabilizationCeil *int32

		if request < currentRequestedReplicas {
			downscaleStabilizationFloor = recs.maxSince(autoscalingSpec.DownscaleStabilizationPeriod)
			if time.Since(startTime) < autoscalingSpec.DownscaleStabilizationPeriod {
				request = currentRequestedReplicas
			} else if downscaleStabilizationFloor != nil && request < *downscaleStabilizationFloor {
				request = *downscaleStabilizationFloor
			}
		}
		if request > currentRequestedReplicas {
			upscaleStabilizationCeil = recs.minSince(autoscalingSpec.UpscaleStabilizationPeriod)
			if time.Since(startTime) < autoscalingSpec.UpscaleStabilizationPeriod {
				request = currentRequestedReplicas
			} else if upscaleStabilizationCeil != nil && request > *upscaleStabilizationCeil {
				request = *upscaleStabilizationCeil
			}
		}

		apiLogger.Debugw(fmt.Sprintf("%s autoscaler tick", apiName),
			"autoscaling", map[string]interface{}{
				"avg_in_flight":                  *avgInFlight,
				"target_in_flight":               *autoscalingSpec.TargetInFlight,
				"raw_recommendation":             rawRecommendation,
				"current_replicas":               currentRequestedReplicas,
				"downscale_tolerance":            autoscalingSpec.DownscaleTolerance,
				"upscale_tolerance":              autoscalingSpec.UpscaleTolerance,
				"max_downscale_factor":           autoscalingSpec.MaxDownscaleFactor,
				"downscale_factor_floor":         downscaleFactorFloor,
				"max_upscale_factor":             autoscalingSpec.MaxUpscaleFactor,
				"upscale_factor_ceil":            upscaleFactorCeil,
				"min_replicas":                   autoscalingSpec.MinReplicas,
				"max_replicas":                   autoscalingSpec.MaxReplicas,
				"recommendation":                 recommendation,
				"downscale_stabilization_period": autoscalingSpec.DownscaleStabilizationPeriod.Seconds(),
				"downscale_stabilization_floor":  downscaleStabilizationFloor,
				"upscale_stabilization_period":   autoscalingSpec.UpscaleStabilizationPeriod.Seconds(),
				"upscale_stabilization_ceil":     upscaleStabilizationCeil,
				"request":                        request,
			},
		)

		if currentRequestedReplicas != request {
			apiLogger.Infof("%s autoscaling event: %d -> %d", apiName, currentRequestedReplicas, request)

			deployment, err := config.K8s.GetDeployment(initialDeployment.Name)
			if err != nil {
				return err
			}

			if deployment == nil {
				return errors.ErrorUnexpected("unable to find k8s deployment", apiName)
			}

			deployment.Spec.Replicas = &request

			if _, err := config.K8s.UpdateDeployment(deployment); err != nil {
				return err
			}

			currentRequestedReplicas = request
		}

		return nil
	}, nil
}
