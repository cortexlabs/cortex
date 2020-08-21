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
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func GetMultipleMetrics(apis []spec.API) ([]metrics.Metrics, error) {
	allMetrics := make([]metrics.Metrics, len(apis))
	fns := make([]func() error, len(apis))

	for i := range apis {
		localIdx := i
		api := apis[i]
		fns[i] = func() error {
			metrics, err := GetMetrics(&api)
			if err != nil {
				return err
			}
			allMetrics[localIdx] = *metrics
			return nil
		}
	}

	if len(fns) > 0 {
		err := parallel.RunFirstErr(fns[0], fns[1:]...)
		if err != nil {
			return nil, err
		}
	}

	return allMetrics, nil
}

func GetMetrics(api *spec.API) (*metrics.Metrics, error) {
	// Get realtime metrics for the seconds elapsed in the latest minute
	realTimeEnd := time.Now().Truncate(time.Second)
	realTimeStart := realTimeEnd.Truncate(time.Minute)

	realTimeMetrics := metrics.Metrics{}
	batchMetrics := metrics.Metrics{}
	requestList := []func() error{}

	if realTimeStart.Before(realTimeEnd) {
		requestList = append(requestList, getMetricsFunc(api, 1, &realTimeStart, &realTimeEnd, &realTimeMetrics))
	}

	batchEnd := realTimeStart
	batchStart := batchEnd.Add(-14 * 24 * time.Hour) // two weeks ago
	requestList = append(requestList, getMetricsFunc(api, 60*60, &batchStart, &batchEnd, &batchMetrics))

	err := parallel.RunFirstErr(requestList[0], requestList[1:]...)
	if err != nil {
		return nil, err
	}

	mergedMetrics := realTimeMetrics.Merge(batchMetrics)
	mergedMetrics.APIName = api.Name
	return &mergedMetrics, nil
}

func getMetricsFunc(api *spec.API, period int64, startTime *time.Time, endTime *time.Time, metrics *metrics.Metrics) func() error {
	return func() error {
		metricDataResults, err := queryMetrics(api, period, startTime, endTime)
		if err != nil {
			return err
		}
		networkStats, err := extractNetworkMetrics(metricDataResults)
		if err != nil {
			return err
		}
		metrics.NetworkStats = networkStats

		if api.Monitoring != nil {
			if api.Monitoring.ModelType == userconfig.ClassificationModelType {
				metrics.ClassDistribution = extractClassificationMetrics(metricDataResults)
			} else {
				regressionStats, err := extractRegressionMetrics(metricDataResults)
				if err != nil {
					return err
				}
				metrics.RegressionStats = regressionStats
			}
		}
		return nil
	}
}

