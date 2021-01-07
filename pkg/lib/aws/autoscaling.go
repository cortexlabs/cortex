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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

// if specified, all tags must be present
func (c *Client) AutoscalingGroups(tags map[string]string) ([]*autoscaling.Group, error) {
	var asgs []*autoscaling.Group

	params := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: nil,
	}
	err := c.Autoscaling().DescribeAutoScalingGroupsPages(&params,
		func(page *autoscaling.DescribeAutoScalingGroupsOutput, lastPage bool) bool {
			for _, asg := range page.AutoScalingGroups {
				asgTags := make(map[string]string, len(asg.Tags))
				for _, asgTag := range asg.Tags {
					if asgTag.Key != nil && asgTag.Value != nil {
						asgTags[*asgTag.Key] = *asgTag.Value
					}
				}

				missingTag := false
				for key, value := range tags {
					if asgTags[key] != value {
						missingTag = true
						break
					}
				}

				if missingTag {
					continue
				}

				asgs = append(asgs, asg)
			}

			return true
		})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return asgs, nil
}

// Returns the most recent activity for the ASG, or nil if there are no activities
func (c *Client) MostRecentASGActivity(asgName string) (*autoscaling.Activity, error) {
	resp, err := c.Autoscaling().DescribeScalingActivities(&autoscaling.DescribeScalingActivitiesInput{
		AutoScalingGroupName: aws.String(asgName),
		MaxRecords:           aws.Int64(1),
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(resp.Activities) == 0 {
		return nil, nil
	}

	return resp.Activities[0], nil
}
