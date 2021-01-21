/*
Copyright 2021 Cortex Labs, Inc.

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

var (
	_dashboardMinWidthUnits  = 1
	_dashboardMaxWidthUnits  = 24
	_dashboardMinHeightUnits = 1
	_dashboardMaxHeightUnits = 1000
)

type CloudWatchDashboard struct {
	Start          string             `json:"start"`
	PeriodOverride string             `json:"periodOverride"`
	Widgets        []CloudWatchWidget `json:"widgets"`
}

// Example:
// CloudWatchWidget{
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
// 	}
//  }
type CloudWatchWidget struct {
	Type       string                 `json:"type"`
	X          int                    `json:"x"`
	Y          int                    `json:"y"`
	Width      int                    `json:"width"`
	Height     int                    `json:"height"`
	Properties map[string]interface{} `json:"properties"`
}

type CloudWatchWidgetGrid struct {
	XOrigin      int                `json:"x_origin"`
	YOrigin      int                `json:"y_origin"`
	NumColumns   int                `json:"num_columns"`
	NumRows      int                `json:"num_rows"`
	WidgetHeight int                `json:"widget_height"`
	WidgetWidth  int                `json:"widget_width"`
	Widgets      []CloudWatchWidget `json:"widgets"`
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

func (c *Client) CreateLogGroup(logGroup string, tags map[string]string) error {
	_, err := c.CloudWatchLogs().CreateLogGroup(&cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(logGroup),
		Tags:         aws.StringMap(tags),
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

// NewDashboard creates a new dashboard object with title
func (c *Client) NewDashboard(title string) *CloudWatchDashboard {
	return &CloudWatchDashboard{
		Start:          "-PT1H",
		PeriodOverride: "inherit",
		Widgets: []CloudWatchWidget{
			TextWidget(7, 0, 10, 1, title),
		},
	}
}

// GetDashboard gets a dashboard from cloudwatch
func (c *Client) GetDashboard(dashboardName string) (*CloudWatchDashboard, error) {
	dashboardOutput, err := c.CloudWatch().GetDashboard(&cloudwatch.GetDashboardInput{
		DashboardName: aws.String(dashboardName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dashboard", dashboardName)
	}

	dashboardString := *dashboardOutput.DashboardBody

	var dashboard CloudWatchDashboard
	err = json.Unmarshal([]byte(dashboardString), &dashboard)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode cloudwatch body json")
	}

	return &dashboard, nil
}

// GetDashboardOrEmpty gets a dashboard if it exists, or initializes an empty one if not
func (c *Client) GetDashboardOrEmpty(dashboardName string, title string) (*CloudWatchDashboard, error) {
	dashboard, err := c.GetDashboard(dashboardName)
	if err != nil {
		if IsErrCode(err, "ResourceNotFound") {
			dashboard = c.NewDashboard(title)
		} else {
			return nil, err
		}
	}

	return dashboard, nil
}

// CreateDashboard creates a new dashboard (or clears an existing one if it already exists)
func (c *Client) CreateDashboard(dashboardName string, title string) error {
	dashboard := c.NewDashboard(title)

	err := c.PutDashboard(dashboard, dashboardName)
	if err != nil {
		return err
	}

	return nil
}

// PutDashboard updates a dashboard (or creates it if it doesn't exit)
func (c *Client) PutDashboard(dashboard *CloudWatchDashboard, dashboardName string) error {
	dashboardJSON, err := json.Marshal(dashboard)
	if err != nil {
		return errors.Wrap(err, "failed to encode cloudwatch body into json")
	}

	_, err = c.CloudWatch().PutDashboard(&cloudwatch.PutDashboardInput{
		DashboardName: aws.String(dashboardName),
		DashboardBody: aws.String(string(dashboardJSON)),
	})
	if err != nil {
		return errors.Wrap(err, "failed to put dashboard", dashboardName)
	}

	return nil
}

// DeleteDashboard deletes a dashboard
func (c *Client) DeleteDashboard(dashboardName string) error {
	_, err := c.CloudWatch().DeleteDashboards(&cloudwatch.DeleteDashboardsInput{
		DashboardNames: []*string{aws.String(dashboardName)},
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete dashboard", dashboardName)
	}

	return nil
}

// DoesDashboardExist checks if a dashboard exists
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

// TextWidget creates new text widget
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

// NewHorizontalGrid sets a CloudWatch Dashboard grid to be filled from left to right, row by row
func NewHorizontalGrid(xOrigin, yOrigin, widgetHeight, widgetWidth, numColumns int) (*CloudWatchWidgetGrid, error) {
	if widgetHeight < 1 || widgetHeight > _dashboardMaxHeightUnits {
		return &CloudWatchWidgetGrid{}, ErrorDashboardHeightOutOfRange(widgetHeight)
	}
	if widgetWidth < 1 || widgetWidth > _dashboardMaxWidthUnits {
		return &CloudWatchWidgetGrid{}, ErrorDashboardWidthOutOfRange(widgetWidth)
	}
	if xOrigin+numColumns*widgetWidth > _dashboardMaxWidthUnits {
		return &CloudWatchWidgetGrid{}, ErrorDashboardWidthOutOfRange(xOrigin + numColumns*widgetWidth)
	}
	return &CloudWatchWidgetGrid{
		XOrigin:      xOrigin,
		YOrigin:      yOrigin,
		WidgetHeight: widgetHeight,
		WidgetWidth:  widgetWidth,
		NumColumns:   numColumns,
		Widgets:      make([]CloudWatchWidget, 0),
	}, nil
}

// NewVerticalGrid sets a CloudWatch Dashboard grid to be filled from top to bottom, column by column
func NewVerticalGrid(xOrigin, yOrigin, widgetHeight, widgetWidth, numRows int) (*CloudWatchWidgetGrid, error) {
	if widgetHeight < 1 || widgetHeight > _dashboardMaxHeightUnits {
		return &CloudWatchWidgetGrid{}, ErrorDashboardHeightOutOfRange(widgetHeight)
	}
	if widgetWidth < 1 || widgetWidth > _dashboardMaxWidthUnits {
		return &CloudWatchWidgetGrid{}, ErrorDashboardWidthOutOfRange(widgetWidth)
	}
	if yOrigin+numRows*widgetHeight > _dashboardMaxHeightUnits {
		return &CloudWatchWidgetGrid{}, ErrorDashboardHeightOutOfRange(yOrigin + numRows*widgetHeight)
	}
	return &CloudWatchWidgetGrid{
		XOrigin:      xOrigin,
		YOrigin:      yOrigin,
		WidgetHeight: widgetHeight,
		WidgetWidth:  widgetWidth,
		NumRows:      numRows,
		Widgets:      make([]CloudWatchWidget, 0),
	}, nil
}

// AddWidget adds a widget to the configured grid
func (grid *CloudWatchWidgetGrid) AddWidget(
	metric []interface{},
	title string,
	stat string,
	period int,
	region string,
) error {
	var currentColumn, currentRow int
	if grid.NumColumns > 0 {
		currentRow = len(grid.Widgets) / grid.NumColumns
		currentColumn = len(grid.Widgets) - currentRow*grid.NumColumns
	}
	if grid.NumRows > 0 {
		currentColumn = len(grid.Widgets) / grid.NumRows
		currentRow = len(grid.Widgets) - currentColumn*grid.NumRows
	}
	x := grid.XOrigin + currentColumn*grid.WidgetWidth
	y := grid.YOrigin + currentRow*grid.WidgetHeight

	if x+grid.WidgetWidth > _dashboardMaxWidthUnits {
		return ErrorDashboardWidthOutOfRange(x + grid.WidgetWidth)
	}
	if y+grid.WidgetHeight > _dashboardMaxHeightUnits {
		return ErrorDashboardHeightOutOfRange(y + grid.WidgetHeight)
	}

	grid.Widgets = append(grid.Widgets,
		CloudWatchWidget{
			Type:   "metric",
			X:      x,
			Y:      y,
			Width:  grid.WidgetWidth,
			Height: grid.WidgetHeight,
			Properties: map[string]interface{}{
				"metrics": metric,
				"title":   title,
				"stat":    stat,
				"period":  period,
				"region":  region,
				"view":    "timeSeries",
			},
		},
	)

	return nil
}

// HighestY returns the largest Y coordinate of a widget on the dashboard (i.e. the lowest widget)
func HighestY(dashboard *CloudWatchDashboard) (int, error) {
	highestY := 0

	for _, wid := range dashboard.Widgets {
		if highestY < wid.Y {
			highestY = wid.Y
		}
	}

	return highestY, nil
}
