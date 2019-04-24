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
	"sync"

	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

// appName -> map(resourceID -> *DataSavedStatus)
var dataStatusCache = struct {
	m map[string]map[string]*resource.DataSavedStatus
	sync.RWMutex
}{m: make(map[string]map[string]*resource.DataSavedStatus)}

func getCachedDataSavedStatus(resourceID string, workloadID string, appName string) (*resource.DataSavedStatus, bool) {
	dataStatusCache.RLock()
	defer dataStatusCache.RUnlock()
	if _, ok := dataStatusCache.m[appName]; ok {
		if savedStatus, ok := dataStatusCache.m[appName][resourceID]; ok {
			if savedStatus != nil && savedStatus.WorkloadID == workloadID {
				return savedStatus.Copy(), true
			}
		}
	}
	return nil, false
}

func isDataSavedStatusCached(savedStatus *resource.DataSavedStatus) bool {
	cachedStatus, _ := getCachedDataSavedStatus(savedStatus.ResourceID, savedStatus.WorkloadID, savedStatus.AppName)
	return resource.DataSavedStatusPtrsEqual(savedStatus, cachedStatus)
}

func cacheDataSavedStatus(savedStatus *resource.DataSavedStatus) {
	if savedStatus.End == nil {
		return // Don't cache non-ended statuses because they will be updated by the jobs
	}
	dataStatusCache.Lock()
	defer dataStatusCache.Unlock()
	if _, ok := dataStatusCache.m[savedStatus.AppName]; !ok {
		dataStatusCache.m[savedStatus.AppName] = make(map[string]*resource.DataSavedStatus)
	}
	dataStatusCache.m[savedStatus.AppName][savedStatus.ResourceID] = savedStatus.Copy()
}

func uncacheDataSavedStatuses(currentResourceWorkloadIDs map[string]string, appName string) {
	dataStatusCache.Lock()
	defer dataStatusCache.Unlock()

	if len(currentResourceWorkloadIDs) == 0 {
		delete(dataStatusCache.m, appName)
		return
	}

	if _, ok := dataStatusCache.m[appName]; !ok {
		return
	}

	for resourceID, savedStatus := range dataStatusCache.m[appName] {
		if savedStatus == nil {
			delete(dataStatusCache.m[appName], resourceID)
		} else if currentResourceWorkloadIDs[resourceID] != savedStatus.WorkloadID {
			delete(dataStatusCache.m[appName], resourceID)
		}
	}
}
