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

package job

import (
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func ListAllInProgressJobKeysByAPI(kind userconfig.Kind, apiName string) ([]spec.JobKey, error) {
	return listAllInProgressJobKeysByAPI(kind, &apiName)
}

func ListAllInProgressJobKeys(kind userconfig.Kind) ([]spec.JobKey, error) {
	return listAllInProgressJobKeysByAPI(kind, nil)
}

func DeleteInProgressFile(jobKey spec.JobKey) error {
	err := config.AWS.DeleteS3File(config.ClusterConfig.Bucket, inProgressKey(jobKey))
	if err != nil {
		return err
	}
	return nil
}

func DeleteAllInProgressFilesByAPI(kind userconfig.Kind, apiName string) error {
	err := config.AWS.DeleteS3Prefix(config.ClusterConfig.Bucket, allInProgressForAPIKey(kind, apiName), true)
	if err != nil {
		return err
	}
	return nil
}

func listAllInProgressJobKeysByAPI(kind userconfig.Kind, apiName *string) ([]spec.JobKey, error) {
	_, ok := _jobKinds[kind]
	if !ok {
		return nil, ErrorInvalidJobKind(kind)
	}

	var jobPath string
	if apiName != nil {
		jobPath = allInProgressForAPIKey(kind, *apiName)
	} else {
		jobPath = allInProgressKey(kind)
	}

	s3Objects, err := config.AWS.ListS3Dir(config.ClusterConfig.Bucket, jobPath, false, nil, nil)
	if err != nil {
		return nil, err
	}

	jobKeys := make([]spec.JobKey, 0, len(s3Objects))
	for _, obj := range s3Objects {
		if obj != nil {
			jobKeys = append(jobKeys, jobKeyFromInProgressKey(*obj.Key))
		}
	}
	return jobKeys, nil
}

func uploadInProgressFile(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, inProgressKey(jobKey))
	if err != nil {
		return err
	}
	return nil
}

// e.g. <cluster_uid>/jobs/<job_api_kind>/in_progress
func allInProgressKey(kind userconfig.Kind) string {
	return path.Join(
		config.ClusterConfig.ClusterUID, _jobsPrefix, kind.String(), _inProgressFilePrefix,
	)
}

// e.g. <cluster_uid>/jobs/<job_api_kind>/in_progress/<api_name>
func allInProgressForAPIKey(kind userconfig.Kind, apiName string) string {
	return path.Join(allInProgressKey(kind), apiName)
}

// e.g. <cluster_uid>/jobs/<job_api_kind>/in_progress/<api_name>/<job_id>
func inProgressKey(jobKey spec.JobKey) string {
	return path.Join(allInProgressForAPIKey(jobKey.Kind, jobKey.APIName), jobKey.ID)
}

func jobKeyFromInProgressKey(s3Key string) spec.JobKey {
	pathSplit := strings.Split(s3Key, "/")

	kind := pathSplit[len(pathSplit)-4]
	apiName := pathSplit[len(pathSplit)-2]
	jobID := pathSplit[len(pathSplit)-1]

	return spec.JobKey{APIName: apiName, ID: jobID, Kind: userconfig.KindFromString(kind)}
}
