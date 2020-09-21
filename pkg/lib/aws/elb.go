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
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

// returns the the first load balancer which has all of the specified tags, or nil if no load balancers match
func (c *Client) LoadBalancer(tags map[string]string) (*elbv2.LoadBalancer, error) {
	var loadBalancer *elbv2.LoadBalancer
	var fnErr error

	params := elbv2.DescribeLoadBalancersInput{
		PageSize: aws.Int64(20), // 20 is the limit for DescribeTags()
	}
	err := c.ELBV2().DescribeLoadBalancersPages(&params,
		func(page *elbv2.DescribeLoadBalancersOutput, lastPage bool) bool {
			arns := make([]string, len(page.LoadBalancers))
			loadBalancers := make(map[string]*elbv2.LoadBalancer)

			for i := range page.LoadBalancers {
				arn := *page.LoadBalancers[i].LoadBalancerArn
				arns[i] = arn
				loadBalancers[arn] = page.LoadBalancers[i]
			}

			tagsOutput, err := c.ELBV2().DescribeTags(&elbv2.DescribeTagsInput{
				ResourceArns: aws.StringSlice(arns),
			})
			if err != nil {
				fnErr = err
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
					loadBalancer = loadBalancers[*tagDescription.ResourceArn]
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