func queryMetrics(api *spec.API, period int64, startTime *time.Time, endTime *time.Time) ([]*cloudwatch.MetricDataResult, error) {
	allMetrics := getNetworkStatsDef(api, period)

	if api.Monitoring != nil {
		if api.Monitoring.ModelType == userconfig.ClassificationModelType {
			classMetrics, err := getClassesMetricDef(api, period)
			if err != nil {
				return nil, err
			}
			allMetrics = append(allMetrics, classMetrics...)
		} else {
			regressionMetrics := getRegressionMetricDef(api, period)
			allMetrics = append(allMetrics, regressionMetrics...)
		}
	}

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

func extractNetworkMetrics(metricsDataResults []*cloudwatch.MetricDataResult) (*metrics.NetworkStats, error) {
	var networkStats metrics.NetworkStats
	var requestCounts []*float64
	var latencyAvgs []*float64

	for _, metricData := range metricsDataResults {
		if metricData.Values == nil {
			continue
		}

		switch {
		case *metricData.Label == "2XX":
			networkStats.Code2XX = slices.Float64PtrSumInt(metricData.Values...)
		case *metricData.Label == "4XX":
			networkStats.Code4XX = slices.Float64PtrSumInt(metricData.Values...)
		case *metricData.Label == "5XX":
			networkStats.Code5XX = slices.Float64PtrSumInt(metricData.Values...)
		case *metricData.Label == "Latency":
			latencyAvgs = metricData.Values
		case *metricData.Label == "RequestCount":
			requestCounts = metricData.Values
		}
	}

	avg, err := slices.Float64PtrAvg(latencyAvgs, requestCounts)
	if err != nil {
		return nil, err
	}
	networkStats.Latency = avg

	networkStats.Total = networkStats.Code2XX + networkStats.Code4XX + networkStats.Code5XX
	return &networkStats, nil
}

func extractClassificationMetrics(metricsDataResults []*cloudwatch.MetricDataResult) map[string]int {
	classDistribution := map[string]int{}
	for _, metricData := range metricsDataResults {
		if metricData.Values == nil {
			continue
		}

		if strings.HasPrefix(*metricData.Label, "class_") {
			className := (*metricData.Label)[len("class_"):]
			classDistribution[className] = slices.Float64PtrSumInt(metricData.Values...)
		}
	}
	return classDistribution
}

func extractRegressionMetrics(metricsDataResults []*cloudwatch.MetricDataResult) (*metrics.RegressionStats, error) {
	var regressionStats metrics.RegressionStats
	var predictionAvgs []*float64
	var requestCounts []*float64

	for _, metricData := range metricsDataResults {
		if metricData.Values == nil {
			continue
		}

		switch {
		case *metricData.Label == "Min":
			regressionStats.Min = slices.Float64PtrMin(metricData.Values...)
		case *metricData.Label == "Max":
			regressionStats.Max = slices.Float64PtrMax(metricData.Values...)
		case *metricData.Label == "SampleCount":
			regressionStats.SampleCount = slices.Float64PtrSumInt(metricData.Values...)
			requestCounts = metricData.Values
		case *metricData.Label == "Avg":
			predictionAvgs = metricData.Values
		}
	}

	avg, err := slices.Float64PtrAvg(predictionAvgs, requestCounts)
	if err != nil {
		return nil, err
	}
	regressionStats.Avg = avg

	return &regressionStats, nil
}

func getAPIDimensions(api *spec.API) []*cloudwatch.Dimension {
	return []*cloudwatch.Dimension{
		{
			Name:  aws.String("APIName"),
			Value: aws.String(api.Name),
		},
		{
			Name:  aws.String("APIID"),
			Value: aws.String(api.ID),
		},
	}
}

func getAPIDimensionsCounter(api *spec.API) []*cloudwatch.Dimension {
	return append(
		getAPIDimensions(api),
		&cloudwatch.Dimension{
			Name:  aws.String("metric_type"),
			Value: aws.String("counter"),
		},
	)
}

func getAPIDimensionsHistogram(api *spec.API) []*cloudwatch.Dimension {
	return append(
		getAPIDimensions(api),
		&cloudwatch.Dimension{
			Name:  aws.String("metric_type"),
			Value: aws.String("histogram"),
		},
	)
}

func getRegressionMetricDef(api *spec.API, period int64) []*cloudwatch.MetricDataQuery {
	metric := &cloudwatch.Metric{
		Namespace:  aws.String(config.Cluster.ClusterName),
		MetricName: aws.String("Prediction"),
		Dimensions: getAPIDimensionsHistogram(api),
	}

	regressionMetric := []*cloudwatch.MetricDataQuery{
		{
			Id:    aws.String("min"),
			Label: aws.String("Min"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: metric,
				Stat:   aws.String("Minimum"),
				Period: aws.Int64(period),
			},
		},
		{
			Id:    aws.String("max"),
			Label: aws.String("Max"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: metric,
				Stat:   aws.String("Maximum"),
				Period: aws.Int64(period),
			},
		},
		{
			Id:    aws.String("sample_count"),
			Label: aws.String("SampleCount"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: metric,
				Stat:   aws.String("SampleCount"),
				Period: aws.Int64(period),
			},
		},
		{
			Id:    aws.String("avg"),
			Label: aws.String("Avg"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: metric,
				Stat:   aws.String("Average"),
				Period: aws.Int64(period),
			},
		},
	}

	return regressionMetric
}

func getNetworkStatsDef(api *spec.API, period int64) []*cloudwatch.MetricDataQuery {
	statusCodes := []string{"2XX", "4XX", "5XX"}
	networkDataQueries := make([]*cloudwatch.MetricDataQuery, len(statusCodes)+2)

	for i, code := range statusCodes {
		dimensions := getAPIDimensionsCounter(api)
		statusCodeDimensions := append(dimensions, &cloudwatch.Dimension{
			Name:  aws.String("Code"),
			Value: aws.String(code),
		})
		networkDataQueries[i] = &cloudwatch.MetricDataQuery{
			Id:    aws.String("datapoints_" + code),
			Label: aws.String(code),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cluster.ClusterName),
					MetricName: aws.String("StatusCode"),
					Dimensions: statusCodeDimensions,
				},
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
		}
	}

	networkDataQueries[3] = &cloudwatch.MetricDataQuery{
		Id:    aws.String("latency"),
		Label: aws.String("Latency"),
		MetricStat: &cloudwatch.MetricStat{
			Metric: &cloudwatch.Metric{
				Namespace:  aws.String(config.Cluster.ClusterName),
				MetricName: aws.String("Latency"),
				Dimensions: getAPIDimensionsHistogram(api),
			},
			Stat:   aws.String("Average"),
			Period: aws.Int64(period),
		},
	}

	networkDataQueries[4] = &cloudwatch.MetricDataQuery{
		Id:    aws.String("request_count"),
		Label: aws.String("RequestCount"),
		MetricStat: &cloudwatch.MetricStat{
			Metric: &cloudwatch.Metric{
				Namespace:  aws.String(config.Cluster.ClusterName),
				MetricName: aws.String("Latency"),
				Dimensions: getAPIDimensionsHistogram(api),
			},
			Stat:   aws.String("SampleCount"),
			Period: aws.Int64(period),
		},
	}
	return networkDataQueries
}

func getClassesMetricDef(api *spec.API, period int64) ([]*cloudwatch.MetricDataQuery, error) {
	prefix := filepath.Join(api.MetadataRoot, "classes") + "/"
	classes, err := config.AWS.ListS3Prefix(config.Cluster.Bucket, prefix, false, pointer.Int64(int64(consts.MaxClassesPerMonitoringRequest)))
	if err != nil {
		return nil, err
	}

	if len(classes) == 0 {
		return nil, nil
	}

	classMetricQueries := []*cloudwatch.MetricDataQuery{}

	for i, classObj := range classes {
		classKey := *classObj.Key
		urlSplit := strings.Split(classKey, "/")
		encodedClassName := urlSplit[len(urlSplit)-1]
		decodedBytes, err := base64.URLEncoding.DecodeString(encodedClassName)
		if err != nil {
			return nil, errors.Wrap(err, "encoded class name", encodedClassName)
		}

		className := string(decodedBytes)
		if len(className) == 0 {
			continue
		}

		classMetricQueries = append(classMetricQueries, &cloudwatch.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("id_%d", i)),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cluster.ClusterName),
					MetricName: aws.String("Prediction"),
					Dimensions: append(getAPIDimensionsCounter(api), &cloudwatch.Dimension{
						Name:  aws.String("Class"),
						Value: aws.String(className),
					}),
				},
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
			Label: aws.String("class_" + className),
		})
	}
	return classMetricQueries, nil
}
