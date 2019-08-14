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
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
)

const (
	maxClasses        = 50
	aggregationPeriod = int64(60 * 60) // 1 hour
)

func GetMetrics(w http.ResponseWriter, r *http.Request) {
	appName, err := getRequiredQueryParam("appName", r)
	if err != nil {
		RespondError(w, err)
		return
	}

	ctx := workloads.CurrentContext(appName)
	err = workloads.PopulateWorkloadIDs(ctx)
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
	twoWeeksAgo := endTime.Add(-14 * 24 * time.Hour)
	metricsStartTime := apiSavedStatus.Start
	var startTime *time.Time
	if twoWeeksAgo.Before(*metricsStartTime) {
		startTime = metricsStartTime
	} else {
		startTime = &twoWeeksAgo
	}

	metricDataResults, err := QueryMetrics(appName, api, aggregationPeriod, &endTime, startTime)
	if err != nil {
		RespondError(w, err)
	}

	var apiMetrics schema.APIMetrics

	err = ExtractNetworkMetrics(metricDataResults, &apiMetrics)
	if err != nil {
		RespondError(w, err)
		return
	}

	if api.Tracker != nil {
		if api.Tracker.ModelType == userconfig.RegressionModelType {
			err = ExtractRegressionMetrics(metricDataResults, &apiMetrics)
			if err != nil {
				RespondError(w, err)
				return
			}
		} else {
			ExtractClassificationMetrics(metricDataResults, &apiMetrics)
		}
	}

	Respond(w, apiMetrics)
}

