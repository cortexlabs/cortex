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

package aws

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types/status"
)

func (c *Client) DoesLogGroupExist(logGroup string) (bool, error) {
	_, err := c.CloudWatchLogs().ListTagsLogGroup(&cloudwatchlogs.ListTagsLogGroupInput{
		LogGroupName: aws.String(logGroup),
	})
	if err != nil {
		if CheckErrCode(err, "ResourceNotFoundException") {
			return false, nil
		}
		return false, errors.Wrap(err, "log group "+logGroup)
	}

	return true, nil
}

func (c *Client) CreateLogGroup(logGroup string) error {
	_, err := c.CloudWatchLogs().CreateLogGroup(&cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(logGroup),
	})
	if err != nil {
		return errors.Wrap(err, "creating log group "+logGroup)
	}

	return nil
}

func (c *Client) TagLogGroup(logGroup string, tagMap map[string]string) error {
	tags := map[string]*string{}
	for key, value := range tagMap {
		tags[key] = aws.String(value)
	}

	_, err := c.CloudWatchLogs().TagLogGroup(&cloudwatchlogs.TagLogGroupInput{
		LogGroupName: aws.String(logGroup),
		Tags:         tags,
	})

	if err != nil {
		return errors.Wrap(err, "failed to add tags to log group", logGroup)

	}

	return nil
}

// UpdateDashboard updates existing dashboard by adding API name title, In flight request time, latency and request status metric
func (c *Client) UpdateDashboard(dashboardName string, dashboardRegion string, nameAPI string, apiID string) error {

	// get current dashboard form cloudwatch
	currDashboardOutput, err := c.CloudWatch().GetDashboard(&cloudwatch.GetDashboardInput{
		DashboardName: aws.String(dashboardName),
	})
	if err != nil {
		return errors.Wrap(err)
	}
	// get body string from GetDashboard return object
	currDashboardString := *currDashboardOutput.DashboardBody
	// get lowest element of cloudwatch. Needed to place metrics below all existing metrics
	highestY, err := getHighestYDashboard(currDashboardString)
	if err != nil {
		return errors.Wrap(err)
	}

	// create widgets for title and metrics
	apiTitleWidget := createTextWidget(1, highestY+1, 22, 1, "## API: "+nameAPI)
	inFlightWidget := createMetricWidget(1, highestY+2, 11, 6, inFlightMetric(dashboardName, nameAPI), "In flight requests", "Sum", dashboardRegion)
	latencyWidget := createMetricWidget(12, highestY+2, 11, 6, latencyMetric(dashboardName, nameAPI, apiID), "p99 of request response time", "p99", dashboardRegion)
	statCodeWidget := createMetricWidget(1, highestY+8, 11, 6, statCodeMetric(dashboardName, nameAPI, apiID), "Statuscode stats", "Sum", dashboardRegion)

	//define interface to unmarshal received body string
	var currDashboardHolder interface{}
	err = json.Unmarshal([]byte(currDashboardString), &currDashboardHolder)
	if err != nil {
		return errors.Wrap(err)
	}

	currDashboard := currDashboardHolder.(map[string]interface{})
	widgetSlice := currDashboard["widgets"].([]interface{})
	// append new API metrics widgets to existing widgets
	widgetSlice = append(widgetSlice, inFlightWidget, latencyWidget, statCodeWidget, apiTitleWidget)

	currDashboard["widgets"] = widgetSlice
	currDashboardJSON, err := json.Marshal(currDashboard)
	if err != nil {
		return errors.Wrap(err)
	}
	// upload updated dashboard to cloudwatch
	_, err = c.CloudWatch().PutDashboard(&cloudwatch.PutDashboardInput{
		DashboardName: aws.String(dashboardName),
		DashboardBody: aws.String(string(currDashboardJSON)),
	})
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}

