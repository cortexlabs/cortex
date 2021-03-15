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

type Status string

const (
	StatusNotFound               Status = "not_found"
	StatusCreateInProgress       Status = "create_in_progress"
	StatusCreateFailed           Status = "create_failed"
	StatusCreateFailedTimedOut   Status = "create_failed_timed_out"
	StatusCreateComplete         Status = "create_complete"
	StatusUpdateComplete         Status = "update_complete"
	StatusUpdateRollbackComplete Status = "update_rollback_complete"
	StatusDeleteInProgress       Status = "delete_in_progress"
	StatusDeleteComplete         Status = "delete_complete"
	StatusDeleteFailed           Status = "delete_failed"
)

func AssertClusterStatus(clusterName string, region string, status Status, allowedStatuses ...Status) error {
	for _, allowedStatus := range allowedStatuses {
		if status == allowedStatus {
			return nil
		}
	}

	switch status {
	case StatusNotFound:
		return ErrorClusterDoesNotExist(clusterName, region)
	case StatusCreateInProgress:
		return ErrorClusterUpInProgress(clusterName, region)
	case StatusCreateFailed:
		return ErrorClusterCreateFailed(clusterName, region)
	case StatusCreateFailedTimedOut:
		return ErrorClusterCreateFailedTimeout(clusterName, region)
	case StatusCreateComplete:
		return ErrorClusterAlreadyCreated(clusterName, region)
	case StatusUpdateComplete:
		return ErrorClusterAlreadyUpdated(clusterName, region)
	case StatusUpdateRollbackComplete:
		return ErrorClusterAlreadyUpdated(clusterName, region)
	case StatusDeleteInProgress:
		return ErrorClusterDownInProgress(clusterName, region)
	case StatusDeleteComplete:
		return ErrorClusterAlreadyDeleted(clusterName, region)
	case StatusDeleteFailed:
		return ErrorClusterDeleteFailed(clusterName, region)
	default:
		return ErrorUnexpectedCloudFormationStatus(clusterName, region, status)
	}
}
