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

package operator

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func DownloadAPISpec(apiName string, apiID string) (*spec.API, error) {
	bucketKey := spec.Key(apiName, apiID, config.ClusterName())
	var api spec.API
	if err := config.ReadJSONFromBucket(&api, bucketKey); err != nil {
		return nil, err
	}
	return &api, nil
}

func DownloadAPISpecs(apiNames []string, apiIDs []string) ([]spec.API, error) {
	apis := make([]spec.API, len(apiNames))
	fns := make([]func() error, len(apiNames))

	for i := range apiNames {
		localIdx := i
		fns[i] = func() error {
			api, err := DownloadAPISpec(apiNames[localIdx], apiIDs[localIdx])
			if err != nil {
				return err
			}
			apis[localIdx] = *api
			return nil
		}
	}

	if len(fns) > 0 {
		err := parallel.RunFirstErr(fns[0], fns[1:]...)
		if err != nil {
			return nil, err
		}
	}

	return apis, nil
}

func DownloadBatchJobSpec(jobKey spec.JobKey) (*spec.BatchJob, error) {
	jobSpec := spec.BatchJob{}
	err := config.ReadJSONFromBucket(&jobSpec, jobKey.SpecFilePath(config.ClusterName()))
	if err != nil {
		return nil, errors.Wrap(err, "unable to download job specification", jobKey.UserString())
	}
	return &jobSpec, nil
}

func DownloadTaskJobSpec(jobKey spec.JobKey) (*spec.TaskJob, error) {
	jobSpec := spec.TaskJob{}
	err := config.ReadJSONFromBucket(&jobSpec, jobKey.SpecFilePath(config.ClusterName()))
	if err != nil {
		return nil, errors.Wrap(err, "unable to download job specification", jobKey.UserString())
	}
	return &jobSpec, nil
}
