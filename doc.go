// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package typhoeus is a pure-Go (CGO-free) reimplementation of the client model
// of Ruby's `typhoeus` gem — the parallel HTTP client that, in Ruby, drives
// libcurl (via Ethon) to run many requests concurrently. Here the same object
// model (Request, Response, Hydra, the one-shot verb helpers, on-complete
// callbacks and the libcurl-style return codes) is backed by Go's net/http and
// goroutines instead of libcurl, with no cgo and no libcurl dependency.
//
// # What it is — and isn't
//
// Everything Typhoeus models around the wire is deterministic and needs no
// interpreter and no libcurl, so it lives here as pure Go: the [Request] with its
// [Options] (params, body, headers, userpwd, timeout, followlocation), the
// on-complete callbacks, the [Response] (code, body, headers, total_time,
// success?/timed_out?, return_code), and — the gem's signature feature — the
// parallel [Hydra] runner that executes a queue of requests concurrently with a
// bounded max-concurrency and fires their callbacks.
//
// The HTTP round-trip itself is a host seam: a [Transport] performs the transfer.
// The default production Transport is [NetHTTP], backed by net/http; tests inject
// a [TransportFunc] stub, so the deterministic suite drives the whole model —
// including the Hydra's parallelism — without opening a socket, and the concrete
// net/http mapping (libcurl return-code classification, redirect and timeout
// policy) is exercised through the same seam. This mirrors the gem, whose Ethon
// adapter is the only piece that touches the network.
//
// # One-shot
//
//	resp := typhoeus.Get("https://api.example.com/widgets",
//		typhoeus.Options{Params: typhoeus.ParamsOf([2]string{"q", "go ruby"})})
//	_ = resp.Code       // 200
//	_ = resp.Body       // response body string
//	_ = resp.Success()  // true for a 2xx with return_code == :ok
//
// # Parallel (Hydra)
//
//	hydra := typhoeus.NewHydra(typhoeus.HydraOptions{MaxConcurrency: 5})
//	for _, u := range urls {
//		req := typhoeus.NewRequest(u, "GET")
//		req.OnComplete(func(r *typhoeus.Response) { collect(r) })
//		hydra.Queue(req)
//	}
//	hydra.Run() // runs them concurrently, then fires callbacks in queue order
//
// # Value model
//
// Query params and url-encoded form bodies are carried as an ordered
// string→string [Params]; headers as a case-insensitive ordered [Headers]. A host
// (go-embedded-ruby / rbgo) maps its Ruby Typhoeus::Request / Response / Hydra
// objects to and from these shapes.
package typhoeus
