// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import (
	"errors"
	"net"
	"testing"
)

func TestReturnCodeString(t *testing.T) {
	cases := map[ReturnCode]string{
		ReturnOK:                 "ok",
		ReturnCouldntResolveHost: "couldnt_resolve_host",
		ReturnCouldntConnect:     "couldnt_connect",
		ReturnOperationTimedout:  "operation_timedout",
		ReturnRecvError:          "recv_error",
		ReturnInternalError:      "internal_error",
		ReturnCode(999):          "unknown",
	}
	for code, want := range cases {
		if got := code.String(); got != want {
			t.Errorf("ReturnCode(%d).String() = %q, want %q", int(code), got, want)
		}
	}
}

// timeoutErr is a net.Error whose Timeout reports true.
type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

// plainErr is a net.Error whose Timeout reports false.
type plainErr struct{}

func (plainErr) Error() string   { return "connection refused" }
func (plainErr) Timeout() bool   { return false }
func (plainErr) Temporary() bool { return false }

func TestClassifyError(t *testing.T) {
	if got := classifyError(timeoutErr{}); got != ReturnOperationTimedout {
		t.Errorf("timeout -> %v", got)
	}
	if got := classifyError(&net.DNSError{Err: "no such host"}); got != ReturnCouldntResolveHost {
		t.Errorf("dns -> %v", got)
	}
	if got := classifyError(plainErr{}); got != ReturnCouldntConnect {
		t.Errorf("plain net.Error -> %v", got)
	}
	if got := classifyError(errors.New("boom")); got != ReturnCouldntConnect {
		t.Errorf("generic -> %v", got)
	}
}
