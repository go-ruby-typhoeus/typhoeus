// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

// Response is the result of a transfer, mirroring Typhoeus::Response. A Response
// is always produced — even for a failed transfer, where Code is 0 and
// ReturnCode names the libcurl failure (see [ReturnCode]).
type Response struct {
	// Code is the HTTP status code (Response#code / #response_code); 0 when the
	// transfer never produced a status (a connection/timeout failure).
	Code int
	// Body is the response body (Response#body).
	Body string
	// Headers are the response headers (Response#headers).
	Headers *Headers
	// TotalTime is the wall-clock duration of the transfer in seconds
	// (Response#total_time).
	TotalTime float64
	// ReturnCode is the libcurl return code (Response#return_code): [ReturnOK] on
	// success, otherwise the failure mode.
	ReturnCode ReturnCode

	// request is the [Request] that produced this response (Response#request).
	request *Request
}

// Success reports whether the transfer succeeded with a 2xx status, mirroring
// Typhoeus::Response#success? (return_code == :ok and a 200..299 code).
func (r *Response) Success() bool {
	return r.ReturnCode == ReturnOK && r.Code >= 200 && r.Code < 300
}

// TimedOut reports whether the transfer timed out, mirroring
// Typhoeus::Response#timed_out? (return_code == :operation_timedout).
func (r *Response) TimedOut() bool { return r.ReturnCode == ReturnOperationTimedout }

// ReturnMessage returns the human-readable return-code name
// (Response#return_message), e.g. "ok" or "couldnt_connect".
func (r *Response) ReturnMessage() string { return r.ReturnCode.String() }

// Request returns the [Request] that produced this response
// (Typhoeus::Response#request).
func (r *Response) Request() *Request { return r.request }
