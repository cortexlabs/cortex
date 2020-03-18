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
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

func (c *Client) ListStacks(controlPlaneStack string, nodegroupStacks ...string) ([]*cloudformation.StackSummary, error) {
	stacks, err := c.CloudFormation().ListStacks(
		&cloudformation.ListStacksInput{},
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var searchStacks []*cloudformation.StackSummary
	var stackSummaries []*cloudformation.StackSummary

	nodegroupStackSet := strset.New(nodegroupStacks...)

	for idx, stackSummary := range stacks.StackSummaries {
		if *stackSummary.StackName == controlPlaneStack {
			stackSummaries = append(stackSummaries, stackSummary)
			searchStacks = stacks.StackSummaries[:idx]
			break
		}
	}
	if len(stackSummaries) == 0 {
		return nil, nil
	}

	for _, stackSummary := range searchStacks {
		if nodegroupStackSet.Has(*stackSummary.StackName) {
			stackSummaries = append(stackSummaries, stackSummary)
		}
	}

	return stackSummaries, nil
}
