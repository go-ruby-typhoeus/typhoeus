// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

// DefaultTransport is the [Transport] used by the one-shot verb helpers and by
// any [Request] whose Transport field is nil. It defaults to [NetHTTP]; a host or
// test may replace it to route every request through a custom transport.
var DefaultTransport Transport = NetHTTP()

// Get performs a one-shot GET, mirroring Typhoeus.get(url, options). It builds a
// [Request], runs it through [DefaultTransport] (firing any callbacks the request
// would carry), and returns the [Response].
func Get(url string, opts ...Options) *Response { return NewRequest(url, "GET", opts...).Run() }

// Post performs a one-shot POST, mirroring Typhoeus.post(url, options).
func Post(url string, opts ...Options) *Response { return NewRequest(url, "POST", opts...).Run() }

// Put performs a one-shot PUT, mirroring Typhoeus.put(url, options).
func Put(url string, opts ...Options) *Response { return NewRequest(url, "PUT", opts...).Run() }

// Delete performs a one-shot DELETE, mirroring Typhoeus.delete(url, options).
func Delete(url string, opts ...Options) *Response { return NewRequest(url, "DELETE", opts...).Run() }

// Head performs a one-shot HEAD, mirroring Typhoeus.head(url, options).
func Head(url string, opts ...Options) *Response { return NewRequest(url, "HEAD", opts...).Run() }

// Patch performs a one-shot PATCH, mirroring Typhoeus.patch(url, options).
func Patch(url string, opts ...Options) *Response { return NewRequest(url, "PATCH", opts...).Run() }
