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
	"fmt"
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
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
	debug.Pp(currentReplicas)
	recs := make(recommendations)

	return func() error {

		totalInFlight := getInflightRequests() // TODO window will go here
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

		fmt.Printf("requested replica: %d", request)
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

func getInflightRequests() float64 {
	debug.Pp("Hello!")
	endTime := time.Now().Truncate(time.Second)
	debug.Pp(endTime)
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
					Stat:   aws.String("Average"),
					Period: aws.Int64(10),
				},
			},
		},
	}
	output, err := config.AWS.CloudWatchMetrics().GetMetricData(&metricsDataQuery)
	if err != nil {
		debug.Pp(err)
	}

	timestampCounter := 0
	for i, timeStamp := range output.MetricDataResults[0].Timestamps {
		if endTime.Sub(*timeStamp) < 20*time.Second {
			timestampCounter = i
		} else {
			break
		}
	}
	value := output.MetricDataResults[0].Values[timestampCounter]
	debug.Pp(output.MetricDataResults[0])
	fmt.Printf("inflight: %f\n", *value)
	return *value
}
