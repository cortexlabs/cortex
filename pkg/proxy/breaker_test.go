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

package proxy

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

const (
	// semAcquireTimeout is a timeout for tests that try to acquire
	// a token of a semaphore.
	semAcquireTimeout = 10 * time.Second

	// semNoChangeTimeout is some additional wait time after a number
	// of acquires is reached to assert that no more acquires get through.
	semNoChangeTimeout = 50 * time.Millisecond
)

func TestBreakerInvalidConstructor(t *testing.T) {
	tests := []struct {
		name    string
		options BreakerParams
	}{{
		name:    "QueueDepth = 0",
		options: BreakerParams{QueueDepth: 0, MaxConcurrency: 1, InitialCapacity: 1},
	}, {
		name:    "MaxConcurrency negative",
		options: BreakerParams{QueueDepth: 1, MaxConcurrency: -1, InitialCapacity: 1},
	}, {
		name:    "InitialCapacity negative",
		options: BreakerParams{QueueDepth: 1, MaxConcurrency: 1, InitialCapacity: -1},
	}, {
		name:    "InitialCapacity out-of-bounds",
		options: BreakerParams{QueueDepth: 1, MaxConcurrency: 5, InitialCapacity: 6},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected a panic but the code didn't panic.")
				}
			}()

			NewBreaker(test.options)
		})
	}
}

func TestBreakerReserveOverload(t *testing.T) {
	params := BreakerParams{QueueDepth: 1, MaxConcurrency: 1, InitialCapacity: 1}
	b := NewBreaker(params) // Breaker capacity = 2
	cb1, rr := b.Reserve(context.Background())
	if !rr {
		t.Fatal("Reserve1 failed")
	}
	_, rr = b.Reserve(context.Background())
	if rr {
		t.Fatal("Reserve2 was an unexpected success.")
	}
	// Release a slot.
	cb1()
	// And reserve it again.
	cb2, rr := b.Reserve(context.Background())
	if !rr {
		t.Fatal("Reserve2 failed")
	}
	cb2()
}

func TestBreakerOverloadMixed(t *testing.T) {
	// This tests when reservation and maybe are intermised.
	params := BreakerParams{QueueDepth: 1, MaxConcurrency: 1, InitialCapacity: 1}
	b := NewBreaker(params) // Breaker capacity = 2
	reqs := newRequestor(b)

	// Bring breaker to capacity.
	reqs.request()
	// This happens in go-routine, so spin.
	for _, in := unpack(b.sem.state.Load()); in != 1; _, in = unpack(b.sem.state.Load()) {
		time.Sleep(time.Millisecond * 2)
	}
	_, rr := b.Reserve(context.Background())
	if rr {
		t.Fatal("Reserve was an unexpected success.")
	}
	// Open a slot.
	reqs.processSuccessfully(t)
	// Now reservation should work.
	cb, rr := b.Reserve(context.Background())
	if !rr {
		t.Fatal("Reserve unexpectedly failed")
	}
	// Process the reservation.
	cb()
}

func TestBreakerOverload(t *testing.T) {
	params := BreakerParams{QueueDepth: 1, MaxConcurrency: 1, InitialCapacity: 1}
	b := NewBreaker(params) // Breaker capacity = 2
	reqs := newRequestor(b)

	// Bring breaker to capacity.
	reqs.request()
	reqs.request()

	// Overshoot by one.
	reqs.request()
	reqs.expectFailure(t)

	// The remainer should succeed.
	reqs.processSuccessfully(t)
	reqs.processSuccessfully(t)
}

func TestBreakerQueueing(t *testing.T) {
	params := BreakerParams{QueueDepth: 2, MaxConcurrency: 1, InitialCapacity: 0}
	b := NewBreaker(params) // Breaker capacity = 2
	reqs := newRequestor(b)

	// Bring breaker to capacity. Doesn't error because queue subsumes these requests.
	reqs.request()
	reqs.request()

	// Update concurrency to allow the requests to be processed.
	b.UpdateConcurrency(1)

	// They should pass just fine.
	reqs.processSuccessfully(t)
	reqs.processSuccessfully(t)
}

