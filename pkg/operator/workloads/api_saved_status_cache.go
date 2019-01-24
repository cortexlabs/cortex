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

	"github.com/cortexlabs/cortex/pkg/api/resource"
)

// appName -> map(resourceID -> map(workloadID -> *APISavedStatus))
var apiStatusCache = struct {
	m map[string]map[string]map[string]*resource.APISavedStatus
	sync.RWMutex
}{m: make(map[string]map[string]map[string]*resource.APISavedStatus)}

func getCachedAPISavedStatus(
	resourceID string, workloadID string, appName string,
) (*resource.APISavedStatus, bool) {

	apiStatusCache.RLock()
	defer apiStatusCache.RUnlock()
	if _, ok := apiStatusCache.m[appName]; ok {
		if _, ok := apiStatusCache.m[appName][resourceID]; ok {
			if savedStatus, ok := apiStatusCache.m[appName][resourceID][workloadID]; ok {
				return savedStatus.Copy(), true
			}
		}
	}
	return nil, false
}

func isAPISavedStatusCached(savedStatus *resource.APISavedStatus) bool {
	cachedStatus, _ := getCachedAPISavedStatus(
		savedStatus.ResourceID, savedStatus.WorkloadID, savedStatus.AppName)
	return resource.APISavedStatusPtrsEqual(savedStatus, cachedStatus)
}

func cacheAPISavedStatus(savedStatus *resource.APISavedStatus) {
	apiStatusCache.Lock()
	defer apiStatusCache.Unlock()
	setAPISavedStatusMap(savedStatus.ResourceID, savedStatus.WorkloadID,
		savedStatus.AppName, savedStatus.Copy(), apiStatusCache.m)
}

func cacheNilAPISavedStatus(resourceID string, workloadID string, appName string) {
	apiStatusCache.Lock()
	defer apiStatusCache.Unlock()
	setAPISavedStatusMap(resourceID, workloadID, appName, nil, apiStatusCache.m)
}

func apiSavedStatusesToMap(
	savedStatuses []*resource.APISavedStatus,
) map[string]map[string]map[string]*resource.APISavedStatus {

	savedStatusMap := make(map[string]map[string]map[string]*resource.APISavedStatus)
	for _, savedStatus := range savedStatuses {
		setAPISavedStatusMap(savedStatus.ResourceID, savedStatus.WorkloadID,
			savedStatus.AppName, savedStatus, savedStatusMap)
	}
	return savedStatusMap
}

func getStaleAPISavedStatuses(
	currentSavedStatuses []*resource.APISavedStatus,
) []*resource.APISavedStatus {

	currentSavedStatusMap := apiSavedStatusesToMap(currentSavedStatuses)

	apiStatusCache.RLock()
	defer apiStatusCache.RUnlock()

	var staleSavedStatuses []*resource.APISavedStatus
	for appName := range apiStatusCache.m {
		for resourceID, _ := range apiStatusCache.m[appName] {
			for workloadID, savedStatus := range apiStatusCache.m[appName][resourceID] {
				if _, ok := currentSavedStatusMap[appName]; !ok {
					staleSavedStatuses = append(staleSavedStatuses, savedStatus.Copy())
				} else if _, ok := currentSavedStatusMap[appName][resourceID]; !ok {
					staleSavedStatuses = append(staleSavedStatuses, savedStatus.Copy())
				} else if _, ok := currentSavedStatusMap[appName][resourceID][workloadID]; !ok {
					staleSavedStatuses = append(staleSavedStatuses, savedStatus.Copy())
				}
			}
		}
	}

	return staleSavedStatuses
}

func uncacheFinishedAPISavedStatuses(currentSavedStatuses []*resource.APISavedStatus) {
	currentSavedStatusMap := apiSavedStatusesToMap(currentSavedStatuses)

	apiStatusCache.Lock()
	defer apiStatusCache.Unlock()

	for appName := range apiStatusCache.m {
		if _, ok := currentSavedStatusMap[appName]; !ok {
			delete(apiStatusCache.m, appName)
			continue
		}
		for resourceID, _ := range apiStatusCache.m[appName] {
			if _, ok := currentSavedStatusMap[appName][resourceID]; !ok {
				delete(apiStatusCache.m[appName], resourceID)
				continue
			}
			for workloadID, _ := range apiStatusCache.m[appName][resourceID] {
				if _, ok := currentSavedStatusMap[appName][resourceID][workloadID]; !ok {
					delete(apiStatusCache.m[appName][resourceID], workloadID)
					continue
				}
			}
		}
	}
}

func setAPISavedStatusMap(
	resourceID string,
	workloadID string,
	appName string,
	savedStatus *resource.APISavedStatus,
	savedStatusMap map[string]map[string]map[string]*resource.APISavedStatus,
) {
	if _, ok := savedStatusMap[appName]; !ok {
		savedStatusMap[appName] = make(map[string]map[string]*resource.APISavedStatus)
	}
	if _, ok := savedStatusMap[appName][resourceID]; !ok {
		savedStatusMap[appName][resourceID] = make(map[string]*resource.APISavedStatus)
	}
	savedStatusMap[appName][resourceID][workloadID] = savedStatus
}
