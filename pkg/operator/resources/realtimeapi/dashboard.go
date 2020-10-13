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
	"fmt"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func addAPIToDashboard(dashboardName string, apiName string) error {
	// get current dashboard from cloudwatch (or a new dashboard if it was deleted)
	dashboard, err := config.AWS.GetDashboardOrEmpty(dashboardName, consts.DashboardTitle)
	if err != nil {
		return err
	}

	err = addAPIToDashboardObject(dashboard, dashboardName, apiName)
	if err != nil {
		return err
	}

	err = config.AWS.PutDashboard(dashboard, dashboardName)
	if err != nil {
		return err
	}

	return nil
}

func removeAPIFromDashboard(allAPINames []string, dashboardName string, apiToRemove string) error {
	// create a new base dashboard
	dashboard := config.AWS.NewDashboard(consts.DashboardTitle)

	// update dashboard by adding all APIs except the one to delete
	for _, apiName := range allAPINames {
		if apiName == apiToRemove {
			continue
		}
		err := addAPIToDashboardObject(dashboard, dashboardName, apiName)
		if err != nil {
			return err
		}
	}

	err := config.AWS.PutDashboard(dashboard, dashboardName)
	if err != nil {
		return err
	}

	return nil
}

func addAPIToDashboardObject(dashboard *aws.CloudWatchDashboard, dashboardName string, apiName string) error {
	// get lowest element on the dashboard (need to place new widgets below all existing widgets)
	highestY, err := aws.HighestY(dashboard)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to add API \"%s\" to cloudwatch dashboard", apiName))
	}

	// create widget for title
	dashboard.Widgets = append(dashboard.Widgets, aws.TextWidget(1, highestY+1, 22, 1, "## "+apiName))

	grid, err := aws.NewVerticalGrid(1, highestY+2, 6, 11, 3)
	if err != nil {
		return nil
	}

	// first grid column
	grid.AddWidget(statusCodeMetric(dashboardName, apiName), "responses per minute", "Sum", 60, config.AWS.Region)
	grid.AddWidget(latencyMetric(dashboardName, apiName), "median response time (ms)", "p50", 60, config.AWS.Region)
	grid.AddWidget(latencyMetric(dashboardName, apiName), "p99 response time (ms)", "p99", 60, config.AWS.Region)

	// second grid column
	grid.AddWidget(inFlightMetric(dashboardName, apiName), "total in-flight requests", "Sum", 10, config.AWS.Region)
	grid.AddWidget(inFlightMetric(dashboardName, apiName), "avg in-flight requests per replica", "Average", 10, config.AWS.Region)
	// setting the period to 10 seconds because the publishing frequency of the request monitor is 10 seconds
	grid.AddWidget(inFlightMetric(dashboardName, apiName), "active replicas", "SampleCount", 10, config.AWS.Region)

	// append new API metrics widgets to existing widgets
	dashboard.Widgets = append(dashboard.Widgets, grid.Widgets...)

	return nil
}

func inFlightMetric(dashboardName string, apiName string) []interface{} {
	var metric []interface{}
	metric = append(metric, dashboardName)
	metric = append(metric, "in-flight")
	metric = append(metric, "apiName")
	metric = append(metric, apiName)

	return []interface{}{metric}
}

func latencyMetric(dashboardName string, apiName string) []interface{} {
	var metric []interface{}
	metric = append(metric, dashboardName)
	metric = append(metric, "Latency")
	metric = append(metric, "APIName")
	metric = append(metric, apiName)
	metric = append(metric, "metric_type")
	metric = append(metric, "histogram")

	return []interface{}{metric}
}

func statusCodeMetric(dashboardName string, apiName string) []interface{} {
	var metric2XX []interface{}
	metric2XX = append(metric2XX, dashboardName)
	metric2XX = append(metric2XX, "StatusCode")
	metric2XX = append(metric2XX, "APIName")
	metric2XX = append(metric2XX, apiName)
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

func DashboardURL() string {
	return fmt.Sprintf("https://%s.console.aws.amazon.com/cloudwatch/home#dashboards:name=%s", *config.Cluster.Region, config.Cluster.ClusterName)
}