// CreateDashboard updates existing dashboard by adding new widgets for new API
func (c *Client) CreateDashboard(dashboardName string) error {

	// delete Dashboard if already existing exists
	exists, err := c.DoesDashboardExist(dashboardName)
	if err != nil {
		return err
	}
	if exists {
		err = c.DeleteDashboard(dashboardName)
		if err != nil {
			return err
		}
	}

	//create cloudwatch base body with title
	cloudwatchBaseBody := map[string]interface{}{"start": "-PT1H", "periodOverride": "inherit", "widgets": []interface{}{createTextWidget(7, 0, 10, 1, "# CORTEX MONITORING DASHBOARD")}}
	cloudwatchBaseBodyJSON, err := json.Marshal(cloudwatchBaseBody)
	if err != nil {
		return errors.Wrap(err, "Failed to encode CLoudwatch Base Body into json")
	}
	_, err = c.CloudWatch().PutDashboard(&cloudwatch.PutDashboardInput{
		DashboardName: aws.String(dashboardName),
		DashboardBody: aws.String(string(cloudwatchBaseBodyJSON)),
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create Dashboard: "+dashboardName)
	}

	return nil
}

// DeleteDashboard deletes dashboard
func (c *Client) DeleteDashboard(dashboardName string) error {

	var toDelete []*string
	toDelete = append(toDelete, &dashboardName)
	_, err := c.CloudWatch().DeleteDashboards(&cloudwatch.DeleteDashboardsInput{
		DashboardNames: toDelete,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to delete dashboard: "+dashboardName)
	}
	return nil
}

// DoesDashboardExist will check if dashboard with same name as cluster already exists
func (c *Client) DoesDashboardExist(dashboardName string) (bool, error) {
	_, err := c.CloudWatch().GetDashboard(&cloudwatch.GetDashboardInput{
		DashboardName: aws.String(dashboardName),
	})
	if err != nil {
		if CheckErrCode(err, "ResourceNotFound") {
			return false, nil
		}
		return false, errors.Wrap(err, "dashboard: "+dashboardName)
	}

	return true, nil
}

// DeleteAPICloudwatch deletes api and reformats cloudwatch
func (c *Client) DeleteAPICloudwatch(statuses []status.Status, clusterName, apiName string) error {

	//delete old dashboard by creating a new base dashboard
	err := c.CreateDashboard(clusterName)
	if err != nil {
		return errors.Wrap(err, "Failed getting API statuses while deleting API")
	}

	// update dashboard for with all api execpt one to delete
	for _, stat := range statuses {
		if stat.APIName != apiName {
			c.UpdateDashboard(clusterName, c.Region, stat.APIName, stat.APIID)
		}

	}

	if err != nil {
		return errors.Wrap(err, "Failed to encode CLoudwatch Base Body into json")
	}

	return nil
}

// createTextWidget create new text widget with properties as parameter
// Example:
// title_widget = {
//     "type": "text",
//     "x": x,
//     "y": y,
//     "width": wewidthi,
//     "height": height,
//     "properties": {"markdown": markdown},
// }
func createTextWidget(x int, y int, width int, height int, markdown string) map[string]interface{} {

	return map[string]interface{}{"type": "text", "x": x, "y": y, "width": width, "height": height, "properties": map[string]string{"markdown": markdown}}

}

// createMetricWidget create new text widget with properties as parameter
// Example:
// metric_widget={
// 	"type":"metric",
// 	"x":0,
// 	"y":0,
// 	"width":12,
// 	"height":6,
// 	"properties":{
// 	   "metrics":[
// 		  [
// 			 "AWS/EC2",
// 			 "CPUUtilization",
// 			 "InstanceId",
// 			 "i-012345"
// 		  ]
// 	   ],
// 	   "period":300,
// 	   "stat":"Average",
// 	   "region":"us-east-1",
// 	   "title":"EC2 Instance CPU"
// 		}
//  }
func createMetricWidget(x int,
	y int,
	width int,
	height int,
	metric []interface{},
	title string,
	stat string,
	region string) map[string]interface{} {

	return map[string]interface{}{
		"type":   "metric",
		"x":      x,
		"y":      y,
		"width":  width,
		"height": height,
		"properties": map[string]interface{}{
			"metrics": metric,
			"period":  60,
			"title":   title,
			"stat":    stat,
			"region":  region,
			"view":    "timeSeries",
		}}

}

// inFlightMetric c
func inFlightMetric(dashboardName string, nameAPI string) []interface{} {
	var metric []interface{}
	metric = append(metric, dashboardName)
	metric = append(metric, "in-flight")
	metric = append(metric, "apiName")
	metric = append(metric, nameAPI)
	return []interface{}{metric}

}

func latencyMetric(dashboardName string, nameAPI string, apiID string) []interface{} {
	var metric []interface{}
	metric = append(metric, dashboardName)
	metric = append(metric, "Latency")
	metric = append(metric, "APIName")
	metric = append(metric, nameAPI)
	metric = append(metric, "metric_type")
	metric = append(metric, "histogram")
	metric = append(metric, "APIID")
	metric = append(metric, apiID)
	metric = append(metric, map[string]string{"id": "m1"})
	return []interface{}{metric}

}

func statCodeMetric(dashboardName string, nameAPI string, apiID string) []interface{} {

	var metric2XX []interface{}
	metric2XX = append(metric2XX, "...")
	metric2XX = append(metric2XX, "2XX")

	var metric3XX []interface{}
	metric3XX = append(metric3XX, "...")
	metric3XX = append(metric3XX, "3XX")

	var metric4XX []interface{}
	metric4XX = append(metric4XX, dashboardName)
	metric4XX = append(metric4XX, "StatusCode")
	metric4XX = append(metric4XX, "APIName")
	metric4XX = append(metric4XX, nameAPI)
	metric4XX = append(metric4XX, "metric_type")
	metric4XX = append(metric4XX, "counter")
	metric4XX = append(metric4XX, "APIID")
	metric4XX = append(metric4XX, apiID)
	metric4XX = append(metric4XX, "Code")
	metric4XX = append(metric4XX, "4XX")

	var metric5XX []interface{}
	metric5XX = append(metric5XX, "...")
	metric5XX = append(metric5XX, "5XX")

	return []interface{}{metric4XX, metric3XX, metric2XX, metric5XX}

}

// getHighestYDashboard takes dashboard string as input an gives back highest Y coordinate of a cloudwatch widget
func getHighestYDashboard(dash string) (int, error) {
	highestY := 0

	var currDash interface{}
	err := json.Unmarshal([]byte(dash), &currDash)
	if err != nil {
		return -1, err
	}
	currDashInter := currDash.(map[string]interface{})
	widgets := currDashInter["widgets"].([]interface{})

	for _, wid := range widgets {
		widInter := wid.(map[string]interface{})
		y := int(widInter["y"].(float64))
		if highestY < y {
			highestY = y
		}

	}

	return highestY, nil
}
