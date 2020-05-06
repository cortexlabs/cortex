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

// UpdateDashboard updates existing dashboard by adding new widgets for new API
func (c *Client) UpdateDashboard(apiName string) error {

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
	cloudwatchBaseBody := map[string]interface{}{"start": "-PT1H", "periodOverride": "inherit", "widgets": []interface{}{createTextWidget(8, 1, 8, 1, "# CORTEX MONITORING DASHBOARD")}}
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

//createTextWidget create new text widget with properties as parameter
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
