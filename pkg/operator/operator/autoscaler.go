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
	"log"
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
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

func autoscaleFn(initialDeployment *kapps.Deployment) (func() error, error) {
	minReplicas, err1 := getMinReplicas(initialDeployment)
	maxReplicas, err2 := getMaxReplicas(initialDeployment)
	threadsPerReplica, err3 := getThreadsPerReplicas(initialDeployment)
	targetQueueLen, err4 := getTargetQueueLength(initialDeployment)
	window, err5 := getWindow(initialDeployment)
	downscaleStabilizationPeriod, err6 := getDownscaleStabilizationPeriod(initialDeployment)
	upscaleStabilizationPeriod, err7 := getUpscaleStabilizationPeriod(initialDeployment)
	maxDownscaleFactor, err8 := getMaxDownscaleFactor(initialDeployment)
	maxUpscaleFactor, err9 := getMaxUpscaleFactor(initialDeployment)
	downscaleTolerance, err10 := getDownscaleTolerance(initialDeployment)
	upscaleTolerance, err11 := getUpscaleTolerance(initialDeployment)

	err := errors.FirstError(err1, err2, err3, err4, err5, err6, err7, err8, err9, err10, err11)
	if err != nil {
		return nil, err
	}

	apiName := initialDeployment.Labels["apiName"]
	currentReplicas := *initialDeployment.Spec.Replicas

	log.Printf("%s autoscaler init: min_replicas=%d, max_replicas=%d, window=%s, target_queue_length=%s, threads_per_replica=%d, downscale_tolerance=%s, upscale_tolerance=%s, downscale_stabilization_period=%s, upscale_stabilization_period=%s, max_downscale_factor=%s, max_upscale_factor=%s", apiName, minReplicas, maxReplicas, window, s.Float64(targetQueueLen), threadsPerReplica, s.Float64(downscaleTolerance), s.Float64(upscaleTolerance), downscaleStabilizationPeriod, upscaleStabilizationPeriod, s.Float64(maxDownscaleFactor), s.Float64(maxUpscaleFactor))

	var startTime time.Time
	recs := make(recommendations)

	return func() error {
		if startTime.IsZero() {
			startTime = time.Now()
		}

		totalInFlight, err := getInflightRequests() // TODO window will go here
		if err != nil {
			debug.Pp(err)
			return nil
		}
		if totalInFlight == nil {
			return nil
		}

		rawRecommendation := *totalInFlight / (float64(threadsPerReplica) + targetQueueLen)
		recommendation := int32(math.Ceil(rawRecommendation))

		if rawRecommendation < float64(currentReplicas) && rawRecommendation > float64(currentReplicas)*(1-downscaleTolerance) {
			recommendation = currentReplicas
		}

		if rawRecommendation > float64(currentReplicas) && rawRecommendation < float64(currentReplicas)*(1+upscaleTolerance) {
			recommendation = currentReplicas
		}

		// always allow subtraction of 1
		downscaleFactorFloor := libmath.MinInt32(currentReplicas-1, int32(math.Ceil(float64(currentReplicas)*maxDownscaleFactor)))
		if recommendation < downscaleFactorFloor {
			recommendation = downscaleFactorFloor
		}

		// always allow addition of 1
		upscaleFactorCeil := libmath.MaxInt32(currentReplicas+1, int32(math.Ceil(float64(currentReplicas)*maxUpscaleFactor)))
		if recommendation > upscaleFactorCeil {
			recommendation = upscaleFactorCeil
		}

		if recommendation < 1 {
			recommendation = 1
		}

		if recommendation < minReplicas {
			recommendation = minReplicas
		}

		if recommendation > maxReplicas {
			recommendation = maxReplicas
		}

		// Rule of thumb: any modifications that don't consider historical recommendations should be performed before
		// recording the recommendation, any modifications that use historical recommendations should be performed after
		recs.add(recommendation)

		// This is just for garbage collection
		recs.clearOlderThan(libtime.MaxDuration(downscaleStabilizationPeriod, upscaleStabilizationPeriod))

		request := recommendation

		downscaleStabilizationFloor := recs.maxSince(downscaleStabilizationPeriod)
		if time.Since(startTime) < downscaleStabilizationPeriod {
			if request < currentReplicas {
				request = currentReplicas
			}
		} else if downscaleStabilizationFloor != nil && request < *downscaleStabilizationFloor {
			request = *downscaleStabilizationFloor
		}

		upscaleStabilizationCeil := recs.minSince(upscaleStabilizationPeriod)
		if time.Since(startTime) < upscaleStabilizationPeriod {
			if request > currentReplicas {
				request = currentReplicas
			}
		} else if upscaleStabilizationCeil != nil && request > *upscaleStabilizationCeil {
			request = *upscaleStabilizationCeil
		}

		log.Printf("%s autoscaler tick: total_in_flight=%s, raw_recommendation=%s, downscale_factor_floor=%d, upscale_factor_ceil=%d, min_replicas=%d, max_replicas=%d, recommendation=%d, downscale_stabilization_floor=%s, upscale_stabilization_ceil=%s, current_replicas=%d, request=%d", apiName, s.Round(*totalInFlight, 2, 0), s.Round(rawRecommendation, 2, 0), downscaleFactorFloor, upscaleFactorCeil, minReplicas, maxReplicas, recommendation, s.ObjFlatNoQuotes(downscaleStabilizationFloor), s.ObjFlatNoQuotes(upscaleStabilizationCeil), currentReplicas, request)

		if currentReplicas != request {
			log.Printf("%s autoscaling event: %d -> %d", apiName, currentReplicas, request)

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
	}, nil
}

func getInflightRequests() (*float64, error) {
	endTime := time.Now().Truncate(time.Second)
	startTime := endTime.Add(-60 * time.Second)
	metricsDataQuery := cloudwatch.GetMetricDataInput{
		EndTime:   &endTime,
		StartTime: &startTime,
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id:    aws.String("inflight"),
				Label: aws.String("InFlight"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String("cortex"),
						MetricName: aws.String("in-flight"),
						Dimensions: []*cloudwatch.Dimension{
							&cloudwatch.Dimension{
								Name:  aws.String("apiName"),
								Value: aws.String("test"),
							},
						},
					},
					Stat:   aws.String("Sum"),
					Period: aws.Int64(10),
				},
			},
		},
	}

	output, err := config.AWS.CloudWatchMetrics().GetMetricData(&metricsDataQuery)
	if err != nil {
		return nil, err
	}

	if len(output.MetricDataResults[0].Timestamps) == 0 {
		return nil, nil
	}
	timestampCounter := 0
	for i, timeStamp := range output.MetricDataResults[0].Timestamps {
		if endTime.Sub(*timeStamp) < 20*time.Second {
			timestampCounter = i
		} else {
			break
		}
	}

	endTimeStampCounter := libmath.MinInt(timestampCounter+6, len(output.MetricDataResults[0].Timestamps))

	values := output.MetricDataResults[0].Values[timestampCounter:endTimeStampCounter]

	avg := 0.0
	for _, val := range values {
		avg += *val
	}
	avg = avg / float64(len(values))

	return &avg, nil
}
