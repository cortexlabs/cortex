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
	client       *aws.Client
	StatusMap    map[string]string // cloudformation stackname to cloudformation stackstatus
	ControlPlane string
	Nodegroups   []string
	Status       Status
}

func any(statusMap map[string]string, statuses ...string) bool {
	statusSet := strset.New(statuses...)
	for _, stackStatus := range statusMap {
		if statusSet.Has(stackStatus) {
			return true
		}
	}

	return false
}

func all(statusMap map[string]string, statuses ...string) bool {
	statusSet := strset.New(statuses...)
	for _, stackStatus := range statusMap {
		if !statusSet.Has(stackStatus) {
			return false
		}
	}

	return true
}

func (cs ClusterState) FlatString() string {
	return s.ObjFlat(cs.StatusMap)
}

func (cs ClusterState) PrettyString() string {
	var items table.KeyValuePairs
	items.Add(cs.ControlPlane, cs.StatusMap[cs.ControlPlane])

	for _, nodeGroupName := range cs.Nodegroups {
		items.Add(nodeGroupName, cs.StatusMap[nodeGroupName])
	}
	return items.String()
}

func (cs ClusterState) getStatus() (Status, error) {
	if cs.StatusMap[cs.ControlPlane] == string(StatusNotFound) {
		return StatusNotFound, nil
	}

	if any(cs.StatusMap, cloudformation.StackStatusCreateFailed) {
		return StatusCreateFailed, nil
	}

	if any(cs.StatusMap, cloudformation.StackStatusDeleteFailed) {
		return StatusDeleteFailed, nil
	}

	if all(cs.StatusMap, cloudformation.StackStatusCreateComplete) {
		return StatusCreateComplete, nil
	}

	if all(cs.StatusMap, cloudformation.StackStatusDeleteComplete) {
		return StatusDeleteComplete, nil
	}

	if any(cs.StatusMap, cloudformation.StackStatusDeleteInProgress) {
		return StatusDeleteInProgress, nil
	}

	if all(cs.StatusMap, cloudformation.StackStatusCreateInProgress, string(StatusNotFound), cloudformation.StackStatusCreateComplete) {
		return StatusCreateInProgress, nil
	}

	return StatusNotFound, ErrorUnexpectedCloudFormationStatus(cs.FlatString())
}

func (cs *ClusterState) GetStatus() (Status, error) {
	stackSummaries, err := cs.client.ListEKSStacks(cs.ControlPlane, cs.Nodegroups...)
	if err != nil {
		return StatusNotFound, errors.Wrap(err, "unable to get cluster state from cloudformation")
	}

	statusMap := map[string]string{}
	statusMap[cs.ControlPlane] = getStatusFromSummaries(stackSummaries, cs.ControlPlane)

	for _, nodeGroupName := range cs.Nodegroups {
		statusMap[nodeGroupName] = getStatusFromSummaries(stackSummaries, nodeGroupName)
	}

	cs.StatusMap = statusMap

	return cs.getStatus()
}

func GetClusterState(awsClient *aws.Client, clusterConfig *clusterconfig.Config) ClusterState {
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

	clusterstate := ClusterState{
		client:       awsClient,
		ControlPlane: controlPlaneStackName,
		Nodegroups:   nodeGroupStackNames,
	}

	return clusterstate
}

func getStatusFromSummaries(stackSummaries []*cloudformation.StackSummary, stackName string) string {
	for _, stackSummary := range stackSummaries {
		if *stackSummary.StackName == stackName {
			return *stackSummary.StackStatus
		}
	}

	return string(StatusNotFound)
}
