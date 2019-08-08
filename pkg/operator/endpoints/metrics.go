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

package endpoints

import (
	"fmt"
	"strings"
	// "strconv"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
)

func GetMetrics(w http.ResponseWriter, r *http.Request) {
	appName, err := getRequiredQueryParam("appName", r)
	if err != nil {
		RespondError(w, err)
		return
	}

	ctx := workloads.CurrentContext(appName)
	err := workloads.PopulateWorkloadIDs(ctx)
	if err != nil {
		RespondError(w, err)
		return
	}

	apiName, err := getRequiredQueryParam("apiName", r)
	if err != nil {
		RespondError(w, err)
		return
	}

	api := ctx.APIs[apiName]
	apiSavedStatus, err := workloads.GetAPISavedStatus(api.ID, api.WorkloadID, appName)
	if err != nil {
		RespondError(w, err)
		return
	}

	endTime := time.Now()
	twoWeeksAgo := endTime.Add(- 14 * 24 * time.Hour)
	metricsStartTime := apiSavedStatus.Start
	if twoWeeksAgo

	period := int64(60 * 60)

	if *ctx.APIs[apiName].Tracker.ModelType == "classification" {
		metrics := GetClassificationMetrics(appName, apiName, ctx.APIs[apiName].ID, period, &endTime)
		Respond(w, metrics)
	} else {
		metrics := GetRegressionMetrics(appName, apiName, ctx.APIs[apiName].ID, period, &endTime)
		Respond(w, metrics)
	}
}

func SumInt(floats ...*float64) *int {
	sum := 0
	for _, num := range floats {
		sum += int(*num)
	}
	return &sum
}

func SumFloat(floats ...*float64) *float64 {
	sum := 0.0
	for _, num := range floats {
		sum += *num
	}
	return &sum
}

func Min(floats ...*float64) *float64{
	if len(floats) == 0 {
		return nil
	}

	min := *floats[0]
	for _, num := range floats {
		if min > *num {
			min = *num
		}
	}
	return &min
}

func Max(floats ...*float64) *float64 {
	if len(floats) == 0 {
		return nil
	}

	max := *floats[0]
	for _, num := range floats {
		if max < *num {
			max = *num
		}
	}
	return &max
}

func Avg(values []*float64, counts []*float64) *float64 {
	if len(values) == 0 || len(counts) == 0 {
		return nil
	}

	total := float64(*SumInt(counts...))
	avg := 0.0
	for idx, valPtr := range values {
		weight := *counts[idx]
		value := *valPtr
		if weight != 0 {
			avg += value * weight / total
		}
	}

	return &avg
}

func GetClassificationMetrics(appName string, apiName string, apiID string, period int64, endTime *time.Time) *schema.APIMetrics {
	startTime := endTime.Add(-7 * 24 * 60 * 60 * time.Second)

	networkDataQueries := GetNetworkStatsDef(appName, apiName, apiID, period)
	classMetrics, _ := GetClassesMetricDef(appName, apiName, apiID, period)

	metricsDataQuery := cloudwatch.GetMetricDataInput{
		EndTime:           endTime,
		StartTime:         &startTime,
		MetricDataQueries: append(networkDataQueries, classMetrics...),
	}

	output, err := config.AWS.CloudWatchMetrics.GetMetricData(&metricsDataQuery)
	debug.Pp(output)
	debug.Pp(err)

	var apiMetrics schema.APIMetrics
	classDistribution := map[string]int{}
	var statusCodes  schema.StatusCodes
	var networkStats schema.NetworkStats
	for _, metricData := range output.MetricDataResults {
		if metricData.Values == nil {
			continue
		}

		switch {
		case *metricData.Label == "2XX":
			statusCodes.Code2XX = *SumInt(metricData.Values...)
		case *metricData.Label == "4XX":
			statusCodes.Code4XX = *SumInt(metricData.Values...)
		case *metricData.Label == "5XX":
			statusCodes.Code5XX = *SumInt(metricData.Values...)
		case strings.HasPrefix(*metricData.Label, "class_"):
			className := (*metricData.Label)[6:]
			classDistribution[className] = *SumInt(metricData.Values...)
		}	
	}
	networkStats.StatusCodes = &statusCodes
	apiMetrics.NetworkStats = &networkStats
	apiMetrics.ClassDistribution = classDistribution
	return &apiMetrics
}

