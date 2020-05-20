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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

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
