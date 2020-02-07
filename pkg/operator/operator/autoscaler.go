/*
Copyright 2020 Cortex Labs, Inc.

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

package operator

import (
	"math"
	"math/rand"
	"time"

	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	kapps "k8s.io/api/apps/v1"
)

type recommendations map[time.Time]int32

func (recs recommendations) add(rec int32) {
	recs[time.Now()] = rec
}

func (recs recommendations) clearOlderThan(period time.Duration) {
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

func autoscaleFn(initialDeployment *kapps.Deployment) func() error {
	// window := 60 * time.Second
	var targetQueueLen float64 = 0
	var concurrency int32 = 1
	downscaleStabilizationPeriod := 5 * time.Minute
	upscaleStabilizationPeriod := 0 * time.Minute
	var maxUpscaleFactor float64 = 100   // must be > 1
	var maxDownscaleFactor float64 = 0.5 // must be < 1

	currentReplicas := *initialDeployment.Spec.Replicas

	recs := make(recommendations)

	return func() error {
		totalInFlight := rand.Float64() * 100 // TODO window will go here

		recommendationFloat := totalInFlight / (float64(concurrency) + targetQueueLen)
		recommendation := int32(math.Ceil(recommendationFloat))

		// always allow addition of 1
		upscaleFactorCeil := libmath.MaxInt32(currentReplicas+1, int32(math.Ceil(float64(currentReplicas)*maxUpscaleFactor)))
		if recommendation > upscaleFactorCeil {
			recommendation = upscaleFactorCeil
		}

		// always allow subtraction of 1
		downscaleFactorFloor := libmath.MinInt32(currentReplicas-1, int32(math.Ceil(float64(currentReplicas)*maxDownscaleFactor)))
		if recommendation < downscaleFactorFloor {
			recommendation = downscaleFactorFloor
		}

		if recommendation < 1 {
			recommendation = 1
		}

		recs.add(recommendation)
		recs.clearOlderThan(libtime.MaxDuration(downscaleStabilizationPeriod, upscaleStabilizationPeriod))

		request := recommendation
		downscaleStabilizationFloor := recs.maxSince(downscaleStabilizationPeriod)
		if downscaleStabilizationFloor != nil && recommendation < *downscaleStabilizationFloor {
			request = *downscaleStabilizationFloor
		}

		upscaleStabilizationCeil := recs.minSince(upscaleStabilizationPeriod)
		if upscaleStabilizationCeil != nil && recommendation > *upscaleStabilizationCeil {
			request = *upscaleStabilizationCeil
		}

		if currentReplicas != request {
			deployment, err := config.K8s.GetDeployment(initialDeployment.Name)
			if err != nil {
				return err
			}

			deployment.Spec.Replicas = &request

			if _, err := config.K8s.UpdateDeployment(deployment); err != nil {
				return err
			}

			currentReplicas = request
		}

		return nil
	}
}
