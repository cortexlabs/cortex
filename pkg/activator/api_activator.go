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

package activator

import (
	"context"
	"sync"

	"github.com/cortexlabs/cortex/pkg/proxy"
	kapps "k8s.io/api/apps/v1"
)

type apiActivator struct {
	apiName        string
	breaker        *proxy.Breaker
	maxConcurrency int
	maxQueueLength int
}

func newAPIActivator(apiName string, maxQueueLength, maxConcurrency int) *apiActivator {
	breaker := proxy.NewBreaker(proxy.BreakerParams{
		QueueDepth:      maxQueueLength,
		MaxConcurrency:  maxConcurrency,
		InitialCapacity: maxConcurrency,
	})

	return &apiActivator{
		apiName:        apiName,
		breaker:        breaker,
		maxConcurrency: maxConcurrency,
		maxQueueLength: maxQueueLength,
	}
}

func (a *apiActivator) try(ctx context.Context, fn func() error, tracker *readinessTracker) error {
	var execErr error

	if err := a.breaker.Maybe(ctx, func() {
		if !tracker.IsReady() {
		loop:
			for {
				select {
				case <-ctx.Done():
					execErr = ctx.Err()
					return
				case <-tracker.Wait():
					break loop
				}
			}
		}
		execErr = fn()
	}); err != nil {
		return err
	}

	return execErr
}

func (a *apiActivator) inFlight() int64 {
	return a.breaker.InFlight()
}

type readinessTracker struct {
	mux   sync.RWMutex
	c     chan bool
	ready bool
}

func newReadinessTracker() *readinessTracker {
	return &readinessTracker{
		c: make(chan bool),
	}
}

func (t *readinessTracker) Update(deployment *kapps.Deployment) {
	t.mux.Lock()
	defer t.mux.Unlock()

	// if changing from unready to ready state, wake up waiting goroutines
	if !t.ready && deployment.Status.ReadyReplicas > 0 {
		close(t.c)            // wake up all of the goroutines waiting on this channel
		t.c = make(chan bool) // make a new channel ready to be used
	}
	t.ready = deployment.Status.ReadyReplicas > 0
}

func (t *readinessTracker) IsReady() bool {
	t.mux.RLock()
	defer t.mux.RUnlock()
	return t.ready
}

func (t *readinessTracker) Wait() chan bool {
	t.mux.RLock()
	defer t.mux.RUnlock()
	return t.c
}
