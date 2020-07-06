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

// Find retention period in https://aws.amazon.com/cloudwatch/faqs/

func GetJobMetrics(jobKey spec.JobKey, startTime time.Time, endTime time.Time) (*metrics.JobMetrics, error) {
	batchMetrics := metrics.JobMetrics{}

	err := getMetricsFunc(&jobKey, 60*60, &startTime, &endTime, &batchMetrics)()
	if err != nil {
		return nil, err
	}

	return &batchMetrics, nil
}

func GetRealTimeJobMetrics(jobKey spec.JobKey) (*metrics.JobMetrics, error) {
	// Get realtime metrics for the seconds elapsed in the latest minute
	realTimeEnd := time.Now().Truncate(time.Second)
	realTimeStart := realTimeEnd.Truncate(time.Minute)

	realTimeMetrics := metrics.JobMetrics{}
	batchMetrics := metrics.JobMetrics{}
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

func getMetricsFunc(jobKey *spec.JobKey, period int64, startTime *time.Time, endTime *time.Time, metrics *metrics.JobMetrics) func() error {
	return func() error {
		metricDataResults, err := queryMetrics(jobKey, period, startTime, endTime)
		if err != nil {
			return err
		}
		jobMetrics, err := extractJobStats(metricDataResults)
		if err != nil {
			return err
		}

		metrics.MergeInPlace(*jobMetrics)

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

func extractJobStats(metricsDataResults []*cloudwatch.MetricDataResult) (*metrics.JobMetrics, error) {
	var jobStats metrics.JobMetrics
	var partitionCounts []*float64
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
		case *metricData.Label == "AverageTimePerPartition":
			latencyAvgs = metricData.Values
		case *metricData.Label == "Total":
			partitionCounts = metricData.Values
		}
	}

	avg, err := slices.Float64PtrAvg(latencyAvgs, partitionCounts)
	if err != nil {
		return nil, err
	}
	jobStats.AverageTimePerPartition = avg

	jobStats.TotalCompleted = jobStats.Succeeded + jobStats.Failed
	return &jobStats, nil
}

func getAPIDimensions(jobKey *spec.JobKey) []*cloudwatch.Dimension {
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

func getAPIDimensionsCounter(jobKey *spec.JobKey) []*cloudwatch.Dimension {
	return append(
		getAPIDimensions(jobKey),
		&cloudwatch.Dimension{
			Name:  aws.String("metric_type"),
			Value: aws.String("counter"),
		},
	)
}

func getAPIDimensionsHistogram(jobKey *spec.JobKey) []*cloudwatch.Dimension {
	return append(
		getAPIDimensions(jobKey),
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
					Namespace:  aws.String(config.Cluster.LogGroup),
					MetricName: aws.String("Succeeded"),
					Dimensions: getAPIDimensionsCounter(jobKey),
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
					Namespace:  aws.String(config.Cluster.LogGroup),
					MetricName: aws.String("Failed"),
					Dimensions: getAPIDimensionsCounter(jobKey),
				},
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
		},
		{
			Id:    aws.String("average_time_per_partition"),
			Label: aws.String("AverageTimePerPartition"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cluster.LogGroup),
					MetricName: aws.String("TimePerPartition"),
					Dimensions: getAPIDimensionsHistogram(jobKey),
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
					Namespace:  aws.String(config.Cluster.LogGroup),
					MetricName: aws.String("TimePerPartition"),
					Dimensions: getAPIDimensionsHistogram(jobKey),
				},
				Stat:   aws.String("SampleCount"),
				Period: aws.Int64(period),
			},
		},
	}
}
