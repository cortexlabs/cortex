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

package clusterstatus

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

const (
	controlPlaneTemplate = "eksctl-%s-cluster"
	operatorTemplate     = "eksctl-%s-nodegroup-ng-cortex-operator"
	spotTemplate         = "eksctl-%s-nodegroup-ng-cortex-worker-spot"
	onDemandTemplate     = "eksctl-%s-nodegroup-ng-cortex-worker-on-demand"
)

func GetStatus(awsClient *aws.Client, clusterConfig *clusterconfig.Config) (Status, error) {
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

	stackSummaries, err := awsClient.ListStacks(controlPlaneStackName, nodeGroupStackNames...)
	if err != nil {
		return StatusNotFound, errors.Wrap(err, "cluster status")
	}

	statusMap := map[string]string{}
	statusMap[controlPlaneStackName] = getStatusFromSummaries(stackSummaries, controlPlaneStackName)

	for _, nodeGroupName := range nodeGroupStackNames {
		statusMap[nodeGroupName] = getStatusFromSummaries(stackSummaries, nodeGroupName)
	}

	debug.Pp(statusMap)
	if statusMap[controlPlaneStackName] == string(StatusNotFound) {
		return StatusNotFound, nil
	}

	if any(statusMap, cloudformation.StackStatusCreateFailed) {
		return StatusCreateFailed, nil
	}

	if any(statusMap, cloudformation.StackStatusDeleteFailed) {
		return StatusDeleteFailed, nil
	}

	if all(statusMap, cloudformation.StackStatusCreateComplete) {
		return StatusCreateComplete, nil
	}

	if all(statusMap, cloudformation.StackStatusDeleteComplete) {
		return StatusDeleteComplete, nil
	}

	if all(statusMap, cloudformation.StackStatusDeleteComplete, cloudformation.StackStatusDeleteInProgress) {
		return StatusDeleteInProgress, nil
	}

	if all(statusMap, cloudformation.StackStatusCreateComplete) {
		return StatusCreateComplete, nil
	}

	if all(statusMap, cloudformation.StackStatusCreateComplete, cloudformation.StackStatusCreateInProgress) {
		return StatusCreateInProgress, nil
	}

	return StatusNotFound, ErrorUnsupportedCloudFormationStatus(statusMap)
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

func getStatusFromSummaries(stackSummaries []*cloudformation.StackSummary, stackName string) string {
	for _, stackSummary := range stackSummaries {
		if *stackSummary.StackName == stackName {
			return *stackSummary.StackStatus
		}
	}

	return string(StatusNotFound)
}
