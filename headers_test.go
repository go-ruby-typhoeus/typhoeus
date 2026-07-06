// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import "testing"

func TestHeadersCaseInsensitive(t *testing.T) {
	h := HeadersOf([2]string{"Content-Type", "text/plain"})
	h.Set("content-type", "application/json") // same entry, keeps first casing
	if h.Len() != 1 {
		t.Fatalf("Len = %d, want 1", h.Len())
	}
	if v, ok := h.Get("CONTENT-TYPE"); !ok || v != "application/json" {
		t.Fatalf("Get = %q,%v", v, ok)
	}
	if h.Pairs()[0].Key != "Content-Type" {
		t.Fatalf("first-seen casing lost: %q", h.Pairs()[0].Key)
	}
	if !h.Has("content-type") || h.Has("missing") {
		t.Fatal("Has wrong")
	}
	if _, ok := h.Get("missing"); ok {
		t.Fatal("Get(missing) should be false")
	}
}

func TestHeadersSetOnEmpty(t *testing.T) {
	var h Headers // nil index
	h.Set("A", "1")
	if v, _ := h.Get("a"); v != "1" {
		t.Fatalf("Set on zero-value Headers failed: %q", v)
	}
}

func TestHeadersDelete(t *testing.T) {
	h := HeadersOf([2]string{"A", "1"}, [2]string{"B", "2"}, [2]string{"C", "3"})
	h.Delete("missing") // no-op
	h.Delete("b")
	if h.Len() != 2 || h.Has("B") {
		t.Fatalf("Delete failed: len=%d", h.Len())
	}
	// remaining keep order and are re-indexed
	if h.Pairs()[0].Key != "A" || h.Pairs()[1].Key != "C" {
		t.Fatalf("order after delete wrong: %+v", h.Pairs())
	}
	if v, _ := h.Get("C"); v != "3" {
		t.Fatalf("reindex after delete wrong: %q", v)
	}
}

func TestHeadersClone(t *testing.T) {
	h := HeadersOf([2]string{"A", "1"})
	c := h.Clone()
	c.Set("A", "2")
	if v, _ := h.Get("A"); v != "1" {
		t.Fatal("Clone shares state")
	}
}
