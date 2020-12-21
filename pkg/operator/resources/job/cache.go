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

package job

import (
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func ListAllInProgressJobKeysByAPI(apiName string, kind userconfig.Kind) ([]spec.JobKey, error) {
	return listAllInProgressJobKeysByAPI(&apiName, kind)
}

func ListAllInProgressJobKeys(kind userconfig.Kind) ([]spec.JobKey, error) {
	return listAllInProgressJobKeysByAPI(nil, kind)
}

func DeleteInProgressFile(jobKey spec.JobKey) error {
	err := config.AWS.DeleteS3File(config.Cluster.Bucket, inProgressS3Key(jobKey))
	if err != nil {
		return err
	}
	return nil
}

func DeleteAllInProgressFilesByAPI(apiName string, kind userconfig.Kind) error {
	err := config.AWS.DeleteS3Prefix(config.Cluster.Bucket, allInProgressS3Key(apiName, kind), true)
	if err != nil {
		return err
	}
	return nil
}

func listAllInProgressJobKeysByAPI(apiName *string, kind userconfig.Kind) ([]spec.JobKey, error) {
	_, ok := _jobKinds[kind]
	if !ok {
		return nil, ErrorInvalidJobKind(kind)
	}

	var jobPath string
	if apiName != nil {
		jobPath = path.Join(config.Cluster.ClusterName, _inProgressFilePrefix, *apiName)
	} else {
		jobPath = path.Join(config.Cluster.ClusterName, _inProgressFilePrefix)
	}

	s3Objects, err := config.AWS.ListS3Dir(config.Cluster.Bucket, jobPath, false, nil)
	if err != nil {
		return nil, err
	}

	jobKeys := make([]spec.JobKey, 0, len(s3Objects))
	for _, obj := range s3Objects {
		jobKeys = append(jobKeys, jobKeyFromInProgressS3Key(*obj.Key))
	}

	return jobKeys, nil
}

func uploadInProgressFile(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.Cluster.Bucket, inProgressS3Key(jobKey))
	if err != nil {
		return err
	}
	return nil
}

// e.g. <cluster_name>/jobs/<job_api_kind>/in_progress/<api_name>
func allInProgressS3Key(apiName string, kind userconfig.Kind) string {
	return path.Join(
		config.Cluster.ClusterName, _jobsPrefix, kind.String(), _inProgressFilePrefix, apiName,
	)
}

// e.g. <cluster_name>/jobs/<job_api_kind>/in_progress/<api_name>/<job_id>
func inProgressS3Key(jobKey spec.JobKey) string {
	return path.Join(
		allInProgressS3Key(jobKey.APIName, jobKey.Kind), jobKey.ID,
	)
}

func jobKeyFromInProgressS3Key(s3Key string) spec.JobKey {
	s3PathSplit := strings.Split(s3Key, "/")

	kind := s3PathSplit[len(s3PathSplit)-4]
	apiName := s3PathSplit[len(s3PathSplit)-2]
	jobID := s3PathSplit[len(s3PathSplit)-1]

	return spec.JobKey{APIName: apiName, ID: jobID, Kind: userconfig.KindFromString(kind)}
}
