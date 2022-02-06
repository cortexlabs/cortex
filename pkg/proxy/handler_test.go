/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Modifications Copyright 2022 Cortex Labs, Inc.
*/

package proxy_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cortexlabs/cortex/pkg/proxy"
	"github.com/stretchr/testify/require"
)

const (
	userContainerHost = "http://user-container.cortex.dev"
)

func TestProxyHandlerQueueFull(t *testing.T) {
	// This test sends three requests of which one should fail immediately as the queue
	// saturates.
	resp := make(chan struct{})
	blockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-resp
	})

	breaker := proxy.NewBreaker(
		proxy.BreakerParams{
			QueueDepth:      1,
			MaxConcurrency:  1,
			InitialCapacity: 1,
		},
	)

	h := proxy.Handler(breaker, blockHandler)

	req := httptest.NewRequest(http.MethodGet, userContainerHost, nil)
	resps := make(chan *httptest.ResponseRecorder)
	for i := 0; i < 3; i++ {
		go func() {
			rec := httptest.NewRecorder()
			h(rec, req)
			resps <- rec
		}()
	}

	// One of the three requests fails and it should be the first we see since the others
	// are still held by the resp channel.
	failure := <-resps
	require.Equal(t, http.StatusServiceUnavailable, failure.Code)
	require.True(t, strings.Contains(failure.Body.String(), "pending request queue full"))

	// Allow the remaining requests to pass.
	close(resp)
	for i := 0; i < 2; i++ {
		res := <-resps
		require.Equal(t, http.StatusOK, res.Code)
	}
}

func TestProxyHandlerBreakerTimeout(t *testing.T) {
	// This test sends a request which will take a long time to complete.
	// Then another one with a very short context timeout.
	// Verifies that the second one fails with timeout.
	seen := make(chan struct{})
	resp := make(chan struct{})
	defer close(resp) // Allow all requests to pass through.
	blockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- struct{}{}
		<-resp
	})
	breaker := proxy.NewBreaker(proxy.BreakerParams{
		QueueDepth: 1, MaxConcurrency: 1, InitialCapacity: 1,
	})
	h := proxy.Handler(breaker, blockHandler)

	go func() {
		h(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, userContainerHost, nil))
	}()

	// Wait until the first request has entered the handler.
	<-seen

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, userContainerHost, nil).WithContext(ctx))

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.True(t, strings.Contains(rec.Body.String(), context.DeadlineExceeded.Error()))
}
