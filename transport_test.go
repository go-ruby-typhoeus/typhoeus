// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeTransport builds a NetHTTPTransport whose round-trip and clock are
// controlled, so the response/error mapping is exercised without a socket.
func fakeTransport(do func(*http.Client, *http.Request) (*http.Response, error)) *NetHTTPTransport {
	calls := 0
	return &NetHTTPTransport{
		do: do,
		now: func() time.Time {
			calls++
			return time.Unix(0, 0).Add(time.Duration(calls) * time.Second)
		},
	}
}

// errReadCloser fails on Read, to drive the recv-error branch.
type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errReadCloser) Close() error             { return nil }

func TestTransportFuncAdapter(t *testing.T) {
	want := &Response{Code: 204, ReturnCode: ReturnOK}
	var tf Transport = TransportFunc(func(*Request) *Response { return want })
	if tf.Run(NewRequest("http://x", "GET")) != want {
		t.Fatal("TransportFunc.Run mismatch")
	}
}

func TestNetHTTPRunSuccess(t *testing.T) {
	tr := fakeTransport(func(_ *http.Client, r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("hi")),
			Header:     http.Header{"X-A": {"1", "2"}},
		}, nil
	})
	resp := tr.Run(NewRequest("http://x", "GET"))
	if resp.Code != 200 || resp.Body != "hi" || resp.ReturnCode != ReturnOK {
		t.Fatalf("resp = %+v", resp)
	}
	if v, _ := resp.Headers.Get("X-A"); v != "1, 2" {
		t.Fatalf("multi-header join = %q", v)
	}
	if resp.TotalTime != 1 { // now() advances 1s per call, 2 calls -> delta 1s
		t.Fatalf("TotalTime = %v, want 1", resp.TotalTime)
	}
}

func TestNetHTTPRunBuildError(t *testing.T) {
	tr := fakeTransport(func(*http.Client, *http.Request) (*http.Response, error) {
		t.Fatal("do should not be called on a build error")
		return nil, nil
	})
	resp := tr.Run(NewRequest("http://x", "BAD METHOD")) // space -> invalid method token
	if resp.ReturnCode != ReturnInternalError || resp.Code != 0 {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestNetHTTPRunTransportError(t *testing.T) {
	tr := fakeTransport(func(*http.Client, *http.Request) (*http.Response, error) {
		return nil, plainErr{}
	})
	resp := tr.Run(NewRequest("http://x", "GET"))
	if resp.ReturnCode != ReturnCouldntConnect || resp.Code != 0 {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestNetHTTPRunRecvError(t *testing.T) {
	tr := fakeTransport(func(*http.Client, *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: errReadCloser{}, Header: http.Header{}}, nil
	})
	resp := tr.Run(NewRequest("http://x", "GET"))
	if resp.ReturnCode != ReturnRecvError || resp.Code != 200 || resp.Body != "" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestBuildClient(t *testing.T) {
	// FollowLocation false -> stop at first redirect.
	c := buildClient(Options{Timeout: 3 * time.Second})
	if c.Timeout != 3*time.Second {
		t.Fatal("timeout not applied")
	}
	if err := c.CheckRedirect(nil, nil); err != http.ErrUseLastResponse {
		t.Fatalf("noRedirect = %v", err)
	}

	// FollowLocation true + MaxRedirects: follow up to the cap.
	c2 := buildClient(Options{FollowLocation: true, MaxRedirects: 2})
	if err := c2.CheckRedirect(nil, make([]*http.Request, 1)); err != nil {
		t.Fatalf("under cap should follow: %v", err)
	}
	if err := c2.CheckRedirect(nil, make([]*http.Request, 2)); err != http.ErrUseLastResponse {
		t.Fatalf("at cap should stop: %v", err)
	}

	// FollowLocation true, no cap -> net/http default policy (nil CheckRedirect).
	c3 := buildClient(Options{FollowLocation: true})
	if c3.CheckRedirect != nil {
		t.Fatal("expected default redirect policy")
	}
}

// End-to-end through the real NetHTTP transport over a localhost httptest server
// (no external network): exercises the default do and clock.
func TestNetHTTPEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != "u" || p != "p" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-Echo", r.URL.RawQuery)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "pong")
	}))
	defer srv.Close()

	resp := NetHTTP().Run(NewRequest(srv.URL, "GET", Options{
		Params:  ParamsOf([2]string{"q", "1"}),
		UserPwd: "u:p",
	}))
	if !resp.Success() || resp.Code != 201 || resp.Body != "pong" {
		t.Fatalf("resp = %+v", resp)
	}
	if v, _ := resp.Headers.Get("X-Echo"); v != "q=1" {
		t.Fatalf("query not sent: %q", v)
	}
	if resp.TotalTime < 0 {
		t.Fatal("total time should be non-negative")
	}
}

func TestNetHTTPEndToEndNoFollowRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/from" {
			http.Redirect(w, r, "/to", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, "landed")
	}))
	defer srv.Close()

	// Default (FollowLocation false): the 302 is returned as-is.
	noFollow := NetHTTP().Run(NewRequest(srv.URL+"/from", "GET"))
	if noFollow.Code != http.StatusFound {
		t.Fatalf("no-follow code = %d, want 302", noFollow.Code)
	}
	// FollowLocation true: follows to /to.
	follow := NetHTTP().Run(NewRequest(srv.URL+"/from", "GET", Options{FollowLocation: true}))
	if follow.Code != 200 || follow.Body != "landed" {
		t.Fatalf("follow resp = %+v", follow)
	}
}
