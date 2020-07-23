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

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

var (
	_inProgressFilePrefix = "in_progress_jobs"
)

func uploadInProgressFile(jobKey spec.JobKey) error {
	err := config.AWS.UploadJSONToS3("", config.Cluster.Bucket, path.Join(_inProgressFilePrefix, jobKey.APIName, jobKey.ID))
	if err != nil {
		return err
	}
	return nil
}

func deleteInProgressFile(jobKey spec.JobKey) error {
	err := config.AWS.DeleteS3Prefix(config.Cluster.Bucket, path.Join(_inProgressFilePrefix, jobKey.APIName, jobKey.ID), false)
	if err != nil {
		return err
	}
	return nil
}

func deleteAllInProgressFilesByAPI(apiName string) error {
	jobKeys, err := listAllInProgressJobsByAPI(apiName)
	if err != nil {
		return err
	}

	errs := []error{}
	for _, jobKey := range jobKeys {
		err := deleteInProgressFile(jobKey)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.FirstError(errs...)
}

func listAllInProgressJobs() ([]spec.JobKey, error) {
	s3Objects, err := config.AWS.ListS3Dir(config.Cluster.Bucket, _inProgressFilePrefix, false, nil)
	if err != nil {
		return nil, err
	}

	return extractJobIDSFromS3ObjectList(s3Objects), nil
}

func listAllInProgressJobsByAPI(apiName string) ([]spec.JobKey, error) {
	s3Objects, err := config.AWS.ListS3Dir(config.Cluster.Bucket, path.Join(_inProgressFilePrefix, apiName), false, nil)
	if err != nil {
		return nil, err
	}

	return extractJobIDSFromS3ObjectList(s3Objects), nil
}

func extractJobIDSFromS3ObjectList(s3Objects []*s3.Object) []spec.JobKey {
	jobIDs := make([]spec.JobKey, 0, len(s3Objects))
	for _, obj := range s3Objects {
		s3PathSplit := strings.Split(*obj.Key, "/")
		jobIDs = append(jobIDs, spec.JobKey{APIName: s3PathSplit[len(s3PathSplit)-2], ID: s3PathSplit[len(s3PathSplit)-1]})
	}

	return jobIDs
}