func TestBreakerInflight(t *testing.T) {
	params := BreakerParams{QueueDepth: 2, MaxConcurrency: 1, InitialCapacity: 1}
	b := NewBreaker(params) // Breaker capacity = 2
	reqs := newRequestor(b)

	// Bring breaker to capacity. Doesn't error because queue subsumes these requests.
	reqs.request()
	reqs.request()
	reqs.request()

	require.Eventually(t, func() bool {
		return b.InFlight() == int64(3)
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, reqs.InProgress.Load(), int64(1))
}

func TestBreakerNoOverload(t *testing.T) {
	params := BreakerParams{QueueDepth: 1, MaxConcurrency: 1, InitialCapacity: 1}
	b := NewBreaker(params) // Breaker capacity = 2
	reqs := newRequestor(b)

	// Bring request to capacity.
	reqs.request()
	reqs.request()

	// Process one, send a new one in, at capacity again.
	reqs.processSuccessfully(t)
	reqs.request()

	// Process one, send a new one in, at capacity again.
	reqs.processSuccessfully(t)
	reqs.request()

	// Process the remainder successfully.
	reqs.processSuccessfully(t)
	reqs.processSuccessfully(t)
}

func TestBreakerCancel(t *testing.T) {
	params := BreakerParams{QueueDepth: 1, MaxConcurrency: 1, InitialCapacity: 0}
	b := NewBreaker(params)
	reqs := newRequestor(b)

	// Cancel a request which cannot get capacity.
	ctx1, cancel1 := context.WithCancel(context.Background())
	reqs.requestWithContext(ctx1)
	cancel1()
	reqs.expectFailure(t)

	// This request cannot get capacity either. This reproduced a bug we had when
	// freeing slots on the pendingRequests channel.
	ctx2, cancel2 := context.WithCancel(context.Background())
	reqs.requestWithContext(ctx2)
	cancel2()
	reqs.expectFailure(t)

	// Let through a request with capacity then timeout following request
	b.UpdateConcurrency(1)
	reqs.request()

	// Exceed capacity and assert one failure. This makes sure the Breaker is consistently
	// at capacity.
	reqs.request()
	reqs.request()
	reqs.expectFailure(t)

	// This request cannot get capacity.
	ctx3, cancel3 := context.WithCancel(context.Background())
	reqs.requestWithContext(ctx3)
	cancel3()
	reqs.expectFailure(t)

	// The requests that were put in earlier should succeed.
	reqs.processSuccessfully(t)
	reqs.processSuccessfully(t)
}

func TestBreakerUpdateConcurrency(t *testing.T) {
	params := BreakerParams{QueueDepth: 1, MaxConcurrency: 1, InitialCapacity: 0}
	b := NewBreaker(params)
	b.UpdateConcurrency(1)
	if got, want := b.Capacity(), 1; got != want {
		t.Errorf("Capacity() = %d, want: %d", got, want)
	}

	b.UpdateConcurrency(0)
	if got, want := b.Capacity(), 0; got != want {
		t.Errorf("Capacity() = %d, want: %d", got, want)
	}
}

// Test empty semaphore, token cannot be acquired
func TestSemaphoreAcquireHasNoCapacity(t *testing.T) {
	gotChan := make(chan struct{}, 1)

	sem := newSemaphore(1, 0)
	tryAcquire(sem, gotChan)

	select {
	case <-gotChan:
		t.Error("Token was acquired but shouldn't have been")
	case <-time.After(semNoChangeTimeout):
		// Test succeeds, semaphore didn't change in configured time.
	}
}

func TestSemaphoreAcquireNonBlockingHasNoCapacity(t *testing.T) {
	sem := newSemaphore(1, 0)
	if sem.tryAcquire() {
		t.Error("Should have failed immediately")
	}
}

// Test empty semaphore, add capacity, token can be acquired
func TestSemaphoreAcquireHasCapacity(t *testing.T) {
	gotChan := make(chan struct{}, 1)
	want := 1

	sem := newSemaphore(1, 0)
	tryAcquire(sem, gotChan)
	sem.updateCapacity(1) // Allows 1 acquire

	for i := 0; i < want; i++ {
		select {
		case <-gotChan:
			// Successfully acquired a token.
		case <-time.After(semAcquireTimeout):
			t.Error("Was not able to acquire token before timeout")
		}
	}

	select {
	case <-gotChan:
		t.Errorf("Got more acquires than wanted, want = %d, got at least %d", want, want+1)
	case <-time.After(semNoChangeTimeout):
		// No change happened, success.
	}
}

func TestSemaphoreRelease(t *testing.T) {
	sem := newSemaphore(1, 1)
	sem.acquire(context.Background())
	func() {
		defer func() {
			if e := recover(); e != nil {
				t.Error("Expected no panic, got message:", e)
			}
			sem.release()
		}()
	}()
	func() {
		defer func() {
			if e := recover(); e == nil {
				t.Error("Expected panic, but got none")
			}
		}()
		sem.release()
	}()
}

func TestSemaphoreUpdateCapacity(t *testing.T) {
	const initialCapacity = 1
	sem := newSemaphore(3, initialCapacity)
	if got, want := sem.Capacity(), 1; got != want {
		t.Errorf("Capacity = %d, want: %d", got, want)
	}
	sem.acquire(context.Background())
	sem.updateCapacity(initialCapacity + 2)
	if got, want := sem.Capacity(), 3; got != want {
		t.Errorf("Capacity = %d, want: %d", got, want)
	}
}

func TestPackUnpack(t *testing.T) {
	wantL := uint64(256)
	wantR := uint64(513)

	gotL, gotR := unpack(pack(wantL, wantR))

	if gotL != wantL || gotR != wantR {
		t.Fatalf("Got %d, %d want %d, %d", gotL, gotR, wantL, wantR)
	}
}

func tryAcquire(sem *semaphore, gotChan chan struct{}) {
	go func() {
		// blocking until someone puts the token into the semaphore
		sem.acquire(context.Background())
		gotChan <- struct{}{}
	}()
}

// requestor is a set of test helpers around breaker testing.
type requestor struct {
	breaker    *Breaker
	acceptedCh chan bool
	barrierCh  chan struct{}
	InProgress atomic.Int64
}

func newRequestor(breaker *Breaker) *requestor {
	return &requestor{
		breaker:    breaker,
		acceptedCh: make(chan bool),
		barrierCh:  make(chan struct{}),
	}
}

// request is the same as requestWithContext but with a default context.
func (r *requestor) request() {
	r.requestWithContext(context.Background())
}

// requestWithContext simulates a request in a separate goroutine. The
// request will either fail immediately (as observable via expectFailure)
// or block until processSuccessfully is called.
func (r *requestor) requestWithContext(ctx context.Context) {
	go func() {
		err := r.breaker.Maybe(ctx, func() {
			r.InProgress.Inc()
			<-r.barrierCh
		})
		r.acceptedCh <- err == nil
	}()
}

// expectFailure waits for a request to finish and asserts it to be failed.
func (r *requestor) expectFailure(t *testing.T) {
	t.Helper()
	if <-r.acceptedCh {
		t.Error("expected request to fail but it succeeded")
	}
}

// processSuccessfully allows a request to pass the barrier, waits for it to
// be finished and asserts it to succeed.
func (r *requestor) processSuccessfully(t *testing.T) {
	t.Helper()
	r.barrierCh <- struct{}{}
	if !<-r.acceptedCh {
		t.Error("expected request to succeed but it failed")
	}
}

func BenchmarkBreakerMaybe(b *testing.B) {
	op := func() {}

	for _, c := range []int{1, 10, 100, 1000} {
		breaker := NewBreaker(BreakerParams{QueueDepth: 10000000, MaxConcurrency: c, InitialCapacity: c})

		b.Run(fmt.Sprintf("%d-sequential", c), func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				breaker.Maybe(context.Background(), op)
			}
		})

		b.Run(fmt.Sprintf("%d-parallel", c), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					breaker.Maybe(context.Background(), op)
				}
			})
		})
	}
}

func BenchmarkBreakerReserve(b *testing.B) {
	op := func() {}
	breaker := NewBreaker(BreakerParams{QueueDepth: 1, MaxConcurrency: 10000000, InitialCapacity: 10000000})

	b.Run("sequential", func(b *testing.B) {
		for j := 0; j < b.N; j++ {
			free, got := breaker.Reserve(context.Background())
			op()
			if got {
				free()
			}
		}
	})

	b.Run("parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				free, got := breaker.Reserve(context.Background())
				op()
				if got {
					free()
				}
			}
		})
	})
}
