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
	"time"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	ocontext "github.com/cortexlabs/cortex/pkg/operator/context"
)

func uploadDataSavedStatus(savedStatus *resource.DataSavedStatus) error {
	if isDataSavedStatusCached(savedStatus) {
		return nil
	}

	key := ocontext.StatusKey(savedStatus.ResourceID, savedStatus.WorkloadID, savedStatus.AppName)
	err := aws.UploadJSONToS3(savedStatus, key)
	if err != nil {
		return errors.Wrap(err, "upload data saved status", savedStatus.AppName, savedStatus.ResourceID, savedStatus.WorkloadID)
	}
	cacheDataSavedStatus(savedStatus)
	return nil
}

func uploadDataSavedStatuses(savedStatuses []*resource.DataSavedStatus) error {
	fns := make([]func() error, len(savedStatuses))
	for i, savedStatus := range savedStatuses {
		fns[i] = uploadDataSavedStatusFunc(savedStatus)
	}
	return parallel.RunFirstErr(fns...)
}

func uploadDataSavedStatusFunc(savedStatus *resource.DataSavedStatus) func() error {
	return func() error {
		return uploadDataSavedStatus(savedStatus)
	}
}

func getDataSavedStatus(resourceID string, workloadID string, appName string) (*resource.DataSavedStatus, error) {
	if cachedSavedStatus, ok := getCachedDataSavedStatus(resourceID, workloadID, appName); ok {
		return cachedSavedStatus, nil
	}

	key := ocontext.StatusKey(resourceID, workloadID, appName)
	var savedStatus resource.DataSavedStatus
	err := aws.ReadJSONFromS3(&savedStatus, key)
	if aws.IsNoSuchKeyErr(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "download data saved status", appName, resourceID, workloadID)
	}
	cacheDataSavedStatus(&savedStatus)
	return &savedStatus, nil
}

func getDataSavedStatuses(resourceWorkloadIDs map[string]string, appName string) (map[string]*resource.DataSavedStatus, error) {
	savedStatuses := make([]*resource.DataSavedStatus, len(resourceWorkloadIDs))
	fns := make([]func() error, len(resourceWorkloadIDs))
	i := 0
	for resourceID, workloadID := range resourceWorkloadIDs {
		fns[i] = getDataSavedStatusFunc(resourceID, workloadID, appName, savedStatuses, i)
		i++
	}
	err := parallel.RunFirstErr(fns...)
	if err != nil {
		return nil, err
	}

	savedStatusMap := map[string]*resource.DataSavedStatus{}
	for _, savedStatus := range savedStatuses {
		if savedStatus != nil {
			savedStatusMap[savedStatus.ResourceID] = savedStatus
		}
	}
	for resourceID := range resourceWorkloadIDs {
		if _, ok := savedStatusMap[resourceID]; !ok {
			savedStatusMap[resourceID] = nil
		}
	}
	return savedStatusMap, err
}

func getDataSavedStatusFunc(resourceID string, workloadID string, appName string, savedStatuses []*resource.DataSavedStatus, i int) func() error {
	return func() error {
		savedStatus, err := getDataSavedStatus(resourceID, workloadID, appName)
		if err != nil {
			return err
		}
		savedStatuses[i] = savedStatus
		return nil
	}
}

func updateKilledDataSavedStatuses(ctx *context.Context) error {
	resourceWorkloadIDs := ctx.DataResourceWorkloadIDs()
	savedStatuses, err := getDataSavedStatuses(resourceWorkloadIDs, ctx.App.Name)
	if err != nil {
		return err
	}

	var savedStatusesToUpdate []*resource.DataSavedStatus
	for _, savedStatus := range savedStatuses {
		if savedStatus != nil && savedStatus.Start != nil && savedStatus.End == nil {
			savedStatus.End = pointer.Time(time.Now())
			savedStatus.ExitCode = resource.ExitCodeDataKilled
			savedStatusesToUpdate = append(savedStatusesToUpdate, savedStatus)
		}
	}

	err = uploadDataSavedStatuses(savedStatusesToUpdate)
	if err != nil {
		return err
	}
	return nil
}