func GetLatencyMetricsDef(routeName string, period int64, endTime *time.Time) {
	
}

func GetRegressionMetrics(appName string, apiName string, apiID string, period int64, endTime *time.Time) *schema.APIMetrics {
	startTime := endTime.Add(-7 * 24 * 60 * 60 * time.Second)

	networkDataQueries := GetNetworkStatsDef(appName, apiName, apiID, period)
	regressionMetricQueries := GetRegressionMetricDef(appName, apiName, apiID, period)


	metricsDataQuery := cloudwatch.GetMetricDataInput{
		EndTime:           endTime,
		StartTime:         &startTime,
		MetricDataQueries: append(networkDataQueries, regressionMetricQueries...),
	}

	output, _ := config.AWS.CloudWatchMetrics.GetMetricData(&metricsDataQuery)

	var apiMetrics schema.APIMetrics
	var regressionStats schema.RegressionStats
	var statusCodes  schema.StatusCodes
	var networkStats schema.NetworkStats
	var sampleCount []*float64
	var avgValues []*float64

	for _, metricData := range output.MetricDataResults {
		if metricData.Values == nil {
			continue
		}

		switch {
		case *metricData.Label == "2XX":
			statusCodes.Code2XX = *SumInt(metricData.Values...)
		case *metricData.Label == "4XX":
			statusCodes.Code4XX = *SumInt(metricData.Values...)
		case *metricData.Label == "5XX":
			statusCodes.Code5XX = *SumInt(metricData.Values...)
		case *metricData.Label == "min":
			regressionStats.Min = Min(metricData.Values...)
		case *metricData.Label == "max":
			regressionStats.Max = Max(metricData.Values...)
		case *metricData.Label == "count":
			regressionStats.SampleCount = SumInt(metricData.Values...)
			sampleCount = metricData.Values
		case *metricData.Label == "avg":
			avgValues = metricData.Values
		}
	}

	regressionStats.Avg = Avg(avgValues, sampleCount)

	networkStats.StatusCodes = &statusCodes
	apiMetrics.NetworkStats = &networkStats
	apiMetrics.RegressionStats = &regressionStats
	return &apiMetrics
}

func GetRegressionMetricDef(appName string, apiName string, apiID string, period int64) []*cloudwatch.MetricDataQuery {
	metric := &cloudwatch.Metric{
		Namespace:  aws.String("cortex"),
		MetricName: aws.String("Prediction"),
		Dimensions: []*cloudwatch.Dimension{
			&cloudwatch.Dimension{
				Name:  aws.String("AppName"),
				Value: aws.String(appName),
			},
			&cloudwatch.Dimension{
				Name:  aws.String("APIName"),
				Value: aws.String(apiName),
			},
			&cloudwatch.Dimension{
				Name:  aws.String("APIID"),
				Value: aws.String(apiID),
			},
		},
	}
	regressionMetric := []*cloudwatch.MetricDataQuery{
		&cloudwatch.MetricDataQuery{
			Id:    aws.String("min"),
			Label: aws.String("min"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: metric,
				Stat:   aws.String("Minimum"),
				Period: aws.Int64(period),
			},
		},
		&cloudwatch.MetricDataQuery{
			Id:    aws.String("max"),
			Label: aws.String("max"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: metric,
				Stat:   aws.String("Maximum"),
				Period: aws.Int64(period),
			},
		},
		&cloudwatch.MetricDataQuery{
			Id:    aws.String("avg"),
			Label: aws.String("avg"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: metric,
				Period: aws.Int64(period),
				Stat:   aws.String("Average"),
			},
		},
		&cloudwatch.MetricDataQuery{
			Id:    aws.String("count"),
			Label: aws.String("count"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: metric,
				Period: aws.Int64(period),
				Stat:   aws.String("SampleCount"),
			},
		},
	}

	return regressionMetric
}

