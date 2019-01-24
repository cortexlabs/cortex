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
)

// appName -> map(workloadID -> log prefix)
var logPrefixCache = struct {
	m map[string]map[string]string
	sync.RWMutex
}{m: make(map[string]map[string]string)}

func getCachedLogPrefix(workloadID string, appName string) string {
	logPrefixCache.RLock()
	defer logPrefixCache.RUnlock()
	if _, ok := logPrefixCache.m[appName]; ok {
		return logPrefixCache.m[appName][workloadID]
	}
	return ""
}

func isLogPrefixCached(logPrefix string, workloadID string, appName string) bool {
	return logPrefix == getCachedLogPrefix(workloadID, appName)
}

func cacheLogPrefix(logPrefix string, workloadID string, appName string) {
	logPrefixCache.Lock()
	defer logPrefixCache.Unlock()
	if _, ok := logPrefixCache.m[appName]; !ok {
		logPrefixCache.m[appName] = make(map[string]string)
	}
	logPrefixCache.m[appName][workloadID] = logPrefix
}

// currentWorkloadIDs: appName -> workload IDs to keep
func uncacheLogPrefixes(currentWorkloadIDs map[string]map[string]bool) {
	logPrefixCache.Lock()
	defer logPrefixCache.Unlock()
	for appName := range logPrefixCache.m {
		if _, ok := currentWorkloadIDs[appName]; !ok {
			delete(logPrefixCache.m, appName)
		} else {
			for workloadID := range logPrefixCache.m[appName] {
				if !currentWorkloadIDs[appName][workloadID] {
					delete(logPrefixCache.m[appName], workloadID)
				}
			}
		}
	}
}
