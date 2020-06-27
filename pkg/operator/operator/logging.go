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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func LogGroupNameForJob(apiName string, jobID string) string {
	return fmt.Sprintf("%s/%s.%s", config.Cluster.LogGroup, apiName, jobID)
}

func CreateLogGroupForJob(apiName string, jobID string) error {
	tags := map[string]string{
		"apiName": apiName,
		"jobID":   jobID,
	}

	for key, value := range config.Cluster.Tags {
		tags[key] = value
	}

	_, err := config.AWS.CloudWatchLogs().CreateLogGroup(&cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(LogGroupNameForJob(apiName, jobID)),
		Tags:         aws.StringMap(tags),
	})

	return err
}

func WriteToJobLogGroup(apiName string, jobID string, logLine string, logLines ...string) error {

	logLines = slices.PrependStr(logLine, logLines)

	inputLogEvents := make([]*cloudwatchlogs.InputLogEvent, len(logLines))
	curTime := time.Now().Unix()
	for i, line := range logLines {
		inputLogEvents[i] = &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(line),
			Timestamp: aws.Int64(curTime),
		}
	}

	_, err := config.AWS.CloudWatchLogs().PutLogEvents(&cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(LogGroupNameForJob(apiName, jobID)),
		LogStreamName: aws.String("operator"),
		LogEvents:     inputLogEvents,
	},
	)
	if err != nil {
		fmt.Println(err.Error())
		return err // TODO
	}

	return nil
}
