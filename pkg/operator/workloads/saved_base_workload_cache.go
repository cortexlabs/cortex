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

// appName -> map(workloadID -> *BaseWorkload)
var baseWorkloadCache = struct {
	m map[string]map[string]*BaseWorkload
	sync.RWMutex
}{m: make(map[string]map[string]*BaseWorkload)}

func getCachedBaseWorkload(workloadID string, appName string) (*BaseWorkload, bool) {
	baseWorkloadCache.RLock()
	defer baseWorkloadCache.RUnlock()
	if _, ok := baseWorkloadCache.m[appName]; ok {
		if baseWorkload, ok := baseWorkloadCache.m[appName][workloadID]; ok {
			if baseWorkload != nil {
				return baseWorkload.Copy(), true
			}
		}
	}
	return nil, false
}

func isBaseWorkloadCached(baseWorkload *BaseWorkload) bool {
	cachedBaseWorkload, _ := getCachedBaseWorkload(baseWorkload.WorkloadID, baseWorkload.AppName)
	return BaseWorkloadPtrsEqual(baseWorkload, cachedBaseWorkload)
}

func cacheBaseWorkload(baseWorkload *BaseWorkload) {
	baseWorkloadCache.Lock()
	defer baseWorkloadCache.Unlock()
	if _, ok := baseWorkloadCache.m[baseWorkload.AppName]; !ok {
		baseWorkloadCache.m[baseWorkload.AppName] = make(map[string]*BaseWorkload)
	}
	baseWorkloadCache.m[baseWorkload.AppName][baseWorkload.WorkloadID] = baseWorkload.Copy()
}

// app name -> workload IDs
func uncacheBaseWorkloads(currentWorkloadIDs map[string]strset.Set) {
	baseWorkloadCache.Lock()
	defer baseWorkloadCache.Unlock()
	for appName := range baseWorkloadCache.m {
		if _, ok := currentWorkloadIDs[appName]; !ok {
			delete(baseWorkloadCache.m, appName)
		} else {
			for workloadID := range baseWorkloadCache.m[appName] {
				if !currentWorkloadIDs[appName].Has(workloadID) {
					delete(baseWorkloadCache.m[appName], workloadID)
				}
			}
		}
	}
}
