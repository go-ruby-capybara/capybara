<p align="center"><img src="https://go-ruby-capybara.github.io/logo.png" alt="go-ruby-capybara/capybara" width="720"></p>

# capybara — go-ruby-capybara

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-capybara.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the deterministic core of Ruby's
[`capybara`](https://github.com/teamcapybara/capybara) gem** together with its
default, browser-free `:rack_test` driver. It reproduces the acceptance-test DSL
— `visit`, `click_link`, `fill_in`, `find`, `has_content?`, `within` — over an
in-process HTML DOM, driving a [Rack](https://github.com/rack/rack) app directly,
**without any Ruby runtime, browser, or JavaScript**.

It is the Capybara for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a **standalone,
reusable** module — a sibling of the other `go-ruby-*` testing gems.

> **What it is — and isn't.** Everything the rack_test driver does is
> deterministic and needs **no interpreter**: parse the last HTML response,
> resolve a CSS/XPath/semantic selector to a node, mutate the in-memory DOM as
> the "user" interacts, and reconstruct the next request (a link's `href`, a
> form's serialized fields + method + action). The **application under test is a
> host seam** — an [`App`](env.go) func mapping a `*Request` to a `*Response`.
> Tests inject a plain in-memory Go func and **no socket is opened**. The rbgo
> binding wires the seam to a real Ruby Rack app: the `*Request` becomes a Rack
> `env` hash, `app.call(env)` runs, and the `[status, headers, body]` triple
> becomes a `*Response`.

## Features

Faithful port of Capybara's session DSL over the rack_test driver:

- **Navigation** — `Visit(path)`, `CurrentPath`, `CurrentURL`, automatic
  redirect following (limit 5, then `InfiniteRedirect`), `ClickLink`,
  `ClickButton`, `ClickOn`.
- **Forms** — `FillIn`, `Choose`, `Check`/`Uncheck`, `Select(value, from)`,
  `AttachFile`, `FindField`, with HTML-correct submission (GET → query string,
  POST → `application/x-www-form-urlencoded`; disabled controls, unchecked
  boxes, and select defaults handled as the spec requires).
- **Querying / matchers** — `Find`/`FindXPath`, `All`/`AllXPath`, `First`,
  `HasSelector?`/`HasCSS?`/`HasXPath?`/`HasContent?`/`HasText?`/`HasLink?`/
  `HasButton?`/`HasField?` (and negations), the `Assert*` variants, and
  `Within(scope, fn)` to restrict a block to a subtree.
- **Nodes** — `Text`, `Attr`/`AttrOr` (`#[]`), `Value`, `Click`, `Set`,
  `TagName`, `Checked`, `Selected`, `Visible`.
- **Selector engine** — an HTML parser (`golang.org/x/net/html`, pure Go) plus a
  hand-written **CSS** matcher (type, `*`, `#id`, `.class`, `[attr]`, `[a=v]`,
  `~=` `^=` `$=` `*=` `|=`, descendant/child combinators, selector lists;
  pseudo-classes tolerated) and a practical **XPath** subset (`//`, `/`, `.//`
  steps; `@attr`, `text()`, `.`, `contains`, `starts-with`, `normalize-space`,
  `not`, `position`, `last`; `=`, `!=`, `and`, `or`). Semantic Capybara
  selectors (`:link`, `:button`, `:field`, `:checkbox`, `:radio`, `:select`)
  match a string locator by id, name, label, placeholder, value, or visible
  text the way the gem does.

CGO-free, one pure-Go dependency (`golang.org/x/net/html`), **100% test
coverage**, `gofmt` + `go vet` clean, and green across the six 64-bit Go targets
(amd64, arm64, riscv64, loong64, ppc64le, **s390x** — big-endian) plus `js/wasm`
and `wasip1/wasm`.

## Install

```sh
go get github.com/go-ruby-capybara/capybara
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-capybara/capybara"
)

func main() {
	// The application under test is any func(*Request) *Response.
	app := func(req *capybara.Request) *capybara.Response {
		switch req.Path {
		case "/":
			return capybara.NewResponse(200, "", `
				<h1>Welcome</h1>
				<a href="/login">Sign in</a>`)
		case "/login":
			return capybara.NewResponse(200, "", `
				<form action="/session" method="post">
					<label for="user">Username</label>
					<input type="text" id="user" name="user">
					<button type="submit">Log in</button>
				</form>`)
		default:
			return capybara.NewResponse(200, "", "<h1>Signed in</h1>")
		}
	}

	page := capybara.New(app)
	_ = page.Visit("/")
	fmt.Println(page.HasContent("Welcome")) // true

	_ = page.ClickLink("Sign in")
	_ = page.FillIn("Username", "ada")      // by <label> text
	_ = page.ClickButton("Log in")

	fmt.Println(page.CurrentPath())          // /session
	fmt.Println(page.HasContent("Signed in")) // true
}
```

### Scoping and assertions

```go
_ = page.Within("#sidebar", func() error {
	return page.AssertSelector("a.active")
})

node, _ := page.Find("table#users tr.row")
fmt.Println(node.Text(), node.AttrOr("data-id", ""))
```

## Value model

| gem                                        | this package                             |
| ------------------------------------------ | ---------------------------------------- |
| `Capybara::Session.new(:rack_test, app)`   | `capybara.New(app)`                      |
| `session.visit("/x")`                      | `session.Visit("/x")`                    |
| `session.click_link / click_button`        | `session.ClickLink / ClickButton`        |
| `session.fill_in(loc, with: v)`            | `session.FillIn(loc, v)`                 |
| `session.choose / check / uncheck / select`| `session.Choose / Check / Uncheck / Select` |
| `session.find(sel) / all(sel)`             | `session.Find(sel) / All(sel)`           |
| `session.has_content? / has_link?`         | `session.HasContent / HasLink`           |
| `session.within(scope) { … }`              | `session.Within(scope, fn)`              |
| `node.text / node[attr] / node.value`      | `node.Text() / node.Attr() / node.Value()` |
| the Rack app under test                    | `App` (host seam; a Go func in tests)    |
| `Capybara::ElementNotFound` / `Ambiguous`  | `*ElementNotFound` / `*Ambiguous`        |

## Tests & coverage

The suite is deterministic and browser-free: every test drives an in-memory Go
`App` func — no network, no headless browser. Field types, redirect limits,
CSS + XPath selector branches, within-scope, and the not-found / ambiguous /
expectation-not-met error paths are all exercised, so every arch and OS lane
holds coverage at **100%**.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-capybara/capybara authors.
