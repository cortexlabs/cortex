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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
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
	clusterName           string
	StatusMap             map[string]string // cloudformation stackname to cloudformation stackstatus
	ControlPlane          string
	NodeGroupsStacks      []string
	StaleNodeGroupsStacks []string
	Status                Status
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
	for _, nodeGroupName := range cs.NodeGroupsStacks {
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

func (cs ClusterState) GetStaleNodeGroupNames() []string {
	staleNodeGroups := []string{}
	spotPrefix := fmt.Sprintf(spotTemplatePrefix+"-", cs.clusterName)
	onDemandPrefix := fmt.Sprintf(onDemandTemplatePrefix+"-", cs.clusterName)

	for _, stackName := range cs.StaleNodeGroupsStacks {
		if strings.HasPrefix(stackName, spotPrefix) {
			ngName := strings.TrimPrefix(stackName, spotPrefix)
			staleNodeGroups = append(staleNodeGroups, ngName)
		}
		if strings.HasPrefix(stackName, onDemandPrefix) {
			ngName := strings.TrimPrefix(stackName, onDemandPrefix)
			staleNodeGroups = append(staleNodeGroups, ngName)
		}
	}

	return staleNodeGroups
}

func getStatus(statusMap map[string]string, controlPlane string, operatorStackName string, clusterName string, region string, ngNames []string, ngSpotEnabled []bool) (Status, []string, error) {
	statusMapCopy := maps.StrMapsCopy(statusMap)

	allStatuses := []string{}
	controlPlaneStatus := statusMapCopy[controlPlane]
	allStatusesButTheRemovedOnes := []string{controlPlaneStatus}
	existingNodeGroupStatuses := []string{}
	staleNodeGroupStatuses := []string{}
	staleNodeGroupStacks := []string{}

	for i, ngName := range ngNames {
		availability := "d"
		if ngSpotEnabled[i] {
			availability = "s"
		}
		eksStackName := fmt.Sprintf("eksctl-%s-nodegroup-cx-w%s-%s", clusterName, availability, ngName)
		status, ok := statusMapCopy[eksStackName]
		if !ok {
			return StatusNotFound, nil, ErrorUnexpectedCloudFormationStatus(clusterName, region, statusMapCopy)
		}

		allStatuses = append(allStatuses, status)
		existingNodeGroupStatuses = append(existingNodeGroupStatuses, status)
		delete(statusMapCopy, eksStackName)
	}

	for stackName, status := range statusMapCopy {
		allStatuses = append(allStatuses, status)
		if stackName != controlPlane && stackName != operatorStackName {
			staleNodeGroupStatuses = append(staleNodeGroupStatuses, status)
			if !any([]string{status}, cloudformation.StackStatusDeleteComplete, cloudformation.StackStatusDeleteInProgress) {
				staleNodeGroupStacks = append(staleNodeGroupStacks, stackName)
			}
		}
	}

	allStatusesButTheRemovedOnes = append(allStatusesButTheRemovedOnes, existingNodeGroupStatuses...)

	if any(allStatuses, string(StatusCreateFailedTimedOut)) {
		return StatusNotFound, nil, ErrorUnexpectedCloudFormationStatus(clusterName, region, statusMap)
	}

	if len(existingNodeGroupStatuses) == 0 && controlPlaneStatus == string(StatusNotFound) {
		return StatusNotFound, staleNodeGroupStacks, nil
	}

	// controlplane stack may be created while nodegroup stacks aren't listed in cloudformation stacks during cluster spin up
	if len(existingNodeGroupStatuses) == 0 && is(controlPlaneStatus, cloudformation.StackStatusCreateComplete, cloudformation.StackStatusCreateInProgress) {
		return StatusCreateInProgress, staleNodeGroupStacks, nil
	}

	if any(allStatuses, cloudformation.StackStatusCreateFailed) {
		return StatusCreateFailed, staleNodeGroupStacks, nil
	}

	if any(allStatuses, cloudformation.StackStatusDeleteFailed) {
		return StatusDeleteFailed, staleNodeGroupStacks, nil
	}

	if any(allStatusesButTheRemovedOnes, cloudformation.StackStatusDeleteInProgress) {
		return StatusDeleteInProgress, staleNodeGroupStacks, nil
	}

	if all(allStatuses, cloudformation.StackStatusCreateComplete) {
		return StatusCreateComplete, staleNodeGroupStacks, nil
	}

	if all(allStatuses, cloudformation.StackStatusUpdateComplete) {
		return StatusUpdateComplete, staleNodeGroupStacks, nil
	}

	if all(allStatuses, cloudformation.StackStatusUpdateRollbackComplete) {
		return StatusUpdateRollbackComplete, staleNodeGroupStacks, nil
	}

	if all(allStatuses, cloudformation.StackStatusDeleteComplete) {
		return StatusDeleteComplete, staleNodeGroupStacks, nil
	}

	// nodegroup stacks are deleted first while control plane stack is still in create complete state
	if controlPlaneStatus == cloudformation.StackStatusCreateComplete &&
		all(existingNodeGroupStatuses, cloudformation.StackStatusDeleteInProgress, cloudformation.StackStatusDeleteComplete) {
		return StatusDeleteInProgress, staleNodeGroupStacks, nil
	}

	// controlplane stack may be in complete state while nodegroup stacks are still in creating or one nodegroup finishes before the other
	if controlPlaneStatus == cloudformation.StackStatusCreateComplete &&
		any(existingNodeGroupStatuses, cloudformation.StackStatusCreateInProgress) {
		return StatusCreateInProgress, staleNodeGroupStacks, nil
	}

	if controlPlaneStatus == cloudformation.StackStatusCreateComplete &&
		all(existingNodeGroupStatuses, cloudformation.StackStatusCreateComplete, cloudformation.StackStatusUpdateComplete, cloudformation.StackStatusUpdateRollbackComplete) {
		return StatusUpdateComplete, staleNodeGroupStacks, nil
	}

	return StatusNotFound, nil, ErrorUnexpectedCloudFormationStatus(clusterName, region, statusMap)
}

func GetClusterState(awsClient *aws.Client, clusterConfig *clusterconfig.Config) (*ClusterState, error) {
	controlPlaneStackName := fmt.Sprintf(controlPlaneTemplate, clusterConfig.ClusterName)
	operatorStackName := fmt.Sprintf(operatorTemplate, clusterConfig.ClusterName)
	spotStackNamePrefix := fmt.Sprintf(spotTemplatePrefix, clusterConfig.ClusterName)
	onDemandStackNamePrefix := fmt.Sprintf(onDemandTemplatePrefix, clusterConfig.ClusterName)

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

	status, staleStacks, err := getStatus(
		statusMap,
		controlPlaneStackName,
		operatorStackName,
		clusterConfig.ClusterName,
		clusterConfig.Region,
		clusterconfig.GetNodeGroupNames(clusterConfig.NodeGroups),
		clusterconfig.GetNodeGroupAvailabilities(clusterConfig.NodeGroups),
	)
	if err != nil {
		return nil, err
	}

	return &ClusterState{
		clusterName:           clusterConfig.ClusterName,
		ControlPlane:          controlPlaneStackName,
		NodeGroupsStacks:      nodeGroupStackNames,
		StaleNodeGroupsStacks: staleStacks,
		StatusMap:             statusMap,
		Status:                status,
	}, nil
}

func CloudFormationURL(clusterName string, region string) string {
	return fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks?filteringText=eksctl-%s-", region, clusterName)
}
