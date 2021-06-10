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
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

func (c *Client) ListEKSStacks(controlPlaneStackName string, nodegroupStackNames strset.Set) ([]*cloudformation.StackSummary, error) {
	stackRecordsByName := map[string][]*cloudformation.StackSummary{}
	stackSet := strset.Union(nodegroupStackNames, strset.New(controlPlaneStackName))

	err := c.CloudFormation().ListStacksPages(
		&cloudformation.ListStacksInput{},
		func(listStackOutput *cloudformation.ListStacksOutput, lastPage bool) bool {
			for _, stackSummary := range listStackOutput.StackSummaries {
				if stackSummary == nil || stackSummary.StackName == nil || !stackSet.HasWithPrefix(*stackSummary.StackName) {
					continue
				}
				if _, ok := stackRecordsByName[*stackSummary.StackName]; !ok {
					stackRecordsByName[*stackSummary.StackName] = []*cloudformation.StackSummary{
						stackSummary,
					}
				} else {
					stackRecordsByName[*stackSummary.StackName] = append(stackRecordsByName[*stackSummary.StackName], stackSummary)
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

	var stackSummaries []*cloudformation.StackSummary
	for _, stacks := range stackRecordsByName {
		mostRecent := 0
		created := stacks[0].CreationTime
		for i, stack := range stacks[1:] {
			if stack.CreationTime == nil || (created == nil && stack.CreationTime != nil) {
				continue
			}
			if stack.CreationTime.After(*created) {
				mostRecent = i
			}
		}
		stackSummaries = append(stackSummaries, stacks[mostRecent])
	}

	return stackSummaries, nil
}
