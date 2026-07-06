// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import "strings"

// The query/escape helpers mirror the encoding libcurl performs for Typhoeus
// (through Ethon::Easy#escape, i.e. curl_easy_escape): the RFC-3986 unreserved
// set [A-Za-z0-9-._~] is left literal and every other byte — including a space —
// becomes %XX with upper-case hex. This is the byte-for-byte behaviour the oracle
// tests diff against the gem's Ethon.

// BuildQuery renders params as an application/x-www-form-urlencoded query string:
// the params are emitted in insertion order (Typhoeus preserves the params' order
// rather than sorting) and each key and value is run through [Escape].
func BuildQuery(params *Params) string {
	var b strings.Builder
	for i, p := range params.pairs {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(Escape(p.Key))
		b.WriteByte('=')
		b.WriteString(Escape(p.Val))
	}
	return b.String()
}

// ParseQuery decodes an application/x-www-form-urlencoded query string into an
// ordered [Params]: each key and value is [Unescape]d, a bare key (no '=') maps to
// the empty string, an empty segment is skipped, and a later duplicate key
// overwrites an earlier one (keeping its position). A leading '?' is ignored.
func ParseQuery(query string) *Params {
	out := NewParams()
	query = strings.TrimPrefix(query, "?")
	if query == "" {
		return out
	}
	for _, seg := range strings.Split(query, "&") {
		if seg == "" {
			continue
		}
		k, v, _ := strings.Cut(seg, "=")
		out.Set(Unescape(k), Unescape(v))
	}
	return out
}

// Escape percent-encodes s the way libcurl's curl_easy_escape (Ethon::Easy#escape)
// does: the unreserved set [A-Za-z0-9-._~] is left literal and every other byte,
// a space included, becomes %XX with upper-case hex.
func Escape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escapeUnreserved(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hexDigit(c >> 4))
		b.WriteByte(hexDigit(c & 0xf))
	}
	return b.String()
}

// Unescape reverses [Escape]: %XX becomes its byte. An invalid or truncated %XX
// is left literal, matching libcurl's tolerant decoder (curl_easy_unescape). A
// '+' is left as-is (unlike form decoding), since curl's escaping emits %20 for a
// space.
func Unescape(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi, ok1 := fromHex(s[i+1])
			lo, ok2 := fromHex(s[i+2])
			if ok1 && ok2 {
				b.WriteByte(hi<<4 | lo)
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// escapeUnreserved reports whether c is left literal by [Escape]: the RFC-3986
// unreserved set [A-Za-z0-9-._~].
func escapeUnreserved(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '_', c == '.', c == '~':
		return true
	}
	return false
}

// hexDigit maps a nibble (0..15) to its upper-case hexadecimal ASCII digit.
func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + (n - 10)
}

// fromHex parses a single hexadecimal ASCII digit.
func fromHex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
