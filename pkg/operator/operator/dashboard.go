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

package operator

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

// AddAPIToDashboard adds an API's plots to an existing dashboard
func AddAPIToDashboard(dashboardName string, nameAPI string) error {
	// get current dashboard from cloudwatch
	currDashboardOutput, err := config.AWS.CloudWatch().GetDashboard(&cloudwatch.GetDashboardInput{
		DashboardName: pointer.String(dashboardName),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	currDashboardString := *currDashboardOutput.DashboardBody

	var currDashboard aws.CloudWatchDashboard
	err = json.Unmarshal([]byte(currDashboardString), &currDashboard)
	if err != nil {
		return errors.WithStack(err)
	}

	// get lowest element on the dashboard (need to place new widgets below all existing widgets)
	highestY, err := aws.HighestY(currDashboard)
	if err != nil {
		return err
	}

	// create widgets for title and metrics
	apiTitleWidget := aws.TextWidget(1, highestY+1, 22, 1, "## "+nameAPI)
	// top left widget
	statCodeWidget := aws.MetricWidget(1, highestY+2, 11, 6, statusCodeMetric(dashboardName, nameAPI), "responses per minute", "Sum", 60, config.AWS.Region)
	// top right widget
	inFlightWidget := aws.MetricWidget(12, highestY+2, 11, 6, inFlightMetric(dashboardName, nameAPI), "total in-flight requests", "Sum", 10, config.AWS.Region)
	// bottem left widget
	latencyWidgetP50 := aws.MetricWidget(1, highestY+8, 11, 6, latencyMetric(dashboardName, nameAPI), "median response time (ms)", "p50", 60, config.AWS.Region)
	// bottom right widget
	latencyWidgetP99 := aws.MetricWidget(12, highestY+8, 11, 6, latencyMetric(dashboardName, nameAPI), "p99 response time (ms)", "p99", 60, config.AWS.Region)

	// append new API metrics widgets to existing widgets
	currDashboard.Widgets = append(currDashboard.Widgets, apiTitleWidget, statCodeWidget, inFlightWidget, latencyWidgetP50, latencyWidgetP99)
	currDashboardJSON, err := json.Marshal(currDashboard)
	if err != nil {
		return errors.WithStack(err)
	}

	// upload updated dashboard to cloudwatch
	_, err = config.AWS.CloudWatch().PutDashboard(&cloudwatch.PutDashboardInput{
		DashboardName: pointer.String(dashboardName),
		DashboardBody: pointer.String(string(currDashboardJSON)),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// RemoveAPIFromDashboard removes an API's plots from an existing dashboard
func RemoveAPIFromDashboard(allAPINames []string, clusterName, apiName string) error {
	// create a new base dashboard
	err := config.AWS.CreateDashboard(clusterName, "# cortex monitoring dashboard")
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to remove API \"%s\" from cloudwatch dashboard", apiName))
	}

	// update dashboard by adding all APIs except the one to delete
	for _, allAPIname := range allAPINames {
		if allAPIname != apiName {
			err = AddAPIToDashboard(clusterName, allAPIname)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to re-add API \"%s\" to cloudwatch dashboard", allAPIname))
			}
		}
	}

	return nil
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

func statusCodeMetric(dashboardName string, nameAPI string) []interface{} {
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

func DashboardURL() string {
	return fmt.Sprintf("https://%s.console.aws.amazon.com/cloudwatch/home#dashboards:name=%s", *config.Cluster.Region, config.Cluster.ClusterName)
}
