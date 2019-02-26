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

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

// appName -> map(resourceID -> latest workloadID)
var workloadIDCache = struct {
	m map[string]map[string]string
	sync.RWMutex
}{m: make(map[string]map[string]string)}

func getCachedLatestWorkloadID(resourceID string, appName string) (string, bool) {
	workloadIDCache.RLock()
	defer workloadIDCache.RUnlock()
	if _, ok := workloadIDCache.m[appName]; ok {
		if workloadID, ok := workloadIDCache.m[appName][resourceID]; ok {
			return workloadID, true
		}
	}
	return "", false
}

func isLatestWorkloadIDCached(resourceID string, workloadID string, appName string) bool {
	cachedWorkloadID, _ := getCachedLatestWorkloadID(resourceID, appName)
	return cachedWorkloadID == workloadID
}

func cacheLatestWorkloadID(resourceID string, workloadID string, appName string) {
	workloadIDCache.Lock()
	defer workloadIDCache.Unlock()
	if _, ok := workloadIDCache.m[appName]; !ok {
		workloadIDCache.m[appName] = make(map[string]string)
	}
	workloadIDCache.m[appName][resourceID] = workloadID
}

func cacheEmptyLatestWorkloadID(resourceID string, appName string) {
	workloadIDCache.Lock()
	defer workloadIDCache.Unlock()
	if _, ok := workloadIDCache.m[appName]; !ok {
		workloadIDCache.m[appName] = make(map[string]string)
	}
	workloadIDCache.m[appName][resourceID] = ""
}

func uncacheLatestWorkloadIDs(currentResourceIDs strset.Set, appName string) {
	workloadIDCache.Lock()
	defer workloadIDCache.Unlock()

	if len(currentResourceIDs) == 0 {
		delete(workloadIDCache.m, appName)
		return
	}

	if _, ok := workloadIDCache.m[appName]; !ok {
		return
	}

	for resourceID := range workloadIDCache.m[appName] {
		if !currentResourceIDs.Has(resourceID) {
			delete(workloadIDCache.m[appName], resourceID)
		}
	}
}
