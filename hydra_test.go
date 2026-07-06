// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestNewHydraMaxConcurrency(t *testing.T) {
	if NewHydra().maxConcurrency != DefaultMaxConcurrency {
		t.Fatal("default max concurrency wrong")
	}
	if NewHydra(HydraOptions{MaxConcurrency: 0}).maxConcurrency != DefaultMaxConcurrency {
		t.Fatal("<=0 should select default")
	}
	if NewHydra(HydraOptions{MaxConcurrency: 7}).maxConcurrency != 7 {
		t.Fatal("explicit max concurrency not applied")
	}
}

// stubURLTransport returns a 200 whose body echoes the request URL.
var stubURLTransport = TransportFunc(func(r *Request) *Response {
	return &Response{Code: 200, Body: r.URL, ReturnCode: ReturnOK}
})

func TestHydraRunDeterministicOrder(t *testing.T) {
	h := NewHydra(HydraOptions{MaxConcurrency: 3})
	var order []string
	reqs := make([]*Request, 5)
	for i := range reqs {
		req := NewRequest(fmt.Sprintf("http://x/%d", i), "GET")
		req.Transport = stubURLTransport
		req.OnComplete(func(r *Response) { order = append(order, r.Body) })
		reqs[i] = req
		h.Queue(req)
	}
	if h.QueuedCount() != 5 {
		t.Fatalf("QueuedCount = %d", h.QueuedCount())
	}
	h.Run()

	for i, req := range reqs {
		want := fmt.Sprintf("http://x/%d", i)
		if order[i] != want {
			t.Errorf("callback order[%d] = %q, want %q", i, order[i], want)
		}
		if req.Response() == nil || req.Response().request != req {
			t.Errorf("response not wired for %d", i)
		}
	}
	if h.QueuedCount() != 0 {
		t.Fatal("queue not drained")
	}
}

func TestHydraRunEmpty(t *testing.T) {
	NewHydra().Run() // no panic, no-op
}

func TestHydraQueueFromCallback(t *testing.T) {
	h := NewHydra(HydraOptions{MaxConcurrency: 2})
	var seen []string
	first := NewRequest("first", "GET")
	first.Transport = stubURLTransport
	first.OnComplete(func(r *Response) {
		seen = append(seen, r.Body)
		if r.Body == "first" {
			// queue a follow-up request from within the callback -> next round
			second := NewRequest("second", "GET")
			second.Transport = stubURLTransport
			second.OnComplete(func(r *Response) { seen = append(seen, r.Body) })
			h.Queue(second)
		}
	})
	h.Queue(first)
	h.Run()
	if len(seen) != 2 || seen[0] != "first" || seen[1] != "second" {
		t.Fatalf("callback-queued round not run: %v", seen)
	}
}

func TestHydraAbort(t *testing.T) {
	h := NewHydra(HydraOptions{MaxConcurrency: 2})
	ran := 0
	req := NewRequest("http://x", "GET")
	req.Transport = TransportFunc(func(r *Request) *Response {
		ran++
		return &Response{Code: 200, ReturnCode: ReturnOK}
	})
	req.OnComplete(func(*Response) {
		// abort during the first round; a callback-queued request must not run
		h.Abort()
		follow := NewRequest("nope", "GET")
		follow.Transport = req.Transport
		h.Queue(follow)
	})
	h.Queue(req)
	h.Run()
	if ran != 1 {
		t.Fatalf("aborted hydra ran %d requests, want 1", ran)
	}
	if h.QueuedCount() != 1 {
		t.Fatal("callback-queued request should remain un-run after abort")
	}
}

// TestHydraBoundedConcurrency proves the semaphore bounds in-flight requests to
// MaxConcurrency and that Run leaks no goroutine (it blocks until every worker
// has finished).
func TestHydraBoundedConcurrency(t *testing.T) {
	const max = 2
	const n = 5
	entered := make(chan struct{}, n)
	proceed := make(chan struct{})

	tf := TransportFunc(func(*Request) *Response {
		entered <- struct{}{}
		<-proceed // hold the worker so concurrency is observable
		return &Response{Code: 200, ReturnCode: ReturnOK}
	})

	h := NewHydra(HydraOptions{MaxConcurrency: max})
	for i := 0; i < n; i++ {
		req := NewRequest("http://x", "GET")
		req.Transport = tf
		h.Queue(req)
	}

	baseline := runtime.NumGoroutine()
	done := make(chan struct{})
	go func() { h.Run(); close(done) }()

	// Exactly `max` workers should be in flight; a further one must be blocked
	// on the semaphore, so no third has entered.
	for i := 0; i < max; i++ {
		<-entered
	}
	select {
	case <-entered:
		t.Fatal("more than MaxConcurrency requests ran at once")
	case <-time.After(50 * time.Millisecond):
	}

	close(proceed) // release every worker across all rounds
	<-done

	// After Run returns, in-flight worker goroutines are gone (wg.Wait).
	settle(t, baseline)
}

// settle waits for the goroutine count to return to at most the baseline,
// confirming Run left no goroutine running.
func settle(t *testing.T, baseline int) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if runtime.NumGoroutine() <= baseline {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("goroutines did not settle: now %d, baseline %d", runtime.NumGoroutine(), baseline)
}