func GetNetworkStatsDef(appName string, apiName string, apiID string, period int64) []*cloudwatch.MetricDataQuery {
	dimensions := []*cloudwatch.Dimension{
		&cloudwatch.Dimension{
			Name:  aws.String("AppName"),
			Value: aws.String(appName),
		},
		&cloudwatch.Dimension{
			Name:  aws.String("APIName"),
			Value: aws.String(apiName),
		},
		&cloudwatch.Dimension{
			Name:  aws.String("APIID"),
			Value: aws.String(apiID),
		},
	}

	status_200 := append([]*cloudwatch.Dimension{}, dimensions...)
	status_200 = append(status_200, &cloudwatch.Dimension{
		Name:  aws.String("Code"),
		Value: aws.String("2XX"),
	})
	status_400 := append([]*cloudwatch.Dimension{}, dimensions...)
	status_400 = append(status_400, &cloudwatch.Dimension{
		Name:  aws.String("Code"),
		Value: aws.String("4XX"),
	})
	status_500 := append([]*cloudwatch.Dimension{}, dimensions...)
	status_500 = append(status_500, &cloudwatch.Dimension{
		Name:  aws.String("Code"),
		Value: aws.String("5XX"),
	})

	networkDataQueries := []*cloudwatch.MetricDataQuery{
		&cloudwatch.MetricDataQuery{
			Id:    aws.String("datapoints_2XX"),
			Label: aws.String("2XX"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String("cortex"),
					MetricName: aws.String("StatusCode"),
					Dimensions: status_200,
				},
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
		},
		&cloudwatch.MetricDataQuery{
			Id:    aws.String("datapoints_4XX"),
			Label: aws.String("4XX"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String("cortex"),
					MetricName: aws.String("StatusCode"),
					Dimensions: status_400,
				},
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
		},
		&cloudwatch.MetricDataQuery{
			Id:    aws.String("datapoints_5XX"),
			Label: aws.String("5XX"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String("cortex"),
					MetricName: aws.String("StatusCode"),
					Dimensions: status_500,
				},
				Period: aws.Int64(period),
				Stat:   aws.String("Sum"),
			},
		},
	}

	return networkDataQueries
}

func GetClassesMetricDef(appName string, apiName string, apiID string, period int64) ([]*cloudwatch.MetricDataQuery, error) {
	maxClasses := 20
	classMetricQueries := []*cloudwatch.MetricDataQuery{}
	listMetricsInput := &cloudwatch.ListMetricsInput{
		Namespace:  aws.String("cortex"),
		MetricName: aws.String("Prediction"),
		Dimensions: []*cloudwatch.DimensionFilter{
			&cloudwatch.DimensionFilter{
				Name:  aws.String("AppName"),
				Value: aws.String(appName),
			},
			&cloudwatch.DimensionFilter{
				Name:  aws.String("APIName"),
				Value: aws.String(apiName),
			},
			&cloudwatch.DimensionFilter{
				Name:  aws.String("APIID"),
				Value: aws.String(apiID),
			},
		},
	}

	listMetricsOutput, err := config.AWS.CloudWatchMetrics.ListMetrics(listMetricsInput)
	debug.Pp(listMetricsOutput)
	debug.Pp(err)

	if err != nil {
		return nil, err
	}
	if listMetricsOutput.Metrics == nil {
		return classMetricQueries, nil
	}
	for i, metric := range listMetricsOutput.Metrics {
		if i >= maxClasses {
			break
		}

		var className string
		for _, dim := range metric.Dimensions {
			if *dim.Name == "Class" {
				className = *dim.Value
			}
		}

		classMetricQueries = append(classMetricQueries, &cloudwatch.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("id_%d", i)),
			MetricStat: &cloudwatch.MetricStat{
				Metric: metric,
				Stat:   aws.String("Sum"),
				Period: aws.Int64(period),
			},
			Label: aws.String("class_" + className),
		})
	}
	debug.Pp(classMetricQueries)
	return classMetricQueries, nil
}