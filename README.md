<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-typhoeus/brand/main/social/go-ruby-typhoeus-typhoeus.png" alt="go-ruby-typhoeus/typhoeus" width="720"></p>

# typhoeus — go-ruby-typhoeus

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-typhoeus.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the client model of Ruby's
[`typhoeus`](https://github.com/typhoeus/typhoeus) gem** — the parallel HTTP
client. It reproduces the request/response objects, the one-shot verb helpers,
the on-complete callbacks, the libcurl-style return codes, and — the gem's
signature feature — the **`Hydra` parallel runner**, all backed by Go's
`net/http` and goroutines instead of libcurl. **No Ruby runtime, no libcurl, no
cgo.**

It is the Typhoeus client for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a **standalone,
reusable** module — a sibling of
[go-ruby-faraday](https://github.com/go-ruby-faraday/faraday),
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) and
[go-ruby-erb](https://github.com/go-ruby-erb/erb).

> **What it is — and isn't.** Everything Typhoeus models *around* the wire is
> deterministic and needs **no interpreter and no libcurl**, so it lives here as
> pure Go: the `Request` with its `Options` (params, body, headers, userpwd,
> timeout, followlocation), the on-complete callbacks, the `Response` (code, body,
> headers, total_time, `Success`/`TimedOut`, return_code), and the parallel
> `Hydra` runner. The **HTTP round-trip itself is a host seam**: a `Transport`
> performs the transfer. The default production `Transport` is `NetHTTP`, backed
> by `net/http`; **tests inject a `TransportFunc` stub and the model — the Hydra's
> parallelism included — never opens a socket.** This mirrors the gem, whose Ethon
> adapter is the only piece that touches the network.

## Features

Faithful port of the `typhoeus` gem's client model, its escaping validated
against the gem (Ethon/libcurl) on every platform where it is installed:

- **One-shot verbs** — `Get`/`Post`/`Put`/`Delete`/`Head`/`Patch(url, Options)`,
  each returning a `*Response` (mirroring `Typhoeus.get(url, options)`).
- **`Request`** — `NewRequest(url, method, Options)` with `method`, `url` and
  `Options{Params, Body, Headers, UserPwd, Timeout, FollowLocation, MaxRedirects}`,
  `OnComplete` callbacks, and `Run` (mirroring `Typhoeus::Request`).
- **`Hydra`** — the parallel runner: `NewHydra(HydraOptions{MaxConcurrency})`,
  `Queue(req)`, then `Run()` executes the queue **concurrently** (goroutines,
  bounded by `MaxConcurrency`) and fires each request's callbacks **in queue
  order**. A callback may itself `Queue` more requests, which run in a following
  round; `Abort()` stops further rounds. Deterministic and leak-free.
- **`Response`** — `Code`, `Body`, `Headers`, `TotalTime`, `ReturnCode`, plus
  `Success()` (2xx + `:ok`), `TimedOut()` and `ReturnMessage()`.
- **`ReturnCode`** — the libcurl result Typhoeus surfaces as `return_code`
  (`:ok`, `:couldnt_resolve_host`, `:couldnt_connect`, `:operation_timedout`,
  `:recv_error`, `:internal_error`), classified from the transport failure.
- **Transport seam** — `req.Transport` / `DefaultTransport`; `NetHTTP()` is the
  default net/http transport (honouring timeout, followlocation, maxredirs, basic
  `userpwd`), a `TransportFunc` a test stub. **The model never opens a socket.**
- **Ordered `Params` / case-insensitive `Headers`** and curl-style `Escape` /
  `Unescape` / `BuildQuery` / `ParseQuery` (space → `%20`, RFC-3986 unreserved).

CGO-free, dependency-free (stdlib only), **100% test coverage**, `gofmt` +
`go vet` clean, and green across the six 64-bit Go targets (amd64, arm64,
riscv64, loong64, ppc64le, **s390x** — big-endian).

## Install

```sh
go get github.com/go-ruby-typhoeus/typhoeus
```

## Usage

### One-shot

```go
package main

import (
	"fmt"

	"github.com/go-ruby-typhoeus/typhoeus"
)

func main() {
	resp := typhoeus.Get("https://api.example.com/widgets",
		typhoeus.Options{Params: typhoeus.ParamsOf([2]string{"q", "go ruby"})})

	fmt.Println(resp.Code)      // 200
	fmt.Println(resp.Success()) // true for a 2xx with return_code == :ok
	fmt.Println(resp.Body)      // response body string
}
```

### Parallel (Hydra)

```go
hydra := typhoeus.NewHydra(typhoeus.HydraOptions{MaxConcurrency: 5})
for _, u := range urls {
	req := typhoeus.NewRequest(u, "GET")
	req.OnComplete(func(r *typhoeus.Response) {
		fmt.Println(r.Request().URL, r.Code, r.TotalTime)
	})
	hydra.Queue(req)
}
hydra.Run() // runs them concurrently, then fires callbacks in queue order
```

### Injecting a transport (tests / hosts)

```go
req := typhoeus.NewRequest("https://api.example.com/ping", "GET")
req.Transport = typhoeus.TransportFunc(func(r *typhoeus.Request) *typhoeus.Response {
	return &typhoeus.Response{Code: 200, Body: "pong", ReturnCode: typhoeus.ReturnOK}
})
resp := req.Run() // resp.Body == "pong", no socket opened
```

## Value model

| gem                                          | this package                                       |
| -------------------------------------------- | -------------------------------------------------- |
| `Typhoeus.get(url, options)`                 | `Get(url, Options{...})` (`Post`/`Put`/…)          |
| `Typhoeus::Request.new(url, method:, ...)`   | `NewRequest(url, method, Options{...})`            |
| `request.on_complete { \|resp\| ... }`       | `req.OnComplete(func(*Response) { ... })`          |
| `request.run`                                | `req.Run()`                                        |
| `Typhoeus::Hydra.new(max_concurrency: n)`    | `NewHydra(HydraOptions{MaxConcurrency: n})`        |
| `hydra.queue(req)` / `hydra.run`             | `hydra.Queue(req)` / `hydra.Run()`                 |
| `resp.code / body / headers / total_time`    | `resp.Code / Body / Headers / TotalTime`           |
| `resp.success? / timed_out? / return_code`   | `resp.Success() / TimedOut() / ReturnCode`         |
| Ethon `easy.escape` (libcurl)                | `Escape` / `Unescape` / `BuildQuery`               |

## Tests & coverage

The suite pairs deterministic, ruby-free tests (which alone hold coverage at
**100%**, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential oracle** against the reference `typhoeus` gem: the curl-style
percent-escaping is diffed **byte-for-byte** against the gem's
`Ethon::Easy#escape` (libcurl). The oracle skips itself where the gem is absent.
The Hydra's parallelism is covered **deterministically** — a semaphore barrier
proves the `MaxConcurrency` bound and that `Run` leaks no goroutine — and the
whole suite is race-clean (`go test -race`). Only localhost `httptest` servers
back the `net/http` integration path; the transport is a stub everywhere else.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-typhoeus/typhoeus authors.
