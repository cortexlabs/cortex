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

// Return all activities for an ASG which do not have the "Successful" status, sorted from most to least recent
// Also returns true if the most recent activity for the ASG is "Successful", or if there are no activities for the ASG
func (c *Client) AutoscalingGroupUnsuccessfulActivities(asgName string) ([]*autoscaling.Activity, bool, error) {
	var activities []*autoscaling.Activity
	isSuccessful := true
	didCheckFirstActivity := false

	params := autoscaling.DescribeScalingActivitiesInput{
		AutoScalingGroupName: aws.String(asgName),
	}
	err := c.Autoscaling().DescribeScalingActivitiesPages(&params,
		func(page *autoscaling.DescribeScalingActivitiesOutput, lastPage bool) bool {
			for _, activity := range page.Activities {
				if !didCheckFirstActivity {
					if activity.StatusCode == nil || *activity.StatusCode != autoscaling.ScalingActivityStatusCodeSuccessful {
						isSuccessful = false
					}
					didCheckFirstActivity = true
				}

				if activity.StatusCode == nil || *activity.StatusCode != autoscaling.ScalingActivityStatusCodeSuccessful {
					activities = append(activities, activity)
				}
			}
			return true
		})

	if err != nil {
		return nil, false, errors.WithStack(err)
	}

	return activities, isSuccessful, nil
}
