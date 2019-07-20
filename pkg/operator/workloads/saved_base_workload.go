/*
Copyright 2019 Cortex Labs, Inc.

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

package workloads

import (
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	ocontext "github.com/cortexlabs/cortex/pkg/operator/context"
)

func uploadBaseWorkload(baseWorkload *BaseWorkload) error {
	if isBaseWorkloadCached(baseWorkload) {
		return nil
	}

	key := ocontext.BaseWorkloadKey(baseWorkload.WorkloadID, baseWorkload.AppName)
	err := config.AWS.UploadJSONToS3(baseWorkload, key)
	if err != nil {
		return errors.Wrap(err, "upload base workload", baseWorkload.AppName, baseWorkload.WorkloadID)
	}
	cacheBaseWorkload(baseWorkload)
	return nil
}

func uploadBaseWorkloads(baseWorkloads []*BaseWorkload) error {
	fns := make([]func() error, len(baseWorkloads))
	for i, baseWorkload := range baseWorkloads {
		fns[i] = uploadBaseWorkloadFunc(baseWorkload)
	}
	return parallel.RunFirstErr(fns...)
}

func uploadBaseWorkloadsFromWorkloads(workloads []Workload) error {
	fns := make([]func() error, len(workloads))
	for i, workload := range workloads {
		fns[i] = uploadBaseWorkloadFunc(workload.GetBaseWorkloadPtr())
	}
	return parallel.RunFirstErr(fns...)
}

func uploadBaseWorkloadFunc(baseWorkload *BaseWorkload) func() error {
	return func() error {
		return uploadBaseWorkload(baseWorkload)
	}
}

func getSavedBaseWorkload(workloadID string, appName string) (*BaseWorkload, error) {
	if cachedBaseWorkload, ok := getCachedBaseWorkload(workloadID, appName); ok {
		return cachedBaseWorkload, nil
	}

	key := ocontext.BaseWorkloadKey(workloadID, appName)
	var baseWorkload BaseWorkload
	err := config.AWS.ReadJSONFromS3(&baseWorkload, key)
	if aws.IsNoSuchKeyErr(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "download base workload", appName, workloadID)
	}
	cacheBaseWorkload(&baseWorkload)
	return &baseWorkload, nil
}

func getSavedBaseWorkloads(workloadIDs []string, appName string) (map[string]*BaseWorkload, error) {
	baseWorkloads := make([]*BaseWorkload, len(workloadIDs))
	fns := make([]func() error, len(workloadIDs))
	i := 0
	for _, workloadID := range workloadIDs {
		fns[i] = getSavedBaseWorkloadFunc(workloadID, appName, baseWorkloads, i)
		i++
	}
	err := parallel.RunFirstErr(fns...)
	if err != nil {
		return nil, err
	}

	baseWorkloadMap := make(map[string]*BaseWorkload)
	for _, baseWorkload := range baseWorkloads {
		if baseWorkload != nil {
			baseWorkloadMap[baseWorkload.WorkloadID] = baseWorkload
		}
	}
	return baseWorkloadMap, err
}

func getSavedBaseWorkloadFunc(workloadID string, appName string, baseWorkloads []*BaseWorkload, i int) func() error {
	return func() error {
		baseWorkload, err := getSavedBaseWorkload(workloadID, appName)
		if err != nil {
			return err
		}
		baseWorkloads[i] = baseWorkload
		return nil
	}
}
