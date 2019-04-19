/*
Copyright 2019 Cortex Labs, Inc.

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
	"bytes"
	"encoding/json"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
)

type FluentdLog struct {
	Log    string `json:"log"`
	Stream string `json:"stream"`
	Docker struct {
		ContainerID string `json:"container_id"`
	} `json:"docker"`
	Kubernetes struct {
		ContainerName string `json:"container_name"`
		NamespaceName string `json:"namespace_name"`
		PodName       string `json:"pod_name"`
		OrphanedName  string `json:"orphaned_namespace"`
		NamespaceID   string `json:"namespace_id"`
	} `json:"kubernetes"`
}

func (c *Client) GetLogs(prefix, logGroup string) (string, error) {
	logGroupNamePrefix := "var.log.containers."

	ignoreLogStreamNameRegexes := []*regexp.Regexp{
		regexp.MustCompile(`-exec-[0-9]+`),
		regexp.MustCompile(`_spark-init-`),
		regexp.MustCompile(`_cortex_serve-`),
	}

	logStreamsOut, err := c.cloudWatchLogsClient.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		Limit:               aws.Int64(50),
		LogGroupName:        aws.String(logGroup),
		LogStreamNamePrefix: aws.String(logGroupNamePrefix + prefix),
	})
	if err != nil {
		return "", errors.Wrap(err, "cloudwatch logs", prefix)
	}

	var allLogsBuf bytes.Buffer

	for i, logStream := range logStreamsOut.LogStreams {
		if !regex.MatchAnyRegex(*logStream.LogStreamName, ignoreLogStreamNameRegexes) {
			getLogEventsInput := &cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  &logGroup,
				LogStreamName: logStream.LogStreamName,
				StartFromHead: aws.Bool(true),
			}

			err := c.cloudWatchLogsClient.GetLogEventsPages(getLogEventsInput, func(logEventsOut *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool {
				for _, logEvent := range logEventsOut.Events {
					var log FluentdLog
					// millis := *logEvent.Timestamp
					// timestamp := time.Unix(0, millis*int64(time.Millisecond))
					json.Unmarshal([]byte(*logEvent.Message), &log)
					allLogsBuf.WriteString(log.Log)
				}
				return true
			})
			if err != nil {
				return "", errors.Wrap(err, "cloudwatch logs", prefix)
			}

			if i < len(logStreamsOut.LogStreams)-1 {
				allLogsBuf.WriteString("\n----------\n\n")
			}
		}
	}

	return allLogsBuf.String(), nil
}
