// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package capybara is a pure-Go (CGO-free) reimplementation of the
// deterministic core of Ruby's `capybara` gem together with its default,
// browser-free `:rack_test` driver.
//
// Capybara is an acceptance-test DSL: you drive a web application the way a
// user would — visit a path, click a link, fill in a form, assert that the
// resulting page contains some text — without ever touching a real browser.
// The default rack_test driver does this entirely in-process: it speaks to a
// [Rack] application directly (no sockets, no JavaScript), parses each HTML
// response, and mutates an in-memory DOM as the "user" interacts with it.
// Every follow-up request (a link's href, a form's action) is reconstructed
// from that DOM and handed back to the app.
//
// This package reproduces that model in Go. The application under test is an
// injectable seam — an [App] func that maps a [Request] to a [Response] — so
// tests drive a plain Go function and never reach a network. In the
// go-embedded-ruby binding the seam wraps a real Ruby Rack app: the [Request]
// is converted to a Rack `env` hash, `app.call(env)` is invoked, and the
// `[status, headers, body]` triple is converted back into a [Response].
//
// # Ruby surface it mirrors
//
//   - Capybara::Session#visit / #current_path / #current_url
//   - #click_link / #click_button / #click_on
//   - #fill_in / #choose / #check / #uncheck / #select / #attach_file /
//     #find_field
//   - #find / #all / #first
//   - #has_selector? / #has_content? / #has_text? / #has_link? / #has_button? /
//     #has_field? / #has_css? / #has_xpath? and their negations
//   - #assert_selector / #assert_text and friends
//   - #within(scope) { ... }
//   - Node#text / #[] / #value / #click / #set / #tag_name / #checked? /
//     #selected? / #visible?
//
// # Selectors
//
// The heart of the driver is an HTML parser (golang.org/x/net/html, pure Go)
// plus a query engine. Two low-level selector kinds are supported directly:
//
//   - CSS: type, universal (*), #id, .class, [attr], [attr=v], [attr~=v],
//     [attr^=v], [attr$=v], [attr*=v], [attr|=v], the descendant ( ) and
//     child (>) combinators, and comma-separated selector lists.
//   - XPath: a practical subset — // and / and .// steps, element or *
//     node tests, and predicates over @attr, text(), contains(), position,
//     and and/or.
//
// On top of those, the semantic Capybara selectors (:link, :button, :field,
// :fillable_field, :checkbox, :radio_button, :select, :option,
// :link_or_button, :id, :css, :xpath) are implemented so that a plain string
// locator matches by id, name, label text, placeholder or visible text the way
// the gem does.
//
// [Rack]: https://github.com/rack/rack
package capybara
