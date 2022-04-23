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

package clusterstate

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

const (
	controlPlaneTemplate = "eksctl-%s-cluster"
	operatorTemplate     = "eksctl-%s-nodegroup-cx-operator"

	spotTemplatePrefix     = "eksctl-%s-nodegroup-cx-ws"
	onDemandTemplatePrefix = "eksctl-%s-nodegroup-cx-wd"
)

type ClusterStacks struct {
	clusterName       string
	region            string
	ControlPlaneStack *cloudformation.StackSummary
	OperatorStack     *cloudformation.StackSummary
	NodeGroupsStacks  []*cloudformation.StackSummary
}

func (cs ClusterStacks) TableString() string {
	numStacks := len(cs.NodeGroupsStacks)
	if cs.ControlPlaneStack != nil {
		numStacks++
	}
	if cs.OperatorStack != nil {
		numStacks++
	}

	idx := 0
	rows := make([][]interface{}, numStacks)

	if cs.ControlPlaneStack != nil {
		rows[idx] = []interface{}{
			*cs.ControlPlaneStack.StackName, *cs.ControlPlaneStack.StackStatus,
		}
		idx++
	}
	if cs.OperatorStack != nil {
		rows[idx] = []interface{}{
			*cs.OperatorStack.StackName, *cs.OperatorStack.StackStatus,
		}
		idx++
	}

	for _, stack := range cs.NodeGroupsStacks {
		rows[idx] = []interface{}{*stack.StackName, *stack.StackStatus}
		idx++
	}

	t := table.Table{
		Headers: []table.Header{
			{
				Title: "cloudformation stack name",
			},
			{
				Title: "status",
			},
		},
		Rows: rows,
	}
	return t.MustFormat()
}

func GetClusterStacks(awsClient *aws.Client, accessConfig *clusterconfig.AccessConfig) (ClusterStacks, error) {
	controlPlaneStackName := fmt.Sprintf(controlPlaneTemplate, accessConfig.ClusterName)
	operatorStackName := fmt.Sprintf(operatorTemplate, accessConfig.ClusterName)
	spotStackNamePrefix := fmt.Sprintf(spotTemplatePrefix, accessConfig.ClusterName)
	onDemandStackNamePrefix := fmt.Sprintf(onDemandTemplatePrefix, accessConfig.ClusterName)
	nodeGroupStackPrefixesSet := strset.New(operatorStackName, spotStackNamePrefix, onDemandStackNamePrefix)

	stackSummaries, err := awsClient.ListEKSStacks(controlPlaneStackName, nodeGroupStackPrefixesSet)
	if err != nil {
		return ClusterStacks{}, errors.Wrap(err, "unable to get cluster state from cloudformation")
	}

	var controlPlaneStack, operatorStack *cloudformation.StackSummary
	var ngStacks []*cloudformation.StackSummary
	for _, stack := range stackSummaries {
		if stack == nil || stack.StackName == nil {
			continue
		}
		if strings.HasPrefix(*stack.StackName, spotStackNamePrefix) || strings.HasPrefix(*stack.StackName, onDemandStackNamePrefix) {
			ngStacks = append(ngStacks, stack)
		} else if *stack.StackName == controlPlaneStackName {
			controlPlaneStack = stack
		} else if *stack.StackName == operatorStackName {
			operatorStack = stack
		}
	}

	return ClusterStacks{
		clusterName:       accessConfig.ClusterName,
		region:            accessConfig.Region,
		ControlPlaneStack: controlPlaneStack,
		OperatorStack:     operatorStack,
		NodeGroupsStacks:  ngStacks,
	}, nil
}

func GetClusterState(stacks ClusterStacks) State {
	controlPlaneStackName := fmt.Sprintf(controlPlaneTemplate, stacks.clusterName)

	if stacks.ControlPlaneStack == nil || stacks.ControlPlaneStack.StackName == nil {
		return StateClusterDoesntExist
	}
	if *stacks.ControlPlaneStack.StackName == controlPlaneStackName {
		controlPlaneStatus := *stacks.ControlPlaneStack.StackStatus

		if slices.HasString([]string{
			cloudformation.StackStatusDeleteComplete,
			cloudformation.StackStatusDeleteInProgress,
		}, controlPlaneStatus) {
			return StateClusterDoesntExist
		}

		if slices.HasString([]string{
			cloudformation.StackStatusCreateInProgress,
			cloudformation.StackStatusCreateComplete,
			cloudformation.StackStatusUpdateInProgress,
			cloudformation.StackStatusUpdateComplete,
			cloudformation.StackStatusRollbackInProgress,
			cloudformation.StackStatusRollbackComplete,
			cloudformation.StackStatusUpdateRollbackInProgress,
			cloudformation.StackStatusUpdateRollbackComplete,
		}, controlPlaneStatus) {
			return StateClusterExists
		}

		return StateClusterInUnexpectedState
	}
	return StateClusterDoesntExist
}

func CloudFormationURL(clusterName string, region string) string {
	return fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks?filteringText=eksctl-%s-", region, clusterName)
}
