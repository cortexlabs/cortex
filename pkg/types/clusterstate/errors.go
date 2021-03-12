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
	ErrClusterDoesNotExist            = "clusterstatus.cluster_does_not_exist"
	ErrClusterUpInProgress            = "clusterstatus.cluster_up_in_progress"
	ErrClusterCreateFailed            = "clusterstatus.cluster_create_failed"
	ErrClusterCreateFailedTimeout     = "clusterstatus.cluster_create_failed_timeout"
	ErrClusterAlreadyCreated          = "clusterstatus.cluster_already_created"
	ErrClusterAlreadyUpdated          = "clusterstatus.cluster_already_updated"
	ErrClusterDownInProgress          = "clusterstatus.cluster_down_in_progress"
	ErrClusterAlreadyDeleted          = "clusterstatus.cluster_already_deleted"
	ErrClusterDeleteFailed            = "clusterstatus.cluster_delete_failed"
	ErrUnexpectedCloudFormationStatus = "clusterstatus.unexpected_cloud_formation_status"
)

func ErrorClusterDoesNotExist(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterDoesNotExist,
		Message: fmt.Sprintf("there is no cluster named \"%s\" in %s", clusterName, region),
	})
}

func ErrorClusterUpInProgress(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterUpInProgress,
		Message: fmt.Sprintf("creation of cluster \"%s\" in %s is currently in progress", clusterName, region),
	})
}

func ErrorClusterCreateFailed(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterCreateFailed,
		Message: fmt.Sprintf("cluster \"%s\" in %s could not be created; please view error information and delete the CloudFormation stacks directly from your AWS console: %s", clusterName, region, CloudFormationURL(clusterName, region)),
	})
}

func ErrorClusterCreateFailedTimeout(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterCreateFailedTimeout,
		Message: fmt.Sprintf("cluster \"%s\" in %s could not be created; please view error information and delete the CloudFormation stacks directly from your AWS console: %s", clusterName, region, CloudFormationURL(clusterName, region)),
	})
}

func ErrorClusterAlreadyCreated(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterAlreadyCreated,
		Message: fmt.Sprintf("a cluster named \"%s\" already exists in %s", clusterName, region),
	})
}

func ErrorClusterAlreadyUpdated(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterAlreadyUpdated,
		Message: fmt.Sprintf("a cluster named \"%s\" already created and updated in %s", clusterName, region),
	})
}

func ErrorClusterDownInProgress(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterDownInProgress,
		Message: fmt.Sprintf("deletion of cluster \"%s\" in %s is currently in progress", clusterName, region),
	})
}

func ErrorClusterAlreadyDeleted(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterAlreadyDeleted,
		Message: fmt.Sprintf("cluster \"%s\" in %s has already been deleted (or does not exist)", clusterName, region),
	})
}

func ErrorClusterDeleteFailed(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterDeleteFailed,
		Message: fmt.Sprintf("deletion of cluster \"%s\" in %s failed; please run `cortex cluster down` to delete the cluster, or if that fails, delete the CloudFormation stacks directly from your AWS console: %s", clusterName, region, CloudFormationURL(clusterName, region)),
	})
}

func ErrorUnexpectedCloudFormationStatus(clusterName string, region string, metadata interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:     ErrUnexpectedCloudFormationStatus,
		Message:  fmt.Sprintf("cluster named \"%s\" in %s is in an unexpected state; please run `cortex cluster down` to delete the cluster, or if that fails, delete the CloudFormation stacks directly from your AWS console: %s", clusterName, region, CloudFormationURL(clusterName, region)),
		Metadata: metadata,
	})
}
