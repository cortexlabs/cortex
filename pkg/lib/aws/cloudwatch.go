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

package aws

// import (
// 	// "fmt"
// 	// "strconv"
// 	"time"
// 	// "strings"

// 	"github.com/aws/aws-sdk-go/aws"
// 	"github.com/aws/aws-sdk-go/service/cloudwatch"
// 	"github.com/cortexlabs/cortex/pkg/lib/debug"
// )

// func (c *Client) GetMetrics(appName string, apiName string, apiID string) map[string]float64 {
// 	fmt.Println(fmt.Sprintf("APP_NAME: %s, API_NAME: %s", appName, apiName))

// 	listMetricsInput := &cloudwatch.ListMetricsInput{
// 		Namespace:  aws.String("cortex"),
// 		MetricName: aws.String("PREDICTION"),
// 		Dimensions: []*cloudwatch.DimensionFilter{
// 			&cloudwatch.DimensionFilter{
// 				Name:  aws.String("APP_NAME"),
// 				Value: aws.String(appName),
// 			},
// 			&cloudwatch.DimensionFilter{
// 				Name:  aws.String("API_NAME"),
// 				Value: aws.String(apiName),
// 			},
// 			&cloudwatch.DimensionFilter{
// 				Name:  aws.String("API_ID"),
// 				Value: aws.String(apiID),
// 			},
// 		},
// 	}

// 	metricsListOutput, _ := c.cloudWatchMetrics.ListMetrics(listMetricsInput)

// 	metricsList := metricsListOutput.Metrics
// 	for metricsListOutput.NextToken != nil {
// 		listMetricsInput.NextToken = metricsListOutput.NextToken
// 		metricsListOutput, _ := c.cloudWatchMetrics.ListMetrics(listMetricsInput)
// 		metricsList = append(metricsList, metricsListOutput.Metrics...)
// 	}
	
// 	numClasses := len(metricsList)
// 	debug.Pp(metricsList)
// 	metricDataQueryBatchCount := (numClasses / 100) + 1
// 	debug.Pp(metricDataQueryBatchCount)
// 	startTime := time.Now().Add(-5 * 60 * time.Second)
// 	endTime := time.Now()
// 	classMapping := make(map[string]float64, len(metricsList))

// 	for i := 0; i < metricDataQueryBatchCount; i++ {
// 		var batchSize int
// 		if i == metricDataQueryBatchCount - 1{
// 			batchSize = numClasses % 100
// 		} else {
// 			batchSize = 100
// 		}
// 		queries := make([]*cloudwatch.MetricDataQuery, batchSize)

// 		prevPage := i * 100
// 		for metricID := prevPage; metricID < prevPage + batchSize; metricID++ {
// 			queries[metricID-prevPage] = &cloudwatch.MetricDataQuery{
// 				Id: aws.String(fmt.Sprintf("id_%d", metricID)),
// 				MetricStat: &cloudwatch.MetricStat{
// 					Metric: metricsList[metricID],
// 					Period: aws.Int64(5 * 60),
// 					Stat:   aws.String("Sum"),
// 				},
// 			}
// 		}

// 		metricsDataQuery := cloudwatch.GetMetricDataInput{
// 			EndTime:           &endTime,
// 			StartTime:         &startTime,
// 			MetricDataQueries: queries,
// 		}

// 		output, _ := c.cloudWatchMetrics.GetMetricData(&metricsDataQuery)
// 		debug.Pp(output)
// 		for _, result := range output.MetricDataResults {
// 			idList := strings.Split(*result.Id, "_")
// 			id, _ := strconv.Atoi(idList[1])
// 			dims := metricsList[id].Dimensions
// 			var metricName string
// 			for _, dim := range dims {
// 				if *dim.Name == "CLASS" {
// 					metricName = *dim.Value
// 				}
// 			}
// 			if len(result.Values) == 0 {
// 				classMapping[metricName] = 0
// 			} else {
// 				classMapping[metricName] = *result.Values[0]
// 			}
// 		}
// 	}
// 	return classMapping
// }

// func (c *Client) GetMetrics(appName string, apiName string, apiID string) *PredictionMetrics {
// 	debug.Pp(apiID)
// 	var predictionMetrics PredictionMetrics
// 	startTime := time.Now().Add(-5 * 60 * time.Second)
// 	endTime := time.Now()
// 	input := cloudwatch.GetMetricStatisticsInput{
// 		EndTime:           &endTime,
// 		StartTime:         &startTime,
// 		Namespace:  aws.String("cortex"),
// 		MetricName: aws.String("PREDICTION"),
// 		Dimensions: []*cloudwatch.Dimension{
// 			&cloudwatch.Dimension{
// 				Name:  aws.String("APP_NAME"),
// 				Value: aws.String(appName),
// 			},
// 			&cloudwatch.Dimension{
// 				Name:  aws.String("API_NAME"),
// 				Value: aws.String(apiName),
// 			},
// 			&cloudwatch.Dimension{
// 				Name:  aws.String("API_ID"),
// 				Value: aws.String(apiID),
// 			},
// 		},
// 		Period: aws.Int64(5 * 60),
// 		// ExtendedStatistics: []*string{aws.String("p5.0"), aws.String("p25.0")},
// 		Statistics: []*string{aws.String("SampleCount"), aws.String("Minimum"), aws.String("Maximum"), aws.String("Average")},
// 	}

// 	predictionMetrics.startTime = &startTime
// 	predictionMetrics.endTime = &endTime

// 	response, _ := c.cloudWatchMetrics.GetMetricStatistics(&input)
// 	debug.Pp(response)
// 	metrics := map[string]float64{}
// 	metrics["SampleCount"] = *response.Datapoints[0].SampleCount
// 	metrics["Minimum"] = *response.Datapoints[0].Minimum
// 	metrics["Maximum"] = *response.Datapoints[0].Maximum
// 	metrics["Average"] = *response.Datapoints[0].Average


// 	if *response.Datapoints[0].Minimum > 0 {
// 		input.Statistics = nil
// 		input.ExtendedStatistics = []*string{aws.String("p25.0"), aws.String("p50.0"), aws.String("p75.0")}
// 		response, _ := c.cloudWatchMetrics.GetMetricStatistics(&input)
// 		for name, val := range response.Datapoints[0].ExtendedStatistics{
// 			metrics[name] = *val
// 		}
// 		debug.Pp(response)

// 	}
// 	return &PredictionMetrics
// }