func QueryMetrics(appName string, api *context.API, period int64, endTime *time.Time, startTime *time.Time) ([]*cloudwatch.MetricDataResult, error) {
	networkDataQueries := GetNetworkStatsDef(appName, api, period)
	latencyMetrics := GetLatencyMetricsDef(api.Path, period)
	allMetrics := append(latencyMetrics, networkDataQueries...)

	if api.Tracker != nil {
		if api.Tracker.ModelType == userconfig.ClassificationModelType {
			classMetrics, err := GetClassesMetricDef(appName, api, period)
			if err != nil {
				return nil, err
			}
			allMetrics = append(allMetrics, classMetrics...)
		} else {
			regressionMetrics := GetRegressionMetricDef(appName, api, period)
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

func ExtractNetworkMetrics(metricsDataResults []*cloudwatch.MetricDataResult, apiMetrics *schema.APIMetrics) error {
	var networkStats schema.NetworkStats
	var requestCount []*float64
	var latencyAvg []*float64

	for _, metricData := range metricsDataResults {
		if metricData.Values == nil {
			continue
		}

		switch {
		case *metricData.Label == "2XX":
			networkStats.Code2XX = *SumInt(metricData.Values...)
		case *metricData.Label == "4XX":
			networkStats.Code4XX = *SumInt(metricData.Values...)
		case *metricData.Label == "5XX":
			networkStats.Code5XX = *SumInt(metricData.Values...)
		case *metricData.Label == "Latency":
			latencyAvg = metricData.Values
		case *metricData.Label == "RequestCount":
			requestCount = metricData.Values
		}
	}

	avg, err := Avg(latencyAvg, requestCount)
	if err != nil {
		return err
	}
	networkStats.Latency = avg
	apiMetrics.NetworkStats = &networkStats
	return nil
}

func ExtractClassificationMetrics(metricsDataResults []*cloudwatch.MetricDataResult, apiMetrics *schema.APIMetrics) {
	apiMetrics.ClassDistribution = map[string]int{}
	for _, metricData := range metricsDataResults {
		if metricData.Values == nil {
			continue
		}

		if strings.HasPrefix(*metricData.Label, "class_") {
			className := (*metricData.Label)[6:]
			apiMetrics.ClassDistribution[className] = *SumInt(metricData.Values...)
		}
	}
}

func ExtractRegressionMetrics(metricsDataResults []*cloudwatch.MetricDataResult, apiMetrics *schema.APIMetrics) error {
	var regressionStats schema.RegressionStats
	var predictionAvg []*float64
	var requestCount []*float64

	for _, metricData := range metricsDataResults {
		if metricData.Values == nil {
			continue
		}

		switch {
		case *metricData.Label == "Min":
			min := Min(metricData.Values...)
			if min != nil {
				regressionStats.Min = *min
			}
		case *metricData.Label == "Max":
			max := Max(metricData.Values...)
			if max != nil {
				regressionStats.Max = *max
			}
		case *metricData.Label == "RequestCount":
			count := SumInt(metricData.Values...)
			if count != nil {
				regressionStats.SampleCount = *count
			}
			requestCount = metricData.Values
		case *metricData.Label == "Avg":
			predictionAvg = metricData.Values
		}
	}

	avg, err := Avg(predictionAvg, requestCount)
	if err != nil {
		return err
	}
	if avg != nil {
		regressionStats.Avg = *avg
	}
	if regressionStats.SampleCount > 0 {
		apiMetrics.RegressionStats = &regressionStats
	}
	return nil
}

func GetAPIDimensions(appName string, api *context.API) []*cloudwatch.Dimension {
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

func GetLatencyMetricsDef(routeName string, period int64) []*cloudwatch.MetricDataQuery {
	networkDataQueries := []*cloudwatch.MetricDataQuery{
		{
			Id:    aws.String("latency"),
			Label: aws.String("Latency"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cortex.Namespace),
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
					Namespace:  aws.String(config.Cortex.Namespace),
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

func GetRegressionMetricDef(appName string, api *context.API, period int64) []*cloudwatch.MetricDataQuery {
	metric := &cloudwatch.Metric{
		Namespace:  aws.String(config.Cortex.Namespace),
		MetricName: aws.String("Prediction"),
		Dimensions: GetAPIDimensions(appName, api),
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
			Id:    aws.String("avg"),
			Label: aws.String("Avg"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: metric,
				Period: aws.Int64(period),
				Stat:   aws.String("Average"),
			},
		},
	}

	return regressionMetric
}

func GetNetworkStatsDef(appName string, api *context.API, period int64) []*cloudwatch.MetricDataQuery {
	dimensions := GetAPIDimensions(appName, api)

	status200 := append([]*cloudwatch.Dimension{}, dimensions...)
	status200 = append(status200, &cloudwatch.Dimension{
		Name:  aws.String("Code"),
		Value: aws.String("2XX"),
	})
	status400 := append([]*cloudwatch.Dimension{}, dimensions...)
	status400 = append(status400, &cloudwatch.Dimension{
		Name:  aws.String("Code"),
		Value: aws.String("4XX"),
	})
	status500 := append([]*cloudwatch.Dimension{}, dimensions...)
	status500 = append(status500, &cloudwatch.Dimension{
		Name:  aws.String("Code"),
		Value: aws.String("5XX"),
	})

	networkDataQueries := []*cloudwatch.MetricDataQuery{
		{
			Id:    aws.String("datapoints_2XX"),
			Label: aws.String("2XX"),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String(config.Cortex.Namespace),
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
					Namespace:  aws.String(config.Cortex.Namespace),
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
					Namespace:  aws.String(config.Cortex.Namespace),
					MetricName: aws.String("StatusCode"),
					Dimensions: status500,
				},
				Period: aws.Int64(period),
				Stat:   aws.String("Sum"),
			},
		},
	}

	return networkDataQueries
}

func GetClassesMetricDef(appName string, api *context.API, period int64) ([]*cloudwatch.MetricDataQuery, error) {
	classMetricQueries := []*cloudwatch.MetricDataQuery{}
	listMetricsInput := &cloudwatch.ListMetricsInput{
		Namespace:  aws.String(config.Cortex.Namespace),
		MetricName: aws.String("Prediction"),
		Dimensions: []*cloudwatch.DimensionFilter{
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
		},
	}

	listMetricsOutput, err := config.AWS.CloudWatchMetrics.ListMetrics(listMetricsInput)
	if err != nil {
		return nil, err
	}

	if listMetricsOutput.Metrics == nil {
		return classMetricQueries, nil
	}

	classCount := 0
	for i, metric := range listMetricsOutput.Metrics {
		if classCount >= maxClasses {
			break
		}

		var className string
		for _, dim := range metric.Dimensions {
			if *dim.Name == "Class" {
				className = *dim.Value
			}
		}

		if len(className) == 0 {
			continue
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
		classCount++
	}
	return classMetricQueries, nil
}
