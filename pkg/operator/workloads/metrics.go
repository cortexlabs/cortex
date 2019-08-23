/*
Copyright 2019 Cortex Labs, Inc.

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

package workloads

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
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func GetMetrics(appName, apiName string) (*schema.APIMetrics, error) {
	ctx := CurrentContext(appName)
	api := ctx.APIs[apiName]

	apiSavedStatus, err := getAPISavedStatus(api.ID, api.WorkloadID, appName)
	if err != nil {
		return nil, err
	}

	if apiSavedStatus == nil {
		return nil, errors.Wrap(ErrorAPIInitializing(), api.Name)
	}

	if apiSavedStatus.Start == nil {
		return nil, errors.Wrap(ErrorAPIInitializing(), api.Name)
	}

	apiStartTime := apiSavedStatus.Start.Truncate(time.Second)

	// Get realtime metrics for the seconds elapsed in the latest minute
	realTimeEnd := time.Now().Truncate(time.Second)
	realTimeStart := realTimeEnd.Truncate(time.Minute)

	realTimeMetrics := schema.APIMetrics{}
	batchMetrics := schema.APIMetrics{}

	requestList := []func() error{}
	if realTimeStart.Before(realTimeEnd) {
		requestList = append(requestList, getAPIMetricsFunc(ctx, api, 1, &realTimeStart, &realTimeEnd, &realTimeMetrics))
	}

	if apiStartTime.Before(realTimeStart) {
		batchEnd := realTimeStart
		twoWeeksAgo := batchEnd.Add(-14 * 24 * time.Hour)

		var batchStart time.Time
		if twoWeeksAgo.Before(*apiSavedStatus.Start) {
			batchStart = *apiSavedStatus.Start
		} else {
			batchStart = twoWeeksAgo
		}
		requestList = append(requestList, getAPIMetricsFunc(ctx, api, 60*60, &batchStart, &batchEnd, &batchMetrics))
	}

	if len(requestList) != 0 {
		err = parallel.RunFirstErr(requestList...)
		if err != nil {
			return nil, err
		}
	}

	mergedMetrics := realTimeMetrics.Merge(batchMetrics)
	return &mergedMetrics, nil
}

func getAPIMetricsFunc(ctx *context.Context, api *context.API, period int64, startTime *time.Time, endTime *time.Time, apiMetrics *schema.APIMetrics) func() error {
	return func() error {
		metricDataResults, err := queryMetrics(ctx, api, period, startTime, endTime)
		if err != nil {
			return err
		}
		networkStats, err := extractNetworkMetrics(metricDataResults)
		if err != nil {
			return err
		}
		apiMetrics.NetworkStats = networkStats

		if api.Tracker != nil {
			if api.Tracker.ModelType == userconfig.ClassificationModelType {
				apiMetrics.ClassDistribution = extractClassificationMetrics(metricDataResults)
			} else {
				regressionStats, err := extractRegressionMetrics(metricDataResults)
				if err != nil {
					return err
				}
				apiMetrics.RegressionStats = regressionStats
			}
		}
		return nil
	}
}

func queryMetrics(ctx *context.Context, api *context.API, period int64, startTime *time.Time, endTime *time.Time) ([]*cloudwatch.MetricDataResult, error) {
	networkDataQueries := getNetworkStatsDef(ctx.App.Name, api, period)
	latencyMetrics := getLatencyMetricsDef(api.Path, period)
	allMetrics := append(latencyMetrics, networkDataQueries...)

	if api.Tracker != nil {
		if api.Tracker.ModelType == userconfig.ClassificationModelType {
			classMetrics, err := getClassesMetricDef(ctx, api, period)
			if err != nil {
				return nil, err
			}
			allMetrics = append(allMetrics, classMetrics...)
		} else {
			regressionMetrics := getRegressionMetricDef(ctx.App.Name, api, period)
			allMetrics = append(allMetrics, regressionMetrics...)
		}
	}

	metricsDataQuery := cloudwatch.GetMetricDataInput{
		EndTime:           endTime,
		StartTime:         startTime,
		MetricDataQueries: allMetrics,
	}
	output, err := config.AWS.CloudWatchMetrics.GetMetricData(&metricsDataQuery)
	if err != nil {
		return nil, err
	}
	return output.MetricDataResults, nil
}

func extractNetworkMetrics(metricsDataResults []*cloudwatch.MetricDataResult) (*schema.NetworkStats, error) {
	var networkStats schema.NetworkStats
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

func extractRegressionMetrics(metricsDataResults []*cloudwatch.MetricDataResult) (*schema.RegressionStats, error) {
	var regressionStats schema.RegressionStats
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

func getAPIDimensions(appName string, api *context.API) []*cloudwatch.Dimension {
	return []*cloudwatch.Dimension{
		{
			Name:  aws.String("AppName"),
			Value: aws.String(appName),
		},
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

func getLatencyMetricsDef(routeName string, period int64) []*cloudwatch.MetricDataQuery {
	networkDataQueries := []*cloudwatch.MetricDataQuery{
		{
			Id:    aws.String("latency"),
			Label: aws.String("Latency"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cortex.LogGroup),
					MetricName: aws.String("response-time.instance.cortex"),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("RequestPath"),
							Value: aws.String(routeName),
						},
					},
				},
				Stat:   aws.String("Average"),
				Period: aws.Int64(period),
			},
		},
		{
			Id:    aws.String("request_count"),
			Label: aws.String("RequestCount"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cortex.LogGroup),
					MetricName: aws.String("response-time.instance.cortex"),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("RequestPath"),
							Value: aws.String(routeName),
						},
					},
				},
				Stat:   aws.String("SampleCount"),
				Period: aws.Int64(period),
			},
		},
	}

	return networkDataQueries
}

func getRegressionMetricDef(appName string, api *context.API, period int64) []*cloudwatch.MetricDataQuery {
	metric := &cloudwatch.Metric{
		Namespace:  aws.String(config.Cortex.LogGroup),
		MetricName: aws.String("Prediction"),
		Dimensions: getAPIDimensions(appName, api),
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

func getNetworkStatsDef(appName string, api *context.API, period int64) []*cloudwatch.MetricDataQuery {
	dimensions := getAPIDimensions(appName, api)

	status200 := append(dimensions, &cloudwatch.Dimension{
		Name:  aws.String("Code"),
		Value: aws.String("2XX"),
	})
	status400 := append(dimensions, &cloudwatch.Dimension{
		Name:  aws.String("Code"),
		Value: aws.String("4XX"),
	})
	status500 := append(dimensions, &cloudwatch.Dimension{
		Name:  aws.String("Code"),
		Value: aws.String("5XX"),
	})

	networkDataQueries := []*cloudwatch.MetricDataQuery{
		{
			Id:    aws.String("datapoints_2XX"),
			Label: aws.String("2XX"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cortex.LogGroup),
					MetricName: aws.String("StatusCode"),
					Dimensions: status200,
				},
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
		},
		{
			Id:    aws.String("datapoints_4XX"),
			Label: aws.String("4XX"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cortex.LogGroup),
					MetricName: aws.String("StatusCode"),
					Dimensions: status400,
				},
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
		},
		{
			Id:    aws.String("datapoints_5XX"),
			Label: aws.String("5XX"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cortex.LogGroup),
					MetricName: aws.String("StatusCode"),
					Dimensions: status500,
				},
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
		},
	}

	return networkDataQueries
}

func getClassesMetricDef(ctx *context.Context, api *context.API, period int64) ([]*cloudwatch.MetricDataQuery, error) {
	prefix := filepath.Join(ctx.MetadataRoot, api.ID, "classes/")
	classes, err := config.AWS.ListPrefix(prefix, int64(consts.MaxClassesPerRequest))
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
					Namespace:  aws.String(config.Cortex.LogGroup),
					MetricName: aws.String("Prediction"),
					Dimensions: append(getAPIDimensions(ctx.App.Name, api), &cloudwatch.Dimension{
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
