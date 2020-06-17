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

package cloud

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func WriteToJobLogGroup(apiName string, jobID string, logLine string, logLines ...string) error {
	logGroupName := fmt.Sprintf("%s/%s.%s", config.Cluster.LogGroup, apiName, jobID)

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
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String("operator"),
		LogEvents:     inputLogEvents,
	},
	)

	if err != nil {
		return err // TODO
	}

	return nil
}
