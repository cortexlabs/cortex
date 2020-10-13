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

package realtimeapi

import (
	"log"
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kapps "k8s.io/api/apps/v1"
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

func autoscaleFn(initialDeployment *kapps.Deployment) (func() error, error) {
	autoscalingSpec, err := userconfig.AutoscalingFromAnnotations(initialDeployment)
	if err != nil {
		return nil, err
	}

	apiName := initialDeployment.Labels["apiName"]
	currentReplicas := *initialDeployment.Spec.Replicas

	log.Printf("%s autoscaler init", apiName)

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
			log.Printf("%s autoscaler tick: metrics not available yet", apiName)
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

		downscaleStabilizationFloor := recs.maxSince(autoscalingSpec.DownscaleStabilizationPeriod)
		if time.Since(startTime) < autoscalingSpec.DownscaleStabilizationPeriod {
			if request < currentReplicas {
				request = currentReplicas
			}
		} else if downscaleStabilizationFloor != nil && request < *downscaleStabilizationFloor {
			request = *downscaleStabilizationFloor
		}

		upscaleStabilizationCeil := recs.minSince(autoscalingSpec.UpscaleStabilizationPeriod)
		if time.Since(startTime) < autoscalingSpec.UpscaleStabilizationPeriod {
			if request > currentReplicas {
				request = currentReplicas
			}
		} else if upscaleStabilizationCeil != nil && request > *upscaleStabilizationCeil {
			request = *upscaleStabilizationCeil
		}

		log.Printf("%s autoscaler tick: avg_in_flight=%s, target_replica_concurrency=%s, raw_recommendation=%s, current_replicas=%d, downscale_tolerance=%s, upscale_tolerance=%s, max_downscale_factor=%s, downscale_factor_floor=%d, max_upscale_factor=%s, upscale_factor_ceil=%d, min_replicas=%d, max_replicas=%d, recommendation=%d, downscale_stabilization_period=%s, downscale_stabilization_floor=%s, upscale_stabilization_period=%s, upscale_stabilization_ceil=%s, request=%d", apiName, s.Round(*avgInFlight, 2, 0), s.Float64(*autoscalingSpec.TargetReplicaConcurrency), s.Round(rawRecommendation, 2, 0), currentReplicas, s.Float64(autoscalingSpec.DownscaleTolerance), s.Float64(autoscalingSpec.UpscaleTolerance), s.Float64(autoscalingSpec.MaxDownscaleFactor), downscaleFactorFloor, s.Float64(autoscalingSpec.MaxUpscaleFactor), upscaleFactorCeil, autoscalingSpec.MinReplicas, autoscalingSpec.MaxReplicas, recommendation, autoscalingSpec.DownscaleStabilizationPeriod, s.ObjFlatNoQuotes(downscaleStabilizationFloor), autoscalingSpec.UpscaleStabilizationPeriod, s.ObjFlatNoQuotes(upscaleStabilizationCeil), request)

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

func getInflightRequests(apiName string, window time.Duration) (*float64, error) {
	endTime := time.Now().Truncate(time.Second)
	startTime := endTime.Add(-2 * window)
	metricsDataQuery := cloudwatch.GetMetricDataInput{
		EndTime:   &endTime,
		StartTime: &startTime,
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id:    aws.String("inflight"),
				Label: aws.String("InFlight"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String(config.Cluster.ClusterName),
						MetricName: aws.String("in-flight"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("apiName"),
								Value: aws.String(apiName),
							},
						},
					},
					Stat:   aws.String("Sum"),
					Period: aws.Int64(10),
				},
			},
		},
	}

	output, err := config.AWS.CloudWatch().GetMetricData(&metricsDataQuery)
	if err != nil {
		return nil, err
	}
	if len(output.MetricDataResults) == 0 {
		return nil, nil
	}

	timestampCounter := -1
	for i, timeStamp := range output.MetricDataResults[0].Timestamps {
		if endTime.Sub(*timeStamp) < 20*time.Second {
			timestampCounter = i
		} else {
			break
		}
	}

	if timestampCounter == -1 {
		return nil, nil // no metrics were available in the last 2 tick intervals
	}

	steps := int(window.Nanoseconds() / spec.AutoscalingTickInterval.Nanoseconds())

	endTimeStampCounter := libmath.MinInt(timestampCounter+steps, len(output.MetricDataResults[0].Timestamps))

	values := output.MetricDataResults[0].Values[timestampCounter:endTimeStampCounter]
	if len(values) == 0 {
		return nil, nil
	}

	avg := 0.0
	for _, val := range values {
		avg += *val
	}
	avg = avg / float64(len(values))

	return &avg, nil
}
