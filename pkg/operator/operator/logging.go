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
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

type fluentdLog struct {
	Log string `json:"log"`
}

func LogGroupNameForJob(jobID spec.JobID) string {
	return fmt.Sprintf("%s/%s.%s", config.Cluster.LogGroup, jobID.APIName, jobID.ID)
}

func CreateLogGroupForJob(jobID spec.JobID) error {
	tags := map[string]string{
		"apiName": jobID.APIName,
		"jobID":   jobID.ID,
	}

	for key, value := range config.Cluster.Tags {
		tags[key] = value
	}

	_, err := config.AWS.CloudWatchLogs().CreateLogGroup(&cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(LogGroupNameForJob(jobID)),
		Tags:         aws.StringMap(tags),
	})
	if err != nil {
		return err
	}

	_, err = config.AWS.CloudWatchLogs().CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(LogGroupNameForJob(jobID)),
		LogStreamName: aws.String("operator"),
	})
	return err
}

func WriteToJobLogGroup(jobID spec.JobID, logLine string, logLines ...string) error {

	logStreams, err := config.AWS.CloudWatchLogs().DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(LogGroupNameForJob(jobID)),
		LogStreamNamePrefix: aws.String("operator"),
	})
	if err != nil {
		return err // TODO
	}

	if len(logStreams.LogStreams) == 0 {
		return err
	}

	logLines = slices.PrependStr(logLine, logLines)

	debug.Pp(logLines)
	inputLogEvents := make([]*cloudwatchlogs.InputLogEvent, len(logLines))
	curTime := libtime.ToMillis(time.Now())
	for i, line := range logLines {
		jsonBytes, err := json.Marshal(fluentdLog{Log: line})
		if err != nil {
			// TODO
		}
		inputLogEvents[i] = &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(string(jsonBytes)),
			Timestamp: aws.Int64(curTime),
		}
	}

	output, err := config.AWS.CloudWatchLogs().PutLogEvents(&cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(LogGroupNameForJob(jobID)),
		LogStreamName: aws.String("operator"),
		LogEvents:     inputLogEvents,
		SequenceToken: logStreams.LogStreams[0].UploadSequenceToken,
	},
	)
	debug.Pp(output)
	if err != nil {
		fmt.Println(err.Error())
		return err // TODO
	}

	return nil
}
