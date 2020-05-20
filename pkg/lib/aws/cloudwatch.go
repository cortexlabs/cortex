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

// CreateDashboard creates a new dashboard (or clears an existing one if it already exists)
func (c *Client) CreateDashboard(dashboardName string, title string) error {
	//create cloudwatch base body with title
	cloudwatchBaseBody := CloudWatchDashboard{
		Start:          "-PT1H",
		PeriodOverride: "inherit",
		Widgets: []CloudWatchWidget{
			TextWidget(7, 0, 10, 1, title),
		},
	}

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

// TextWidget creates new text widget with properties as parameter
// Example:
// title_widget = {
//     "type": "text",
//     "x": x,
//     "y": y,
//     "width": wewidthi,
//     "height": height,
//     "properties": {"markdown": markdown},
// }
func TextWidget(x int, y int, width int, height int, markdown string) CloudWatchWidget {
	return CloudWatchWidget{Type: "text", X: x, Y: y, Width: width, Height: height, Properties: map[string]interface{}{"markdown": markdown}}
}

// MetricWidget creates new text widget with properties as parameter
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
func MetricWidget(
	x int,
	y int,
	width int,
	height int,
	metric []interface{},
	title string,
	stat string,
	period int,
	region string,
) CloudWatchWidget {
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

// HighestY takes dashboard string as input an gives back highest Y coordinate of a cloudwatch widget
// highest Y coordinate corresponds to lowest widget
func HighestY(dash CloudWatchDashboard) (int, error) {
	highestY := 0

	for _, wid := range dash.Widgets {
		if highestY < wid.Y {
			highestY = wid.Y
		}
	}

	return highestY, nil
}
