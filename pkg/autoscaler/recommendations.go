/*
Copyright 2022 Cortex Labs, Inc.

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

package autoscaler

import (
	"math"
	"sync"
	"time"
)

type recommendations struct {
	mux      sync.RWMutex
	timeline map[time.Time]int32
}

func newRecommendations() *recommendations {
	return &recommendations{
		timeline: make(map[time.Time]int32),
	}
}

func (r *recommendations) add(rec int32) {
	r.mux.Lock()
	defer r.mux.Unlock()

	r.timeline[time.Now()] = rec
}

func (r *recommendations) deleteOlderThan(period time.Duration) {
	r.mux.Lock()
	defer r.mux.Unlock()

	for t := range r.timeline {
		if time.Since(t) > period {
			delete(r.timeline, t)
		}
	}
}

// Returns nil if no recommendations in the period
func (r *recommendations) maxSince(period time.Duration) *int32 {
	r.mux.RLock()
	defer r.mux.RUnlock()

	max := int32(math.MinInt32)
	foundRecommendation := false

	for t, rec := range r.timeline {
		if time.Since(t) <= period && rec > max {
			max = rec
			foundRecommendation = true
		}
	}

	if !foundRecommendation {
		return nil
	}

	return &max
}

// Returns nil if no recommendations in the period
func (r *recommendations) minSince(period time.Duration) *int32 {
	r.mux.RLock()
	defer r.mux.RUnlock()

	min := int32(math.MaxInt32)
	foundRecommendation := false

	for t, rec := range r.timeline {
		if time.Since(t) <= period && rec < min {
			min = rec
			foundRecommendation = true
		}
	}

	if !foundRecommendation {
		return nil
	}

	return &min
}
