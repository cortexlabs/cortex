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

	"github.com/cortexlabs/cortex/pkg/proxy"
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
		maxQueueLength: maxConcurrency,
	}
}

func (a *apiActivator) Try(ctx context.Context, fn func() error) error {
	// FIXME: wait for an api pod to be ready, then try the request

	var fnErr error

	if err := a.breaker.Maybe(ctx, func() {
		fnErr = fn()
	}); err != nil {
		return err
	}

	return fnErr
}
