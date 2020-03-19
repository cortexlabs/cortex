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

package clusterstate

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

const (
	controlPlaneTemplate = "eksctl-%s-cluster"
	operatorTemplate     = "eksctl-%s-nodegroup-ng-cortex-operator"
	spotTemplate         = "eksctl-%s-nodegroup-ng-cortex-worker-spot"
	onDemandTemplate     = "eksctl-%s-nodegroup-ng-cortex-worker-on-demand"
)

type ClusterState struct {
	StatusMap    map[string]string // cloudformation stackname to cloudformation stackstatus
	ControlPlane string
	Nodegroups   []string
	Status       Status
}

func any(statuses []string, allowedStatuses ...string) bool {
	statusSet := strset.New(allowedStatuses...)
	for _, stackStatus := range statuses {
		if statusSet.Has(stackStatus) {
			return true
		}
	}

	return false
}

func all(statuses []string, allowedStatuses ...string) bool {
	statusSet := strset.New(allowedStatuses...)
	for _, stackStatus := range statuses {
		if !statusSet.Has(stackStatus) {
			return false
		}
	}

	return true
}

func (cs ClusterState) TableString() string {
	var items table.KeyValuePairs
	items.Add(cs.ControlPlane, cs.StatusMap[cs.ControlPlane])

	for _, nodeGroupName := range cs.Nodegroups {
		items.Add(nodeGroupName, cs.StatusMap[nodeGroupName])
	}
	return items.String()
}

func getStatus(statusMap map[string]string, controlPlane string) (Status, error) {
	// the order matters

	allStatuses := []string{}
	controlPlaneStatus := []string{statusMap[controlPlane]}
	nodeGroupStatuses := []string{}

	for stackName, status := range statusMap {
		allStatuses = append(allStatuses, status)
		if stackName != controlPlane {
			nodeGroupStatuses = append(nodeGroupStatuses, status)
		}
	}

	if any(allStatuses, cloudformation.StackStatusCreateFailed) {
		return StatusCreateFailed, nil
	}

	if any(allStatuses, cloudformation.StackStatusDeleteFailed) {
		return StatusDeleteFailed, nil
	}

	if all(allStatuses, string(StatusNotFound)) {
		return StatusCreateComplete, nil
	}

	if all(allStatuses, cloudformation.StackStatusCreateComplete) {
		return StatusCreateComplete, nil
	}

	if all(allStatuses, cloudformation.StackStatusDeleteComplete) {
		return StatusDeleteComplete, nil
	}

	if any(allStatuses, cloudformation.StackStatusDeleteInProgress) {
		return StatusDeleteInProgress, nil
	}

	// controlplane stack may be in complete state while nodegroup stacks are still in status not found
	if all(controlPlaneStatus, cloudformation.StackStatusCreateComplete, cloudformation.StackStatusCreateInProgress) &&
		all(nodeGroupStatuses, cloudformation.StackStatusCreateInProgress, string(StatusNotFound), cloudformation.StackStatusCreateComplete) {
		return StatusCreateInProgress, nil
	}

	return StatusNotFound, ErrorUnexpectedCloudFormationStatus(s.ObjFlat(statusMap))
}

func GetClusterState(awsClient *aws.Client, clusterConfig *clusterconfig.Config) (*ClusterState, error) {
	controlPlaneStackName := fmt.Sprintf(controlPlaneTemplate, clusterConfig.ClusterName)
	operatorStackName := fmt.Sprintf(operatorTemplate, clusterConfig.ClusterName)
	spotStackName := fmt.Sprintf(spotTemplate, clusterConfig.ClusterName)
	onDemandStackName := fmt.Sprintf(onDemandTemplate, clusterConfig.ClusterName)

	nodeGroupStackNames := []string{operatorStackName}
	if clusterConfig.Spot != nil && *clusterConfig.Spot {
		nodeGroupStackNames = append(nodeGroupStackNames, spotStackName)
		if clusterConfig.SpotConfig != nil && clusterConfig.SpotConfig.OnDemandBackup != nil && *clusterConfig.SpotConfig.OnDemandBackup {
			nodeGroupStackNames = append(nodeGroupStackNames, onDemandStackName)
		}
	} else {
		nodeGroupStackNames = append(nodeGroupStackNames, onDemandStackName)
	}

	stackSummaries, err := awsClient.ListEKSStacks(controlPlaneStackName, nodeGroupStackNames...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get cluster state from cloudformation")
	}

	statusMap := map[string]string{}
	statusMap[controlPlaneStackName] = getStatusFromSummaries(stackSummaries, controlPlaneStackName)

	for _, nodeGroupName := range nodeGroupStackNames {
		statusMap[nodeGroupName] = getStatusFromSummaries(stackSummaries, nodeGroupName)
	}

	status, err := getStatus(statusMap, controlPlaneStackName)
	if err != nil {
		return nil, err
	}

	return &ClusterState{
		ControlPlane: controlPlaneStackName,
		Nodegroups:   nodeGroupStackNames,
		StatusMap:    statusMap,
		Status:       status,
	}, nil
}

func getStatusFromSummaries(stackSummaries []*cloudformation.StackSummary, stackName string) string {
	for _, stackSummary := range stackSummaries {
		if *stackSummary.StackName == stackName {
			return *stackSummary.StackStatus
		}
	}

	return string(StatusNotFound)
}
