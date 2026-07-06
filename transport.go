// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import (
	"io"
	"net/http"
	"strings"
	"time"
)

// Transport is the host seam that performs a request's HTTP round-trip, the role
// libcurl plays for Typhoeus via Ethon. Given a [Request], a Transport performs
// the transfer and returns the [Response] — including the failure case, which is
// reported as a Response with a non-OK [ReturnCode] rather than an error, exactly
// as libcurl reports a CURLcode.
//
// The default production Transport is [NetHTTP], backed by net/http. The model
// opens no socket itself: every request runs through whatever Transport is set,
// so tests inject a stub (see [TransportFunc]) and the deterministic suite — the
// Hydra parallelism included — never touches the network.
type Transport interface {
	Run(req *Request) *Response
}

// TransportFunc adapts a function to the [Transport] interface, the convenient way
// to inject a stub transport in tests or a custom transport in a host.
type TransportFunc func(req *Request) *Response

// Run invokes f(req).
func (f TransportFunc) Run(req *Request) *Response { return f(req) }

// NetHTTPTransport is the default [Transport]: it turns a [Request] into a
// net/http request, executes it under an http.Client configured from the
// request's [Options] (timeout and redirect policy), and maps the response — or a
// transport failure — onto a [Response], classifying a failure into a libcurl
// [ReturnCode].
type NetHTTPTransport struct {
	// do performs the round-trip; the default calls client.Do. It is a field so
	// the response/error mapping can be driven deterministically in tests.
	do func(client *http.Client, req *http.Request) (*http.Response, error)
	// now supplies the clock for TotalTime; the default is time.Now.
	now func() time.Time
}

// NetHTTP returns the default net/http-backed [Transport].
func NetHTTP() *NetHTTPTransport {
	return &NetHTTPTransport{
		do:  func(c *http.Client, r *http.Request) (*http.Response, error) { return c.Do(r) },
		now: time.Now,
	}
}

// Run performs the HTTP round-trip for req with net/http.
func (t *NetHTTPTransport) Run(req *Request) *Response {
	start := t.now()
	hreq, err := http.NewRequest(req.Method, req.builtURL(), req.bodyReader())
	if err != nil {
		// An invalid method or URL: no transfer is possible at all.
		return &Response{ReturnCode: ReturnInternalError, TotalTime: t.elapsed(start)}
	}
	req.applyHeaders(hreq)

	hresp, err := t.do(buildClient(req.Options), hreq)
	if err != nil {
		return &Response{ReturnCode: classifyError(err), TotalTime: t.elapsed(start)}
	}
	defer hresp.Body.Close()

	raw, err := io.ReadAll(hresp.Body)
	if err != nil {
		return &Response{Code: hresp.StatusCode, ReturnCode: ReturnRecvError, TotalTime: t.elapsed(start)}
	}

	return &Response{
		Code:       hresp.StatusCode,
		Body:       string(raw),
		Headers:    headersFromHTTP(hresp.Header),
		ReturnCode: ReturnOK,
		TotalTime:  t.elapsed(start),
	}
}

// elapsed returns the seconds between start and now, per the transport's clock.
func (t *NetHTTPTransport) elapsed(start time.Time) float64 {
	return t.now().Sub(start).Seconds()
}

// buildClient constructs the http.Client for a request from its [Options],
// honouring the timeout and the followlocation/maxredirs redirect policy the way
// Typhoeus configures a curl handle.
func buildClient(opts Options) *http.Client {
	c := &http.Client{Timeout: opts.Timeout}
	switch {
	case !opts.FollowLocation:
		// Do not follow redirects: return the 3xx response as-is.
		c.CheckRedirect = noRedirect
	case opts.MaxRedirects > 0:
		c.CheckRedirect = maxRedirects(opts.MaxRedirects)
	}
	// FollowLocation with MaxRedirects == 0 leaves net/http's default policy.
	return c
}

// noRedirect is the CheckRedirect that stops at the first redirect, yielding the
// 3xx response.
func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// maxRedirects returns a CheckRedirect that follows at most max redirects.
func maxRedirects(max int) func(*http.Request, []*http.Request) error {
	return func(_ *http.Request, via []*http.Request) error {
		if len(via) >= max {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

// headersFromHTTP converts an http.Header into a [Headers], joining multi-valued
// headers with ", ".
func headersFromHTTP(h http.Header) *Headers {
	out := NewHeaders()
	for k, vs := range h {
		out.Set(k, strings.Join(vs, ", "))
	}
	return out
}
