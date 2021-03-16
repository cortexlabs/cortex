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

package clusterstate

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

const (
	controlPlaneTemplate = "eksctl-%s-cluster"
	operatorTemplate     = "eksctl-%s-nodegroup-cx-operator"

	spotTemplatePrefix     = "eksctl-%s-nodegroup-cx-ws"
	onDemandTemplatePrefix = "eksctl-%s-nodegroup-cx-wd"
)

type ClusterState struct {
	StatusMap    map[string]string // cloudformation stackname to cloudformation stackstatus
	ControlPlane string
	NodeGroups   []string
	Status       Status
}

func is(status string, allowedStatus string, allowedStatuses ...string) bool {
	statusSet := strset.New(allowedStatuses...)
	statusSet.Add(allowedStatus)

	return statusSet.Has(status)
}

func any(statuses []string, allowedStatus string, allowedStatuses ...string) bool {
	statusSet := strset.New(allowedStatuses...)
	statusSet.Add(allowedStatus)
	for _, stackStatus := range statuses {
		if statusSet.Has(stackStatus) {
			return true
		}
	}

	return false
}

func all(statuses []string, allowedStatus string, allowedStatuses ...string) bool {
	statusSet := strset.New(allowedStatuses...)
	statusSet.Add(allowedStatus)
	for _, stackStatus := range statuses {
		if !statusSet.Has(stackStatus) {
			return false
		}
	}

	return true
}

func (cs ClusterState) TableString() string {
	rows := make([][]interface{}, len(cs.StatusMap))
	rows[0] = []interface{}{
		cs.ControlPlane, cs.StatusMap[cs.ControlPlane],
	}

	idx := 1
	for _, nodeGroupName := range cs.NodeGroups {
		if status, ok := cs.StatusMap[nodeGroupName]; ok {
			rows[idx] = []interface{}{nodeGroupName, status}
			idx++
		}
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
	var items table.KeyValuePairs
	items.Add(cs.ControlPlane, cs.StatusMap[cs.ControlPlane])

	return t.MustFormat()
}

func getStatus(statusMap map[string]string, controlPlane string, clusterName string, region string) (Status, error) {
	// the order matters
	allStatuses := []string{}
	controlPlaneStatus := statusMap[controlPlane]
	nodeGroupStatuses := []string{}

	for stackName, status := range statusMap {
		allStatuses = append(allStatuses, status)
		if stackName != controlPlane {
			nodeGroupStatuses = append(nodeGroupStatuses, status)
		}
	}

	if any(allStatuses, string(StatusCreateFailedTimedOut)) {
		return StatusNotFound, ErrorUnexpectedCloudFormationStatus(clusterName, region, statusMap)
	}

	if len(nodeGroupStatuses) == 0 && controlPlaneStatus == string(StatusNotFound) {
		return StatusNotFound, nil
	}

	// controlplane stack may be created while nodegroup stacks aren't listed in cloudformation stacks during cluster spin up
	if len(nodeGroupStatuses) == 0 && is(controlPlaneStatus, cloudformation.StackStatusCreateComplete, cloudformation.StackStatusCreateInProgress) {
		return StatusCreateInProgress, nil
	}

	if any(allStatuses, cloudformation.StackStatusCreateFailed) {
		return StatusCreateFailed, nil
	}

	if any(allStatuses, cloudformation.StackStatusDeleteFailed) {
		return StatusDeleteFailed, nil
	}

	if any(allStatuses, cloudformation.StackStatusDeleteInProgress) {
		return StatusDeleteInProgress, nil
	}

	if all(allStatuses, cloudformation.StackStatusCreateComplete) {
		return StatusCreateComplete, nil
	}

	if all(allStatuses, cloudformation.StackStatusUpdateComplete) {
		return StatusUpdateComplete, nil
	}

	if all(allStatuses, cloudformation.StackStatusUpdateRollbackComplete) {
		return StatusUpdateRollbackComplete, nil
	}

	if all(allStatuses, cloudformation.StackStatusDeleteComplete) {
		return StatusDeleteComplete, nil
	}

	// nodegroup stacks are deleted first while control plane stack is still in create complete state
	if controlPlaneStatus == cloudformation.StackStatusCreateComplete &&
		all(nodeGroupStatuses, cloudformation.StackStatusDeleteInProgress, cloudformation.StackStatusDeleteComplete) {
		return StatusDeleteInProgress, nil
	}

	// controlplane stack may be in complete state while nodegroup stacks are still in creating or one nodegroup finishes before the other
	if controlPlaneStatus == cloudformation.StackStatusCreateComplete &&
		all(nodeGroupStatuses, cloudformation.StackStatusCreateInProgress, cloudformation.StackStatusCreateComplete) {
		return StatusCreateInProgress, nil
	}

	if controlPlaneStatus == cloudformation.StackStatusCreateComplete &&
		all(nodeGroupStatuses, cloudformation.StackStatusCreateComplete, cloudformation.StackStatusUpdateComplete, cloudformation.StackStatusUpdateRollbackComplete) {
		return StatusUpdateComplete, nil
	}

	return StatusNotFound, ErrorUnexpectedCloudFormationStatus(clusterName, region, statusMap)
}

func GetClusterState(awsClient *aws.Client, accessConfig *clusterconfig.AccessConfig) (*ClusterState, error) {
	controlPlaneStackName := fmt.Sprintf(controlPlaneTemplate, accessConfig.ClusterName)
	operatorStackName := fmt.Sprintf(operatorTemplate, accessConfig.ClusterName)
	spotStackNamePrefix := fmt.Sprintf(spotTemplatePrefix, accessConfig.ClusterName)
	onDemandStackNamePrefix := fmt.Sprintf(onDemandTemplatePrefix, accessConfig.ClusterName)

	nodeGroupStackPrefixesSet := strset.New(operatorStackName, spotStackNamePrefix, onDemandStackNamePrefix)

	stackSummaries, err := awsClient.ListEKSStacks(controlPlaneStackName, nodeGroupStackPrefixesSet)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get cluster state from cloudformation")
	}

	statusMap := map[string]string{}
	nodeGroupStackNames := []string{}
	var controlPlaneCreationTime time.Time

	for _, stackSummary := range stackSummaries {
		statusMap[*stackSummary.StackName] = *stackSummary.StackStatus
		if *stackSummary.StackName == controlPlaneStackName {
			controlPlaneCreationTime = *stackSummary.CreationTime
		} else {
			nodeGroupStackNames = append(nodeGroupStackNames, *stackSummary.StackName)
		}
	}

	if _, ok := statusMap[controlPlaneStackName]; !ok {
		statusMap[controlPlaneStackName] = string(StatusNotFound)
	}

	// add a timeout for situations where the control plane is listed in the cloudformation stacks but not the nodegroup stacks
	if !is(statusMap[controlPlaneStackName], string(StatusNotFound), cloudformation.StackStatusDeleteComplete) && len(nodeGroupStackNames) == 0 && time.Now().After(controlPlaneCreationTime.Add(30*time.Minute)) {
		statusMap[operatorStackName] = string(StatusCreateFailedTimedOut)
	}

	status, err := getStatus(statusMap, controlPlaneStackName, accessConfig.ClusterName, accessConfig.Region)
	if err != nil {
		return nil, err
	}

	return &ClusterState{
		ControlPlane: controlPlaneStackName,
		NodeGroups:   nodeGroupStackNames,
		StatusMap:    statusMap,
		Status:       status,
	}, nil
}

func CloudFormationURL(clusterName string, region string) string {
	return fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks?filteringText=eksctl-%s-", region, clusterName)
}
