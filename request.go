// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Options carries the per-request settings, mirroring the options Hash passed to
// Typhoeus::Request.new(url, method:, params:, body:, headers:, userpwd:,
// timeout:, followlocation:). Every field is optional; the zero value requests a
// plain transfer with the transport's defaults.
type Options struct {
	// Params are query parameters appended to the URL (curl-escaped, ordered).
	Params *Params
	// Body is the request body: a string sent verbatim, a [*Params] form-encoded
	// as application/x-www-form-urlencoded, nil for no body, or any other value
	// rendered with fmt.Sprint.
	Body any
	// Headers are the outgoing request headers.
	Headers *Headers
	// UserPwd is the HTTP Basic credential as "login:password" (curl's :userpwd);
	// when set it becomes an Authorization: Basic header.
	UserPwd string
	// Timeout is the overall transfer timeout; 0 means no timeout.
	Timeout time.Duration
	// FollowLocation, when true, follows 3xx redirects (curl's :followlocation).
	// When false the redirect response is returned as-is.
	FollowLocation bool
	// MaxRedirects caps the number of redirects to follow when FollowLocation is
	// true (curl's :maxredirs); 0 leaves the transport default.
	MaxRedirects int
}

// Request is a single HTTP request, mirroring Typhoeus::Request: a method, a URL,
// the per-request [Options], and any number of on-complete callbacks. Run it
// directly with [Request.Run], or queue it on a [Hydra] to run in parallel with
// others.
type Request struct {
	// Method is the upper-case HTTP method ("GET", "POST", …).
	Method string
	// URL is the request URL (query params from Options are appended to it).
	URL string
	// Options are the per-request settings.
	Options Options
	// Transport, when non-nil, overrides [DefaultTransport] for this request — the
	// injectable seam tests use to avoid the network.
	Transport Transport

	response   *Response
	onComplete []func(*Response)
}

// NewRequest builds a [Request] for url with the given method and optional
// [Options], mirroring Typhoeus::Request.new. The method is upper-cased.
func NewRequest(url, method string, opts ...Options) *Request {
	r := &Request{Method: strings.ToUpper(method), URL: url}
	if len(opts) > 0 {
		r.Options = opts[0]
	}
	return r
}

// OnComplete registers a callback fired with the [Response] when the request
// finishes, mirroring Typhoeus::Request#on_complete. Callbacks run in
// registration order.
func (r *Request) OnComplete(cb func(*Response)) {
	r.onComplete = append(r.onComplete, cb)
}

// Response returns the [Response] produced by the last run, or nil if the request
// has not run yet (Typhoeus::Request#response).
func (r *Request) Response() *Response { return r.response }

// Run performs the request through its transport and fires the on-complete
// callbacks, mirroring Typhoeus::Request#run. It returns the [Response] (also
// available via [Request.Response]).
func (r *Request) Run() *Response {
	resp := r.execute()
	r.finish(resp)
	return resp
}

// transport returns the request's transport, falling back to [DefaultTransport].
func (r *Request) transport() Transport {
	if r.Transport != nil {
		return r.Transport
	}
	return DefaultTransport
}

// execute runs the transport and records the response, without firing callbacks;
// the [Hydra] fires callbacks itself, in queue order, after a batch completes.
func (r *Request) execute() *Response {
	resp := r.transport().Run(r)
	resp.request = r
	r.response = resp
	return resp
}

// finish fires the on-complete callbacks with resp.
func (r *Request) finish(resp *Response) {
	for _, cb := range r.onComplete {
		cb(resp)
	}
}

// builtURL returns the URL with the Options params appended as a query string.
func (r *Request) builtURL() string {
	if r.Options.Params == nil || r.Options.Params.Len() == 0 {
		return r.URL
	}
	sep := "?"
	if strings.Contains(r.URL, "?") {
		sep = "&"
	}
	return r.URL + sep + BuildQuery(r.Options.Params)
}

// bodyReader returns the io.Reader for the request body, or nil for no body.
func (r *Request) bodyReader() io.Reader {
	switch b := r.Options.Body.(type) {
	case nil:
		return nil
	case string:
		if b == "" {
			return nil
		}
		return strings.NewReader(b)
	case *Params:
		return strings.NewReader(BuildQuery(b))
	default:
		return strings.NewReader(fmt.Sprint(b))
	}
}

// applyHeaders sets the request headers, a form Content-Type for a [*Params]
// body, and Basic auth from UserPwd, onto the net/http request.
func (r *Request) applyHeaders(hreq *http.Request) {
	if _, ok := r.Options.Body.(*Params); ok {
		hreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if r.Options.Headers != nil {
		for _, p := range r.Options.Headers.Pairs() {
			hreq.Header.Set(p.Key, p.Val)
		}
	}
	if r.Options.UserPwd != "" {
		user, pass, _ := strings.Cut(r.Options.UserPwd, ":")
		hreq.SetBasicAuth(user, pass)
	}
}
