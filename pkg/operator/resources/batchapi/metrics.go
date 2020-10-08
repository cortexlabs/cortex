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

package batchapi

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

// Find data retention period at https://aws.amazon.com/cloudwatch/faqs/

func getCompletedBatchMetrics(jobKey spec.JobKey, startTime time.Time, endTime time.Time) (*metrics.BatchMetrics, error) {
	batchMetrics := metrics.BatchMetrics{}

	if time.Now().Sub(endTime) < 2*time.Hour {
		return getRealTimeBatchMetrics(jobKey)
	}

	minimumEndTime := time.Now()
	if time.Now().Sub(endTime) > 2*time.Minute {
		minimumEndTime = endTime.Add(2 * time.Minute)
	}

	err := getMetricsFunc(&jobKey, 60*60, &startTime, &minimumEndTime, &batchMetrics)()
	if err != nil {
		return nil, err
	}

	return &batchMetrics, nil
}

func getRealTimeBatchMetrics(jobKey spec.JobKey) (*metrics.BatchMetrics, error) {
	// Get realtime metrics for the seconds elapsed in the latest minute
	realTimeEnd := time.Now().Truncate(time.Second)
	realTimeStart := realTimeEnd.Truncate(time.Minute)

	realTimeMetrics := metrics.BatchMetrics{}
	batchMetrics := metrics.BatchMetrics{}
	requestList := []func() error{}

	if realTimeStart.Before(realTimeEnd) {
		requestList = append(requestList, getMetricsFunc(&jobKey, 1, &realTimeStart, &realTimeEnd, &realTimeMetrics))
	}

	batchEnd := realTimeStart
	batchStart := batchEnd.Add(-14 * 24 * time.Hour) // two weeks ago
	requestList = append(requestList, getMetricsFunc(&jobKey, 60*60, &batchStart, &batchEnd, &batchMetrics))

	err := parallel.RunFirstErr(requestList[0], requestList[1:]...)
	if err != nil {
		return nil, err
	}

	mergedMetrics := realTimeMetrics.Merge(batchMetrics)
	return &mergedMetrics, nil
}

func getMetricsFunc(jobKey *spec.JobKey, period int64, startTime *time.Time, endTime *time.Time, metrics *metrics.BatchMetrics) func() error {
	return func() error {
		metricDataResults, err := queryMetrics(jobKey, period, startTime, endTime)
		if err != nil {
			return err
		}

		batchMetrics, err := extractJobStats(metricDataResults)
		if err != nil {
			return err
		}

		metrics.MergeInPlace(*batchMetrics)

		return nil
	}
}

func queryMetrics(jobKey *spec.JobKey, period int64, startTime *time.Time, endTime *time.Time) ([]*cloudwatch.MetricDataResult, error) {
	allMetrics := batchMetricsDef(jobKey, period)

	metricsDataQuery := cloudwatch.GetMetricDataInput{
		EndTime:           endTime,
		StartTime:         startTime,
		MetricDataQueries: allMetrics,
	}
	output, err := config.AWS.CloudWatch().GetMetricData(&metricsDataQuery)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return output.MetricDataResults, nil
}

func extractJobStats(metricsDataResults []*cloudwatch.MetricDataResult) (*metrics.BatchMetrics, error) {
	var jobStats metrics.BatchMetrics
	var batchCounts []*float64
	var latencyAvgs []*float64

	for _, metricData := range metricsDataResults {
		if metricData.Values == nil {
			continue
		}

		switch {
		case *metricData.Label == "Succeeded":
			jobStats.Succeeded = slices.Float64PtrSumInt(metricData.Values...)
		case *metricData.Label == "Failed":
			jobStats.Failed = slices.Float64PtrSumInt(metricData.Values...)
		case *metricData.Label == "AverageTimePerBatch":
			latencyAvgs = metricData.Values
		case *metricData.Label == "Total":
			batchCounts = metricData.Values
		}
	}

	avg, err := slices.Float64PtrAvg(latencyAvgs, batchCounts)
	if err != nil {
		return nil, err
	}
	jobStats.AverageTimePerBatch = avg

	return &jobStats, nil
}

func getJobDimensions(jobKey *spec.JobKey) []*cloudwatch.Dimension {
	return []*cloudwatch.Dimension{
		{
			Name:  aws.String("APIName"),
			Value: aws.String(jobKey.APIName),
		},
		{
			Name:  aws.String("JobID"),
			Value: aws.String(jobKey.ID),
		},
	}
}

func getJobDimensionsCounter(jobKey *spec.JobKey) []*cloudwatch.Dimension {
	return append(
		getJobDimensions(jobKey),
		&cloudwatch.Dimension{
			Name:  aws.String("metric_type"),
			Value: aws.String("counter"),
		},
	)
}

func getJobDimensionsHistogram(jobKey *spec.JobKey) []*cloudwatch.Dimension {
	return append(
		getJobDimensions(jobKey),
		&cloudwatch.Dimension{
			Name:  aws.String("metric_type"),
			Value: aws.String("histogram"),
		},
	)
}

func batchMetricsDef(jobKey *spec.JobKey, period int64) []*cloudwatch.MetricDataQuery {
	return []*cloudwatch.MetricDataQuery{
		{
			Id:    aws.String("succeeded"),
			Label: aws.String("Succeeded"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cluster.ClusterName),
					MetricName: aws.String("Succeeded"),
					Dimensions: getJobDimensionsCounter(jobKey),
				},
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
		},
		{
			Id:    aws.String("failed"),
			Label: aws.String("Failed"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cluster.ClusterName),
					MetricName: aws.String("Failed"),
					Dimensions: getJobDimensionsCounter(jobKey),
				},
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
		},
		{
			Id:    aws.String("average_time_per_batch"),
			Label: aws.String("AverageTimePerBatch"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cluster.ClusterName),
					MetricName: aws.String("TimePerBatch"),
					Dimensions: getJobDimensionsHistogram(jobKey),
				},
				Stat:   aws.String("Average"),
				Period: aws.Int64(period),
			},
		},
		{
			Id:    aws.String("total"),
			Label: aws.String("Total"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cluster.ClusterName),
					MetricName: aws.String("TimePerBatch"),
					Dimensions: getJobDimensionsHistogram(jobKey),
				},
				Stat:   aws.String("SampleCount"),
				Period: aws.Int64(period),
			},
		},
	}
}
