// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import (
	"errors"
	"net"
)

// ReturnCode is the libcurl transfer result Typhoeus exposes as
// Response#return_code (a Ruby Symbol such as :ok or :operation_timedout). A
// successful transfer is [ReturnOK]; every failure mode maps to a specific code,
// so a failed request still yields a [Response] (with Code 0) rather than an
// error — mirroring libcurl, where failures are return codes, not exceptions.
type ReturnCode int

// The return codes modelled here, named after the libcurl CURLcode symbols
// Typhoeus surfaces.
const (
	// ReturnOK is a completed transfer (CURLE_OK, :ok).
	ReturnOK ReturnCode = iota
	// ReturnCouldntResolveHost is a DNS resolution failure
	// (CURLE_COULDNT_RESOLVE_HOST, :couldnt_resolve_host).
	ReturnCouldntResolveHost
	// ReturnCouldntConnect is a connection failure
	// (CURLE_COULDNT_CONNECT, :couldnt_connect).
	ReturnCouldntConnect
	// ReturnOperationTimedout is a timeout (CURLE_OPERATION_TIMEDOUT,
	// :operation_timedout).
	ReturnOperationTimedout
	// ReturnRecvError is a failure receiving the response body
	// (CURLE_RECV_ERROR, :recv_error).
	ReturnRecvError
	// ReturnInternalError is a request that could not be built at all
	// (an invalid method or URL); Typhoeus reports such a handle as
	// :internal_error.
	ReturnInternalError
)

// returnCodeNames maps each code to the Ruby Symbol name Typhoeus uses.
var returnCodeNames = map[ReturnCode]string{
	ReturnOK:                 "ok",
	ReturnCouldntResolveHost: "couldnt_resolve_host",
	ReturnCouldntConnect:     "couldnt_connect",
	ReturnOperationTimedout:  "operation_timedout",
	ReturnRecvError:          "recv_error",
	ReturnInternalError:      "internal_error",
}

// String returns the libcurl symbol name for the code (e.g. "operation_timedout"),
// matching the Symbol Typhoeus returns from Response#return_code. An unknown code
// renders as "unknown".
func (c ReturnCode) String() string {
	if name, ok := returnCodeNames[c]; ok {
		return name
	}
	return "unknown"
}

// classifyError maps a net/http transport error to the matching [ReturnCode], the
// way Typhoeus maps a libcurl failure to a CURLcode: a DNS error becomes
// couldnt_resolve_host, a timeout becomes operation_timedout, and anything else
// becomes couldnt_connect.
func classifyError(err error) ReturnCode {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return ReturnOperationTimedout
	}
	var de *net.DNSError
	if errors.As(err, &de) {
		return ReturnCouldntResolveHost
	}
	return ReturnCouldntConnect
}
