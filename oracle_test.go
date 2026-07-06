// Copyright (c) the go-ruby-typhoeus/typhoeus authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typhoeus

import (
	"os/exec"
	"strings"
	"testing"
)

// The oracle test diffs this package's curl-style escaping against the reference
// `typhoeus` gem, which escapes through Ethon::Easy#escape (libcurl's
// curl_easy_escape). It drives the gem to escape the same inputs and asserts
// byte-for-byte agreement. It skips itself where the gem (or the ruby/libcurl it
// needs) is absent — the qemu cross-arch and Windows lanes — so the
// deterministic, ruby-free suite alone holds the 100% coverage gate there.

// gemRuby reports a ruby whose typhoeus gem exposes the Ethon::Easy#escape helper
// we diff against, or skips.
func gemRuby(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping typhoeus-gem oracle")
	}
	probe := `require "typhoeus"
exit(Ethon::Easy.new.respond_to?(:escape) ? 0 : 1)`
	if err := exec.Command(bin, "-e", probe).Run(); err != nil {
		t.Skip("typhoeus gem absent or lacks Ethon::Easy#escape; skipping")
	}
	return bin
}

// rubyEval runs a ruby script (typhoeus required, stdout binary) and returns the
// newline-trimmed stdout, failing on error.
func rubyEval(t *testing.T, bin, script string) string {
	t.Helper()
	cmd := exec.Command(bin, "-rtyphoeus", "-e", "$stdout.binmode\ne = Ethon::Easy.new\n"+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return strings.TrimRight(string(out), "\n")
}

func TestOracleEscape(t *testing.T) {
	bin := gemRuby(t)
	// Avoid '~' so the diff holds across libcurl versions; every other byte class
	// (alnum, unreserved -._, space, reserved, multibyte) is covered.
	for _, s := range []string{"a b/c&d=é", "hello world", "plain.text-_", "100%x", "/?#[]@"} {
		want := rubyEval(t, bin, "print e.escape("+rubyString(s)+")")
		if got := Escape(s); got != want {
			t.Fatalf("Escape(%q) = %q, gem = %q", s, got, want)
		}
	}
}

// rubyString renders s as a double-quoted ruby string literal (escaping the few
// bytes that matter for the small oracle inputs used here).
func rubyString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return `"` + r.Replace(s) + `"`
}
