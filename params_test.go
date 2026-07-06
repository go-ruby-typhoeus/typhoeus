// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import "testing"

func TestParamsOrderAndOverwrite(t *testing.T) {
	p := ParamsOf([2]string{"b", "1"}, [2]string{"a", "2"})
	p.Set("b", "9") // overwrite keeps position, updates value
	if p.Len() != 2 {
		t.Fatalf("Len = %d", p.Len())
	}
	if p.Pairs()[0].Key != "b" || p.Pairs()[0].Val != "9" {
		t.Fatalf("overwrite wrong: %+v", p.Pairs())
	}
	if v, ok := p.Get("a"); !ok || v != "2" {
		t.Fatalf("Get(a) = %q,%v", v, ok)
	}
	if !p.Has("a") || p.Has("z") {
		t.Fatal("Has wrong")
	}
	if _, ok := p.Get("z"); ok {
		t.Fatal("Get(z) should be false")
	}
}

func TestParamsSetOnEmpty(t *testing.T) {
	var p Params
	p.Set("k", "v")
	if v, _ := p.Get("k"); v != "v" {
		t.Fatalf("Set on zero-value Params failed: %q", v)
	}
}

func TestParamsDelete(t *testing.T) {
	p := ParamsOf([2]string{"a", "1"}, [2]string{"b", "2"}, [2]string{"c", "3"})
	p.Delete("missing") // no-op
	p.Delete("a")
	if p.Len() != 2 || p.Has("a") {
		t.Fatalf("Delete failed: %+v", p.Pairs())
	}
	if v, _ := p.Get("c"); v != "3" {
		t.Fatalf("reindex wrong: %q", v)
	}
}

func TestParamsClone(t *testing.T) {
	p := ParamsOf([2]string{"a", "1"})
	c := p.Clone()
	c.Set("a", "2")
	if v, _ := p.Get("a"); v != "1" {
		t.Fatal("Clone shares state")
	}
}
