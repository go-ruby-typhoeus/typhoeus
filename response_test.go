// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import "testing"

func TestResponseSuccess(t *testing.T) {
	cases := []struct {
		code int
		rc   ReturnCode
		want bool
	}{
		{200, ReturnOK, true},
		{201, ReturnOK, true},
		{299, ReturnOK, true},
		{300, ReturnOK, false},
		{404, ReturnOK, false},
		{199, ReturnOK, false},
		{200, ReturnCouldntConnect, false},
		{0, ReturnOperationTimedout, false},
	}
	for _, c := range cases {
		r := &Response{Code: c.code, ReturnCode: c.rc}
		if r.Success() != c.want {
			t.Errorf("Success(code=%d,rc=%v) = %v, want %v", c.code, c.rc, r.Success(), c.want)
		}
	}
}

func TestResponseTimedOutAndMessage(t *testing.T) {
	r := &Response{ReturnCode: ReturnOperationTimedout}
	if !r.TimedOut() {
		t.Fatal("TimedOut should be true")
	}
	if r.ReturnMessage() != "operation_timedout" {
		t.Fatalf("ReturnMessage = %q", r.ReturnMessage())
	}
	ok := &Response{ReturnCode: ReturnOK}
	if ok.TimedOut() {
		t.Fatal("TimedOut should be false for OK")
	}
}

func TestResponseRequest(t *testing.T) {
	req := NewRequest("http://x", "GET")
	r := &Response{request: req}
	if r.Request() != req {
		t.Fatal("Request() mismatch")
	}
}
