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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type CloudWatchDashboard struct {
	Start          string             `json:"start"`
	PeriodOverride string             `json:"periodOverride"`
	Widgets        []CloudWatchWidget `json:"widgets"`
}

type CloudWatchWidget struct {
	Type       string                 `json:"type"`
	X          int                    `json:"x"`
	Y          int                    `json:"y"`
	Width      int                    `json:"width"`
	Height     int                    `json:"height"`
	Properties map[string]interface{} `json:"properties"`
}

func (c *Client) DoesLogGroupExist(logGroup string) (bool, error) {
	_, err := c.CloudWatchLogs().ListTagsLogGroup(&cloudwatchlogs.ListTagsLogGroupInput{
		LogGroupName: aws.String(logGroup),
	})
	if err != nil {
		if IsErrCode(err, "ResourceNotFoundException") {
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

// AddAPIToDashboard updates existing dashboard by adding API name title, In flight request time, latency and request status metric
func (c *Client) AddAPIToDashboard(dashboardName string, nameAPI string) error {

	// get current dashboard form cloudwatch
	currDashboardOutput, err := c.CloudWatch().GetDashboard(&cloudwatch.GetDashboardInput{
		DashboardName: aws.String(dashboardName),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	// get body string from GetDashboard return object
	currDashboardString := *currDashboardOutput.DashboardBody

	//define interface to unmarshal received body string
	var currDashboard CloudWatchDashboard
	err = json.Unmarshal([]byte(currDashboardString), &currDashboard)
	if err != nil {
		return errors.WithStack(err)
	}

	// get lowest element of cloudwatch. Needed to place metrics below all existing metrics
	highestY, err := getHighestYDashboard(currDashboard)
	if err != nil {
		return err
	}

	// create widgets for title and metrics
	apiTitleWidget := createTextWidget(1, highestY+1, 22, 1, "## API: "+nameAPI)
	// top left widget
	statCodeWidget := createMetricWidget(1, highestY+2, 11, 6, statCodeMetric(dashboardName, nameAPI), "Status Code", "Sum", 60, c.Region)
	// top right widget
	inFlightWidget := createMetricWidget(12, highestY+2, 11, 6, inFlightMetric(dashboardName, nameAPI), "In flight requests", "Sum", 10, c.Region)
	// bottem left widget
	latencyWidgetp50 := createMetricWidget(1, highestY+8, 11, 6, latencyMetric(dashboardName, nameAPI), "median request response time (60s)", "p50", 60, c.Region)
	// bottom right widget
	latencyWidgetp99 := createMetricWidget(12, highestY+8, 11, 6, latencyMetric(dashboardName, nameAPI), "p99 of request response time (60s)", "p99", 60, c.Region)

	// append new API metrics widgets to existing widgets
	currDashboard.Widgets = append(currDashboard.Widgets, apiTitleWidget, statCodeWidget, inFlightWidget, latencyWidgetp50, latencyWidgetp99)
	currDashboardJSON, err := json.Marshal(currDashboard)
	if err != nil {
		return errors.WithStack(err)
	}
	// upload updated dashboard to cloudwatch
	_, err = c.CloudWatch().PutDashboard(&cloudwatch.PutDashboardInput{
		DashboardName: aws.String(dashboardName),
		DashboardBody: aws.String(string(currDashboardJSON)),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// CreateDashboard creates a new dashboard (or clears an existing one if it already exists)
func (c *Client) CreateDashboard(dashboardName string) error {

	//create cloudwatch base body with title
	cloudwatchBaseBody := CloudWatchDashboard{Start: "-PT1H", PeriodOverride: "inherit", Widgets: []CloudWatchWidget{createTextWidget(7, 0, 10, 1, "# CORTEX MONITORING DASHBOARD")}}
	cloudwatchBaseBodyJSON, err := json.Marshal(cloudwatchBaseBody)
	if err != nil {
		return errors.Wrap(err, "failed to encode cloudwatch base body into json")
	}
	_, err = c.CloudWatch().PutDashboard(&cloudwatch.PutDashboardInput{
		DashboardName: aws.String(dashboardName),
		DashboardBody: aws.String(string(cloudwatchBaseBodyJSON)),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create dashboard", dashboardName)
	}

	return nil
}

// DeleteDashboard deletes dashboard
func (c *Client) DeleteDashboard(dashboardName string) error {

	_, err := c.CloudWatch().DeleteDashboards(&cloudwatch.DeleteDashboardsInput{
		DashboardNames: []*string{aws.String(dashboardName)},
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete dashboard", dashboardName)
	}
	return nil
}

// DoesDashboardExist will check if dashboard with same name as cluster already exists
func (c *Client) DoesDashboardExist(dashboardName string) (bool, error) {
	_, err := c.CloudWatch().GetDashboard(&cloudwatch.GetDashboardInput{
		DashboardName: aws.String(dashboardName),
	})
	if err != nil {
		if IsErrCode(err, "ResourceNotFound") {
			return false, nil
		}
		return false, errors.Wrap(err, "dashboard", dashboardName)
	}

	return true, nil
}

// DeleteAPICloudwatch deletes api and reformats cloudwatch
func (c *Client) DeleteAPICloudwatch(allAPINames []string, clusterName, apiName string) error {

	//delete old dashboard by creating a new base dashboard
	err := c.CreateDashboard(clusterName)
	if err != nil {
		return errors.WithStack(err)
	}

	// update dashboard by adding all APIs except the one to delete
	for _, allAPIname := range allAPINames {
		if allAPIname != apiName {
			err = c.AddAPIToDashboard(clusterName, allAPIname)
		}
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to add API \"%s\" to cloudwatch dashboard", apiName))
		}

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
func createTextWidget(x int, y int, width int, height int, markdown string) CloudWatchWidget {

	return CloudWatchWidget{Type: "text", X: x, Y: y, Width: width, Height: height, Properties: map[string]interface{}{"markdown": markdown}}
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
	period int,
	region string) CloudWatchWidget {

	return CloudWatchWidget{
		Type:   "metric",
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		Properties: map[string]interface{}{
			"metrics": metric,
			"period":  period,
			"title":   title,
			"stat":    stat,
			"region":  region,
			"view":    "timeSeries",
		}}
}

func inFlightMetric(dashboardName string, nameAPI string) []interface{} {
	var metric []interface{}
	metric = append(metric, dashboardName)
	metric = append(metric, "in-flight")
	metric = append(metric, "apiName")
	metric = append(metric, nameAPI)

	return []interface{}{metric}
}

func latencyMetric(dashboardName string, nameAPI string) []interface{} {
	var metric []interface{}
	metric = append(metric, dashboardName)
	metric = append(metric, "Latency")
	metric = append(metric, "APIName")
	metric = append(metric, nameAPI)
	metric = append(metric, "metric_type")
	metric = append(metric, "histogram")

	return []interface{}{metric}
}

func statCodeMetric(dashboardName string, nameAPI string) []interface{} {
	var metric2XX []interface{}
	metric2XX = append(metric2XX, dashboardName)
	metric2XX = append(metric2XX, "StatusCode")
	metric2XX = append(metric2XX, "APIName")
	metric2XX = append(metric2XX, nameAPI)
	metric2XX = append(metric2XX, "metric_type")
	metric2XX = append(metric2XX, "counter")
	metric2XX = append(metric2XX, "Code")
	metric2XX = append(metric2XX, "2XX")

	var metric4XX []interface{}
	metric4XX = append(metric4XX, "...")
	metric4XX = append(metric4XX, "4XX")

	var metric5XX []interface{}
	metric5XX = append(metric5XX, "...")
	metric5XX = append(metric5XX, "5XX")

	return []interface{}{metric2XX, metric4XX, metric5XX}
}

// getHighestYDashboard takes dashboard string as input an gives back highest Y coordinate of a cloudwatch widget
// highest Y coordinate corresponds to lowest widget
func getHighestYDashboard(dash CloudWatchDashboard) (int, error) {
	highestY := 0

	for _, wid := range dash.Widgets {
		if highestY < wid.Y {
			highestY = wid.Y
		}
	}

	return highestY, nil
}
