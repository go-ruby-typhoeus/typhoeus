// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import "sync"

// DefaultMaxConcurrency is the number of requests a [Hydra] runs at once when no
// MaxConcurrency is given, matching Typhoeus::Hydra's default of 200.
const DefaultMaxConcurrency = 200

// HydraOptions configures a [Hydra], mirroring Typhoeus::Hydra.new(max_concurrency:).
type HydraOptions struct {
	// MaxConcurrency bounds how many queued requests run at once; <= 0 selects
	// [DefaultMaxConcurrency].
	MaxConcurrency int
}

// Hydra is the parallel runner — Typhoeus's signature feature. Requests are
// queued with [Hydra.Queue] and executed by [Hydra.Run], which runs them
// concurrently (bounded by MaxConcurrency) and then fires each request's
// on-complete callbacks in queue order, mirroring Typhoeus::Hydra.
//
// A Hydra is not safe for concurrent use by multiple goroutines; queue from one
// goroutine and call Run. Run itself is where the parallelism happens.
type Hydra struct {
	maxConcurrency int
	queued         []*Request
	aborted        bool
}

// NewHydra builds a [Hydra], mirroring Typhoeus::Hydra.new. With no options (or
// MaxConcurrency <= 0) it uses [DefaultMaxConcurrency].
func NewHydra(opts ...HydraOptions) *Hydra {
	max := DefaultMaxConcurrency
	if len(opts) > 0 && opts[0].MaxConcurrency > 0 {
		max = opts[0].MaxConcurrency
	}
	return &Hydra{maxConcurrency: max}
}

// Queue adds a request to the hydra, mirroring Typhoeus::Hydra#queue. A request
// may also be queued from within an on-complete callback during [Hydra.Run]; it
// is run in a following round.
func (h *Hydra) Queue(r *Request) { h.queued = append(h.queued, r) }

// QueuedCount reports how many requests are currently queued
// (Typhoeus::Hydra#queued_requests size).
func (h *Hydra) QueuedCount() int { return len(h.queued) }

// Abort stops the hydra from starting further rounds, mirroring
// Typhoeus::Hydra#abort. Requests already in flight in the current round finish.
func (h *Hydra) Abort() { h.aborted = true }

// Run executes the queued requests concurrently and then fires their on-complete
// callbacks in queue order, mirroring Typhoeus::Hydra#run. It blocks until every
// request in flight has completed — no goroutine outlives the call — and returns
// once the queue is drained (or [Hydra.Abort] was called).
//
// Concurrency is bounded to MaxConcurrency by a semaphore; each request writes
// only its own slot of the results slice, and Run waits for the whole round
// before reading any result, so the transports run in parallel without a data
// race and without a leaked goroutine. Callbacks then run on the calling
// goroutine, in queue order, so results are deterministic; a callback may queue
// more requests, which are executed in the next round.
func (h *Hydra) Run() {
	for len(h.queued) > 0 && !h.aborted {
		batch := h.queued
		h.queued = nil

		responses := make([]*Response, len(batch))
		sem := make(chan struct{}, h.maxConcurrency)
		var wg sync.WaitGroup
		for i, req := range batch {
			wg.Add(1)
			sem <- struct{}{} // acquire: at most maxConcurrency requests in flight
			go func(i int, req *Request) {
				defer wg.Done()
				defer func() { <-sem }() // release
				responses[i] = req.execute()
			}(i, req)
		}
		wg.Wait()

		for i, req := range batch {
			req.finish(responses[i])
		}
	}
}
