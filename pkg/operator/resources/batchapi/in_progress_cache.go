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

package batchapi

import (
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

var (
	_inProgressFilePrefix = "in_progress_jobs"
)

func inProgressS3Key(jobKey spec.JobKey) string {
	return path.Join(_inProgressFilePrefix, jobKey.APIName, jobKey.ID)
}

func jobKeyFromInProgressS3Key(s3Key string) spec.JobKey {
	s3PathSplit := strings.Split(s3Key, "/")
	apiName := s3PathSplit[len(s3PathSplit)-2]
	jobID := s3PathSplit[len(s3PathSplit)-1]

	return spec.JobKey{APIName: apiName, ID: jobID}
}

func uploadInProgressFile(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.Cluster.Bucket, inProgressS3Key(jobKey))
	if err != nil {
		return err
	}
	return nil
}

func deleteInProgressFile(jobKey spec.JobKey) error {
	err := config.AWS.DeleteS3File(config.Cluster.Bucket, inProgressS3Key(jobKey))
	if err != nil {
		return err
	}
	return nil
}

func deleteAllInProgressFilesByAPI(apiName string) error {
	err := config.AWS.DeleteS3Prefix(config.Cluster.Bucket, path.Join(_inProgressFilePrefix, apiName), true)
	if err != nil {
		return err
	}
	return nil
}

func listAllInProgressJobKeys() ([]spec.JobKey, error) {
	s3Objects, err := config.AWS.ListS3Dir(config.Cluster.Bucket, _inProgressFilePrefix, false, nil)
	if err != nil {
		return nil, err
	}

	jobKeys := make([]spec.JobKey, 0, len(s3Objects))
	for _, obj := range s3Objects {
		jobKeys = append(jobKeys, jobKeyFromInProgressS3Key(*obj.Key))
	}

	return jobKeys, nil
}

func listAllInProgressJobKeysByAPI(apiName string) ([]spec.JobKey, error) {
	s3Objects, err := config.AWS.ListS3Dir(config.Cluster.Bucket, path.Join(_inProgressFilePrefix, apiName), false, nil)
	if err != nil {
		return nil, err
	}

	jobKeys := make([]spec.JobKey, 0, len(s3Objects))
	for _, obj := range s3Objects {
		jobKeys = append(jobKeys, jobKeyFromInProgressS3Key(*obj.Key))
	}

	return jobKeys, nil
}
