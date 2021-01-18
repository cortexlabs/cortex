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

package job

import (
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/operator/config"
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
	err := config.DeleteBucketFile(inProgressKey(jobKey))
	if err != nil {
		return err
	}
	return nil
}

func DeleteAllInProgressFilesByAPI(kind userconfig.Kind, apiName string) error {
	err := config.DeleteBucketPrefix(allInProgressForAPIKey(kind, apiName), true)
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

	gcsObjects, s3Objects, err := config.ListBucketDir(jobPath, nil)
	if err != nil {
		return nil, err
	}

	if len(gcsObjects) > 0 {
		jobKeys := make([]spec.JobKey, 0, len(gcsObjects))
		for _, obj := range gcsObjects {
			if obj != nil {
				jobKeys = append(jobKeys, jobKeyFromInProgressKey(obj.Name))
			}
		}
		return jobKeys, nil
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
	err := config.UploadStringToBucket("", inProgressKey(jobKey))
	if err != nil {
		return err
	}
	return nil
}

// e.g. <cluster_name>/jobs/<job_api_kind>/in_progress
func allInProgressKey(kind userconfig.Kind) string {
	return path.Join(
		config.ClusterName(), _jobsPrefix, kind.String(), _inProgressFilePrefix,
	)
}

// e.g. <cluster_name>/jobs/<job_api_kind>/in_progress/<api_name>
func allInProgressForAPIKey(kind userconfig.Kind, apiName string) string {
	return path.Join(allInProgressKey(kind), apiName)
}

// e.g. <cluster_name>/jobs/<job_api_kind>/in_progress/<api_name>/<job_id>
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
