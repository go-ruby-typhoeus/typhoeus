// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import "testing"

// withStubTransport swaps DefaultTransport for a recorder for the duration of fn,
// so the package-level verb helpers are tested without touching the network.
func withStubTransport(t *testing.T, fn func(seen *[]string)) {
	t.Helper()
	var seen []string
	prev := DefaultTransport
	DefaultTransport = TransportFunc(func(r *Request) *Response {
		seen = append(seen, r.Method+" "+r.builtURL())
		return &Response{Code: 200, Body: r.Method, ReturnCode: ReturnOK}
	})
	defer func() { DefaultTransport = prev }()
	fn(&seen)
}

func TestVerbHelpers(t *testing.T) {
	withStubTransport(t, func(seen *[]string) {
		verbs := []struct {
			name string
			call func(string, ...Options) *Response
		}{
			{"GET", Get},
			{"POST", Post},
			{"PUT", Put},
			{"DELETE", Delete},
			{"HEAD", Head},
			{"PATCH", Patch},
		}
		for _, v := range verbs {
			resp := v.call("http://x/p")
			if resp.Code != 200 || resp.Body != v.name || !resp.Success() {
				t.Fatalf("%s resp = %+v", v.name, resp)
			}
		}
		want := []string{
			"GET http://x/p", "POST http://x/p", "PUT http://x/p",
			"DELETE http://x/p", "HEAD http://x/p", "PATCH http://x/p",
		}
		if len(*seen) != len(want) {
			t.Fatalf("seen = %v", *seen)
		}
		for i, w := range want {
			if (*seen)[i] != w {
				t.Errorf("seen[%d] = %q, want %q", i, (*seen)[i], w)
			}
		}
	})
}

func TestVerbHelpersPassOptions(t *testing.T) {
	withStubTransport(t, func(seen *[]string) {
		Get("http://x/p", Options{Params: ParamsOf([2]string{"a", "b c"})})
		if (*seen)[0] != "GET http://x/p?a=b%20c" {
			t.Fatalf("options not threaded: %q", (*seen)[0])
		}
	})
}
