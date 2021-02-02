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

package realtimeapi

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/prometheus/common/model"
	kapps "k8s.io/api/apps/v1"
)

const (
	_prometheusQueryTimeoutSeconds = 10
)

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

func autoscaleFn(initialDeployment *kapps.Deployment, apiSpec *spec.API) (func() error, error) {
	autoscalingSpec, err := userconfig.AutoscalingFromAnnotations(initialDeployment)
	if err != nil {
		return nil, err
	}

	apiName := apiSpec.Name
	currentReplicas := *initialDeployment.Spec.Replicas

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

		avgInFlight, err := getInflightRequests(apiName, autoscalingSpec.Window)
		if err != nil {
			return err
		}
		if avgInFlight == nil {
			apiLogger.Debugf("%s autoscaler tick: metrics not available yet", apiName)
			return nil
		}

		rawRecommendation := *avgInFlight / *autoscalingSpec.TargetReplicaConcurrency
		recommendation := int32(math.Ceil(rawRecommendation))

		if rawRecommendation < float64(currentReplicas) && rawRecommendation > float64(currentReplicas)*(1-autoscalingSpec.DownscaleTolerance) {
			recommendation = currentReplicas
		}

		if rawRecommendation > float64(currentReplicas) && rawRecommendation < float64(currentReplicas)*(1+autoscalingSpec.UpscaleTolerance) {
			recommendation = currentReplicas
		}

		// always allow subtraction of 1
		downscaleFactorFloor := libmath.MinInt32(currentReplicas-1, int32(math.Ceil(float64(currentReplicas)*autoscalingSpec.MaxDownscaleFactor)))
		if recommendation < downscaleFactorFloor {
			recommendation = downscaleFactorFloor
		}

		// always allow addition of 1
		upscaleFactorCeil := libmath.MaxInt32(currentReplicas+1, int32(math.Ceil(float64(currentReplicas)*autoscalingSpec.MaxUpscaleFactor)))
		if recommendation > upscaleFactorCeil {
			recommendation = upscaleFactorCeil
		}

		if recommendation < 1 {
			recommendation = 1
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
		recs.deleteOlderThan(libtime.MaxDuration(autoscalingSpec.DownscaleStabilizationPeriod, autoscalingSpec.UpscaleStabilizationPeriod))

		request := recommendation
		var downscaleStabilizationFloor *int32
		var upscaleStabilizationCeil *int32

		if request < currentReplicas {
			downscaleStabilizationFloor = recs.maxSince(autoscalingSpec.DownscaleStabilizationPeriod)
			if time.Since(startTime) < autoscalingSpec.DownscaleStabilizationPeriod {
				request = currentReplicas
			} else if downscaleStabilizationFloor != nil && request < *downscaleStabilizationFloor {
				request = *downscaleStabilizationFloor
			}
		}
		if request > currentReplicas {
			upscaleStabilizationCeil = recs.minSince(autoscalingSpec.UpscaleStabilizationPeriod)
			if time.Since(startTime) < autoscalingSpec.UpscaleStabilizationPeriod {
				request = currentReplicas
			} else if upscaleStabilizationCeil != nil && request > *upscaleStabilizationCeil {
				request = *upscaleStabilizationCeil
			}
		}

		apiLogger.Debugw(fmt.Sprintf("%s autoscaler tick", apiName),
			"autoscaling", map[string]interface{}{
				"avg_in_flight":                  s.Round(*avgInFlight, 2, 0),
				"target_replica_concurrency":     s.Float64(*autoscalingSpec.TargetReplicaConcurrency),
				"raw_recommendation":             s.Round(rawRecommendation, 2, 0),
				"current_replicas":               currentReplicas,
				"downscale_tolerance":            s.Float64(autoscalingSpec.DownscaleTolerance),
				"upscale_tolerance":              s.Float64(autoscalingSpec.UpscaleTolerance),
				"max_downscale_factor":           s.Float64(autoscalingSpec.MaxDownscaleFactor),
				"downscale_factor_floor":         downscaleFactorFloor,
				"max_upscale_factor":             s.Float64(autoscalingSpec.MaxUpscaleFactor),
				"upscale_factor_ceil":            upscaleFactorCeil,
				"min_replicas":                   autoscalingSpec.MinReplicas,
				"max_replicas":                   autoscalingSpec.MaxReplicas,
				"recommendation":                 recommendation,
				"downscale_stabilization_period": autoscalingSpec.DownscaleStabilizationPeriod,
				"downscale_stabilization_floor":  s.ObjFlatNoQuotes(downscaleStabilizationFloor),
				"upscale_stabilization_period":   autoscalingSpec.UpscaleStabilizationPeriod,
				"upscale_stabilization_ceil":     s.ObjFlatNoQuotes(upscaleStabilizationCeil),
				"request":                        request,
			},
		)

		if currentReplicas != request {
			apiLogger.Infof("%s autoscaling event: %d -> %d", apiName, currentReplicas, request)

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

			currentReplicas = request
		}

		return nil
	}, nil
}

func getInflightRequests(apiName string, window time.Duration) (*float64, error) {
	windowSeconds := int64(window.Seconds())

	// PromQL query:
	// 	sum(sum_over_time(cortex_in_flight_requests{api_name="<apiName>"}[60s])) /
	//	sum(count_over_time(cortex_in_flight_requests{api_name="<apiName>"}[60s]))
	query := fmt.Sprintf(
		"sum(sum_over_time(cortex_in_flight_requests{api_name=\"%s\"}[%ds])) / "+
			"max(count_over_time(cortex_in_flight_requests{api_name=\"%s\"}[%ds]))",
		apiName, windowSeconds,
		apiName, windowSeconds,
	)

	ctx, cancel := context.WithTimeout(context.Background(), _prometheusQueryTimeoutSeconds*time.Second)
	defer cancel()

	valuesQuery, err := config.Prometheus.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	values, ok := valuesQuery.(model.Vector)
	if !ok {
		return nil, errors.ErrorUnexpected("failed to convert prometheus metric to vector")
	}

	// no values available
	if values.Len() == 0 {
		return nil, nil
	}

	avgInflightRequests := float64(values[0].Value)

	return &avgInflightRequests, nil
}
