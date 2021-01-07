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

package telemetry

import (
	"sync"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

const (
	_initialCoolDownPeriod = 5 * time.Second  // The initial interval to wait before sending duplicate errors
	_maxCoolDownPeriod     = 24 * time.Hour   // The longest interval to wait before sending duplicate errors
	_coolDownFactor        = 2                // Factor by which to increase cool down period after each error report
	_cacheEvictionPeriod   = 24 * time.Hour   // Duration of not seeing an error after which seeing it again resets the cooldown period
	_cacheCleanupInterval  = 10 * time.Minute // Minimum time to wait before checking error cache for errors to evict
)

var _errorCache = struct {
	m           map[string]*errorStatus
	lastCleanup time.Time
	sync.RWMutex
}{m: make(map[string]*errorStatus)}

type errorStatus struct {
	LastReportTime time.Time
	LastSeenTime   time.Time
	CoolDownPeriod time.Duration
}

func shouldBlock(err error, backoffMode BackoffMode) bool {
	if backoffMode == NoBackoff {
		return false
	}

	errMsg := errors.MessageFirstLine(err)
	now := time.Now()

	if backoffMode == BackoffAnyMessages {
		errMsg = "<msg>"
	}

	defer func() {
		go cleanupCache()
	}()

	_errorCache.Lock()
	defer _errorCache.Unlock()

	errStatus, ok := _errorCache.m[errMsg]

	if !ok || time.Since(errStatus.LastSeenTime) > _cacheEvictionPeriod {
		_errorCache.m[errMsg] = &errorStatus{
			LastReportTime: now,
			LastSeenTime:   now,
			CoolDownPeriod: _initialCoolDownPeriod,
		}
		return false
	}

	if time.Since(errStatus.LastReportTime) > errStatus.CoolDownPeriod {
		errStatus.LastSeenTime = now
		errStatus.LastReportTime = now
		errStatus.CoolDownPeriod = time.Duration(float64(errStatus.CoolDownPeriod.Nanoseconds())*_coolDownFactor) * time.Nanosecond
		if errStatus.CoolDownPeriod > _maxCoolDownPeriod {
			errStatus.CoolDownPeriod = _maxCoolDownPeriod
		}
		return false
	}

	errStatus.LastSeenTime = now
	return true
}

func cleanupCache() {
	staleErrorMessages := findStaleCachedErrors()

	if len(staleErrorMessages) == 0 {
		return
	}

	_errorCache.Lock()
	defer _errorCache.Unlock()
	for errMsg := range staleErrorMessages {
		delete(_errorCache.m, errMsg)
	}
}

func findStaleCachedErrors() strset.Set {
	_errorCache.RLock()
	defer _errorCache.RUnlock()

	if time.Since(_errorCache.lastCleanup) < _cacheCleanupInterval {
		return nil
	}

	_errorCache.lastCleanup = time.Now()

	staleErrorMessages := strset.New()
	for errMsg, errStatus := range _errorCache.m {
		if time.Since(errStatus.LastSeenTime) > _cacheEvictionPeriod {
			staleErrorMessages.Add(errMsg)
		}
	}

	return staleErrorMessages
}
