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

package batchapi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func logGroupNameForAPI(apiName string) string {
	return fmt.Sprintf("%s/%s", config.Cluster.ClusterName, apiName)
}

func logGroupNameForJob(jobKey spec.JobKey) string {
	return logGroupNameForAPI(jobKey.APIName)
}

func operatorLogStream(jobKey spec.JobKey) string {
	return fmt.Sprintf("%s_%s", jobKey.ID, _operatorService)
}

// Checks if log group exists before creating it
func ensureLogGroupForAPI(apiName string) error {
	apiNameLogGroup := logGroupNameForAPI(apiName)

	logGroupExists, err := config.AWS.DoesLogGroupExist(apiNameLogGroup)
	if err != nil {
		return err
	}

	if !logGroupExists {
		input := &cloudwatchlogs.CreateLogGroupInput{
			LogGroupName: aws.String(apiNameLogGroup),
		}

		// There should always be a default tag. Tags are only empty when running local operator without the tags field specified in a cluster config file.
		if len(config.Cluster.Tags) > 0 {
			input.Tags = aws.StringMap(config.Cluster.Tags)
		}

		_, err := config.AWS.CloudWatchLogs().CreateLogGroup(input)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func createOperatorLogStreamForJob(jobKey spec.JobKey) error {
	_, err := config.AWS.CloudWatchLogs().CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(logGroupNameForJob(jobKey)),
		LogStreamName: aws.String(operatorLogStream(jobKey)),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func writeToJobLogStream(jobKey spec.JobKey, logLine string, logLines ...string) error {
	logStreams, err := config.AWS.CloudWatchLogs().DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(logGroupNameForJob(jobKey)),
		LogStreamNamePrefix: aws.String(operatorLogStream(jobKey)),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if len(logStreams.LogStreams) == 0 {
		return errors.ErrorUnexpected(fmt.Sprintf("unable to find log stream named '%s' in log group %s", operatorLogStream(jobKey), logGroupNameForJob(jobKey)))
	}

	logLines = append([]string{logLine}, logLines...)

	inputLogEvents := make([]*cloudwatchlogs.InputLogEvent, len(logLines))
	curTime := libtime.ToMillis(time.Now())
	for i, line := range logLines {
		jsonBytes, _ := json.Marshal(fluentdLog{Log: line})
		inputLogEvents[i] = &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(string(jsonBytes)),
			Timestamp: aws.Int64(curTime),
		}
	}

	_, err = config.AWS.CloudWatchLogs().PutLogEvents(&cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(logGroupNameForJob(jobKey)),
		LogStreamName: aws.String(operatorLogStream(jobKey)),
		LogEvents:     inputLogEvents,
		SequenceToken: logStreams.LogStreams[0].UploadSequenceToken,
	},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
