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
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

func (c *Client) ListEKSStacks(controlPlaneStackName string, nodeGroupStackNamePrefixes strset.Set) ([]*cloudformation.StackSummary, error) {
	mostRecentStackRecordsByName := map[string]*cloudformation.StackSummary{}
	stackSet := strset.Union(nodeGroupStackNamePrefixes, strset.New(controlPlaneStackName))

	err := c.CloudFormation().ListStacksPages(
		&cloudformation.ListStacksInput{},
		func(listStackOutput *cloudformation.ListStacksOutput, lastPage bool) bool {
			for _, stackSummary := range listStackOutput.StackSummaries {
				if stackSummary == nil || stackSummary.StackName == nil || !stackSet.HasWithPrefix(*stackSummary.StackName) {
					continue
				}
				if _, ok := mostRecentStackRecordsByName[*stackSummary.StackName]; !ok {
					mostRecentStackRecordsByName[*stackSummary.StackName] = stackSummary
				} else {
					created := mostRecentStackRecordsByName[*stackSummary.StackName].CreationTime
					if created != nil && stackSummary.CreationTime != nil && stackSummary.CreationTime.After(*created) {
						mostRecentStackRecordsByName[*stackSummary.StackName] = stackSummary
					}
				}

				if *stackSummary.StackName == controlPlaneStackName {
					return false
				}
			}

			return true
		},
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return getStackSummariesFromMap(mostRecentStackRecordsByName), nil
}

func getStackSummariesFromMap(stackSummaries map[string]*cloudformation.StackSummary) []*cloudformation.StackSummary {
	var stackSummariesSlice []*cloudformation.StackSummary
	for _, stack := range stackSummaries {
		stackSummariesSlice = append(stackSummariesSlice, stack)
	}
	return stackSummariesSlice
}
