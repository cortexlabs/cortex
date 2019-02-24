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
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	ocontext "github.com/cortexlabs/cortex/pkg/operator/context"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

func uploadLatestWorkloadID(resourceID string, workloadID string, appName string) error {
	if isLatestWorkloadIDCached(resourceID, workloadID, appName) {
		return nil
	}

	key := ocontext.LatestWorkloadIDKey(resourceID, appName)
	err := aws.UploadStringToS3(workloadID, key)
	if err != nil {
		return errors.Wrap(err, "upload latest workload ID", appName, resourceID, workloadID)
	}
	cacheLatestWorkloadID(resourceID, workloadID, appName)
	return nil
}

func uploadLatestWorkloadIDs(resourceWorkloadIDs map[string]string, appName string) error {
	fns := make([]func() error, len(resourceWorkloadIDs))
	i := 0
	for resourceID, workloadID := range resourceWorkloadIDs {
		fns[i] = uploadLatestWorkloadIDFunc(resourceID, workloadID, appName)
		i++
	}
	return parallel.RunFirstErr(fns...)
}

func uploadLatestWorkloadIDFunc(resourceID string, workloadID string, appName string) func() error {
	return func() error {
		return uploadLatestWorkloadID(resourceID, workloadID, appName)
	}
}

func GetLatestWorkloadID(resourceID string, appName string) (string, error) {
	return getSavedLatestWorkloadID(resourceID, appName)
}

func getSavedLatestWorkloadID(resourceID string, appName string) (string, error) {
	if cachedWorkloadID, ok := getCachedLatestWorkloadID(resourceID, appName); ok {
		return cachedWorkloadID, nil
	}

	key := ocontext.LatestWorkloadIDKey(resourceID, appName)
	workloadID, err := aws.ReadStringFromS3(key)
	if aws.IsNoSuchKeyErr(err) {
		cacheEmptyLatestWorkloadID(resourceID, appName)
		return "", nil
	}
	if err != nil {
		return "", errors.Wrap(err, "download latest workload ID", appName, resourceID)
	}
	cacheLatestWorkloadID(resourceID, workloadID, appName)
	return workloadID, nil
}

func getSavedLatestWorkloadIDs(resourceIDs strset.Set, appName string) (map[string]string, error) {
	resourceIDList := resourceIDs.Slice()
	workloadIDList := make([]string, len(resourceIDList))
	fns := make([]func() error, len(resourceIDList))
	for i, resourceID := range resourceIDList {
		fns[i] = getSavedLatestWorkloadIDFunc(resourceID, appName, workloadIDList, i)
	}
	err := parallel.RunFirstErr(fns...)
	if err != nil {
		return nil, err
	}

	workloadIDMap := map[string]string{}
	for i := range workloadIDList {
		workloadIDMap[resourceIDList[i]] = workloadIDList[i]
	}
	return workloadIDMap, nil
}

func getSavedLatestWorkloadIDFunc(resourceID string, appName string, workloadIDs []string, i int) func() error {
	return func() error {
		workloadID, err := getSavedLatestWorkloadID(resourceID, appName)
		if err != nil {
			return err
		}
		workloadIDs[i] = workloadID
		return nil
	}
}
