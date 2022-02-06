/*
Copyright 2022 Cortex Labs, Inc.

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
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type ClassicLoadBalancerState string

const (
	LoadBalancerStateInService    ClassicLoadBalancerState = "InService"
	LoadBalancerStateOutOfService ClassicLoadBalancerState = "OutOfService"
	LoadBalancerStateUnknown      ClassicLoadBalancerState = "Unknown"
)

func (state ClassicLoadBalancerState) String() string {
	return string(state)
}

// returns the first classic load balancer which has all of the specified tags, or nil if no load balancers match
func (c *Client) FindLoadBalancer(tags map[string]string) (*elb.LoadBalancerDescription, error) {
	var loadBalancer *elb.LoadBalancerDescription
	var fnErr error

	params := elb.DescribeLoadBalancersInput{
		PageSize: aws.Int64(20), // 20 is the limit for DescribeTags()
	}
	err := c.ELB().DescribeLoadBalancersPages(&params,
		func(page *elb.DescribeLoadBalancersOutput, lastPage bool) bool {
			loadBalancerNames := make([]string, len(page.LoadBalancerDescriptions))
			loadBalancers := make(map[string]*elb.LoadBalancerDescription)

			for i := range page.LoadBalancerDescriptions {
				loadBalancerName := *page.LoadBalancerDescriptions[i].LoadBalancerName
				loadBalancerNames[i] = loadBalancerName
				loadBalancers[loadBalancerName] = page.LoadBalancerDescriptions[i]
			}

			tagsOutput, err := c.ELB().DescribeTags(&elb.DescribeTagsInput{
				LoadBalancerNames: aws.StringSlice(loadBalancerNames),
			})
			if err != nil {
				fnErr = errors.WithStack(err)
				return false
			}

			for _, tagDescription := range tagsOutput.TagDescriptions {
				lbTags := make(map[string]string, len(tagDescription.Tags))
				for _, lbTag := range tagDescription.Tags {
					if lbTag.Key != nil && lbTag.Value != nil {
						lbTags[*lbTag.Key] = *lbTag.Value
					}
				}

				missingTag := false
				for key, value := range tags {
					if lbTags[key] != value {
						missingTag = true
						break
					}
				}

				if !missingTag {
					loadBalancer = loadBalancers[*tagDescription.LoadBalancerName]
					return false
				}
			}

			return true
		})

	if err != nil {
		return nil, errors.WithStack(err)
	}
	if fnErr != nil {
		return nil, fnErr
	}

	return loadBalancer, nil
}

func (c *Client) IsLoadBalancerHealthy(loadBalancerName string) (bool, error) {
	instanceHealthOutput, err := c.ELB().DescribeInstanceHealth(&elb.DescribeInstanceHealthInput{
		LoadBalancerName: &loadBalancerName,
	})
	if err != nil {
		return false, errors.WithStack(err)
	}

	for _, instance := range instanceHealthOutput.InstanceStates {
		if instance == nil {
			continue
		}
		if instance.State != nil && *instance.State != LoadBalancerStateInService.String() {
			return false, nil
		}
	}

	return true, nil
}
