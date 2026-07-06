// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import (
	"io"
	"net/http"
	"testing"
	"time"
)

// stubOK is a TransportFunc that returns a fixed 200 response and records the
// request it saw.
func stubOK(seen **Request) TransportFunc {
	return func(req *Request) *Response {
		if seen != nil {
			*seen = req
		}
		return &Response{Code: 200, Body: "ok", ReturnCode: ReturnOK}
	}
}

func TestNewRequest(t *testing.T) {
	r := NewRequest("http://x", "get")
	if r.Method != "GET" || r.URL != "http://x" {
		t.Fatalf("NewRequest = %+v", r)
	}
	if r.Options.Timeout != 0 {
		t.Fatal("default options should be zero")
	}
	r2 := NewRequest("http://x", "post", Options{Timeout: time.Second})
	if r2.Options.Timeout != time.Second {
		t.Fatal("options not applied")
	}
}

func TestRequestRunFiresCallbacks(t *testing.T) {
	var seen *Request
	req := NewRequest("http://x", "GET")
	req.Transport = stubOK(&seen)

	var got []string
	req.OnComplete(func(r *Response) { got = append(got, "a:"+r.Body) })
	req.OnComplete(func(r *Response) { got = append(got, "b:"+r.Body) })

	resp := req.Run()
	if resp.Code != 200 || resp.request != req {
		t.Fatalf("resp = %+v", resp)
	}
	if req.Response() != resp {
		t.Fatal("Response() not stored")
	}
	if seen != req {
		t.Fatal("transport saw wrong request")
	}
	if len(got) != 2 || got[0] != "a:ok" || got[1] != "b:ok" {
		t.Fatalf("callbacks fired wrong/out of order: %v", got)
	}
}

func TestRequestResponseNilBeforeRun(t *testing.T) {
	if NewRequest("http://x", "GET").Response() != nil {
		t.Fatal("Response() should be nil before Run")
	}
}

func TestBuiltURL(t *testing.T) {
	cases := []struct {
		url    string
		params *Params
		want   string
	}{
		{"http://x/p", nil, "http://x/p"},
		{"http://x/p", NewParams(), "http://x/p"},
		{"http://x/p", ParamsOf([2]string{"a", "b c"}), "http://x/p?a=b%20c"},
		{"http://x/p?z=1", ParamsOf([2]string{"a", "2"}), "http://x/p?z=1&a=2"},
	}
	for _, c := range cases {
		r := NewRequest(c.url, "GET", Options{Params: c.params})
		if got := r.builtURL(); got != c.want {
			t.Errorf("builtURL(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func readAll(t *testing.T, rd io.Reader) string {
	t.Helper()
	if rd == nil {
		return "<nil>"
	}
	b, err := io.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBodyReader(t *testing.T) {
	mk := func(body any) io.Reader { return NewRequest("http://x", "POST", Options{Body: body}).bodyReader() }
	if readAll(t, mk(nil)) != "<nil>" {
		t.Error("nil body should give nil reader")
	}
	if readAll(t, mk("")) != "<nil>" {
		t.Error("empty string body should give nil reader")
	}
	if readAll(t, mk("hello")) != "hello" {
		t.Error("string body")
	}
	if readAll(t, mk(ParamsOf([2]string{"a", "b c"}))) != "a=b%20c" {
		t.Error("params body should be form-encoded")
	}
	if readAll(t, mk(42)) != "42" {
		t.Error("other body should be fmt.Sprint'd")
	}
}

func TestApplyHeaders(t *testing.T) {
	// Form body sets Content-Type; explicit header overrides it; userpwd -> Basic.
	req := NewRequest("http://x", "POST", Options{
		Body:    ParamsOf([2]string{"a", "1"}),
		Headers: HeadersOf([2]string{"X-Custom", "v"}),
		UserPwd: "aladdin:opensesame",
	})
	hreq, _ := http.NewRequest("POST", "http://x", nil)
	req.applyHeaders(hreq)
	if hreq.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Error("form content-type not set")
	}
	if hreq.Header.Get("X-Custom") != "v" {
		t.Error("custom header not set")
	}
	user, pass, ok := hreq.BasicAuth()
	if !ok || user != "aladdin" || pass != "opensesame" {
		t.Errorf("basic auth = %q,%q,%v", user, pass, ok)
	}

	// No body, no headers, no userpwd -> nothing added.
	bare := NewRequest("http://x", "GET")
	hreq2, _ := http.NewRequest("GET", "http://x", nil)
	bare.applyHeaders(hreq2)
	if len(hreq2.Header) != 0 {
		t.Errorf("bare request should add no headers, got %v", hreq2.Header)
	}
}
