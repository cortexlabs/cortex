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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrClusterDoesNotExist    = "clusterstatus.cluster_does_not_exist"
	ErrClusterAlreadyExists   = "clusterstatus.cluster_already_exists"
	ErrUnexpectedClusterState = "clusterstatus.unexpected_cluster_state"
)

func ErrorClusterDoesNotExist(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterDoesNotExist,
		Message: fmt.Sprintf("there is no cluster named \"%s\" in %s", clusterName, region),
	})
}

func ErrorClusterAlreadyExists(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterAlreadyExists,
		Message: fmt.Sprintf("a cluster named \"%s\" already exists in %s", clusterName, region),
	})
}

func ErrorUnexpectedClusterState(stacks ClusterStacks) error {
	msg := fmt.Sprintf("cluster named \"%s\" in %s is in an unexpected state; if your CloudFormation stacks are updating, please wait for them to complete. Otherwise, run `cortex cluster down` to delete the cluster, or if that fails, delete the CloudFormation stacks directly from your AWS console: %s\n\n", stacks.clusterName, stacks.region, CloudFormationURL(stacks.clusterName, stacks.region))
	msg += fmt.Sprintf(stacks.TableString())
	return errors.WithStack(&errors.Error{
		Kind:     ErrUnexpectedClusterState,
		Message:  msg,
		Metadata: stacks,
	})
}
