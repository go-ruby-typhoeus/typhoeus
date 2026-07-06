// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import "testing"

func TestEscape(t *testing.T) {
	cases := map[string]string{
		"hello world":   "hello%20world",
		"a b/c&d=é":     "a%20b%2Fc%26d%3D%C3%A9",
		"plain.text-_~": "plain.text-_~",
		"100%x":         "100%25x",
		"/?#[]@":        "%2F%3F%23%5B%5D%40",
	}
	for in, want := range cases {
		if got := Escape(in); got != want {
			t.Errorf("Escape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnescape(t *testing.T) {
	cases := map[string]string{
		"hello%20world": "hello world",
		"%C3%A9":        "é", // upper-case hex
		"%c3%a9":        "é", // lower-case hex
		"nochange":      "nochange",
		"bad%2":         "bad%2",  // truncated escape left literal
		"bad%zz":        "bad%zz", // invalid hex left literal
		"a+b":           "a+b",    // '+' is not a space in curl unescaping
	}
	for in, want := range cases {
		if got := Unescape(in); got != want {
			t.Errorf("Unescape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildQuery(t *testing.T) {
	p := ParamsOf([2]string{"b", "2"}, [2]string{"a", "hello world"}, [2]string{"c", "x&y"})
	want := "b=2&a=hello%20world&c=x%26y"
	if got := BuildQuery(p); got != want {
		t.Fatalf("BuildQuery = %q, want %q", got, want)
	}
	if got := p.Encode(); got != want {
		t.Fatalf("Encode = %q, want %q", got, want)
	}
	if got := BuildQuery(NewParams()); got != "" {
		t.Fatalf("BuildQuery(empty) = %q, want empty", got)
	}
}

func TestParseQuery(t *testing.T) {
	p := ParseQuery("?a=hello%20world&b=2&bare&&c=x%26y")
	want := map[string]string{"a": "hello world", "b": "2", "bare": "", "c": "x&y"}
	for k, w := range want {
		if v, ok := p.Get(k); !ok || v != w {
			t.Errorf("ParseQuery[%q] = %q,%v, want %q", k, v, ok, w)
		}
	}
	if ParseQuery("").Len() != 0 {
		t.Fatal("ParseQuery(\"\") should be empty")
	}
	if ParseQuery("?").Len() != 0 {
		t.Fatal("ParseQuery(\"?\") should be empty")
	}
}
