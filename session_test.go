// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"errors"
	"io"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// staticApp serves a fixed set of pages keyed by path and records the last
// request for assertions.
type recordingApp struct {
	pages map[string]*Response
	last  *Request
}

func (a *recordingApp) call(req *Request) *Response {
	a.last = req
	if r, ok := a.pages[req.Path]; ok {
		return r
	}
	return NewResponse(404, "", "<html><body>not found</body></html>")
}

func page(body string) *Response { return NewResponse(200, "", "<html><body>"+body+"</body></html>") }

func redirect(loc string) *Response {
	r := NewResponse(302, "", "")
	r.Header.Set("Location", loc)
	return r
}

func TestVisitAndCurrentPath(t *testing.T) {
	app := &recordingApp{pages: map[string]*Response{
		"/hello": page("<h1>Hello</h1>"),
	}}
	s := New(app.call)
	if err := s.Visit("/hello?x=1"); err != nil {
		t.Fatal(err)
	}
	if s.CurrentPath() != "/hello" {
		t.Fatalf("current path %q", s.CurrentPath())
	}
	if app.last.Query != "x=1" {
		t.Fatalf("query not passed: %q", app.last.Query)
	}
	if s.CurrentURL() != "http://www.example.com/hello?x=1" {
		t.Fatalf("current url %q", s.CurrentURL())
	}
	if !s.HasContent("Hello") {
		t.Fatalf("expected content Hello, body=%q", s.Body())
	}
	if s.StatusCode() != 200 {
		t.Fatalf("status %d", s.StatusCode())
	}
}

func TestBodyStatusBeforeVisit(t *testing.T) {
	s := New(func(*Request) *Response { return page("") })
	if s.Body() != "" || s.StatusCode() != 0 || s.Response() != nil {
		t.Fatalf("expected zero state before visit")
	}
	if s.CurrentURL() != "http://www.example.com" {
		t.Fatalf("empty current url: %q", s.CurrentURL())
	}
}

func TestVisitFollowsRedirect(t *testing.T) {
	app := &recordingApp{pages: map[string]*Response{
		"/start": redirect("/dest"),
		"/dest":  page("arrived"),
	}}
	s := New(app.call)
	if err := s.Visit("/start"); err != nil {
		t.Fatal(err)
	}
	if s.CurrentPath() != "/dest" {
		t.Fatalf("did not follow redirect: %q", s.CurrentPath())
	}
	if !s.HasContent("arrived") {
		t.Fatalf("wrong page after redirect")
	}
}

func TestVisitRedirectWithQuery(t *testing.T) {
	app := &recordingApp{pages: map[string]*Response{
		"/start": redirect("/dest?ok=1"),
		"/dest":  page("arrived"),
	}}
	s := New(app.call)
	if err := s.Visit("/start"); err != nil {
		t.Fatal(err)
	}
	if s.curQuery != "ok=1" {
		t.Fatalf("redirect query lost: %q", s.curQuery)
	}
}

func TestInfiniteRedirect(t *testing.T) {
	app := &recordingApp{pages: map[string]*Response{
		"/loop": redirect("/loop"),
	}}
	s := New(app.call)
	err := s.Visit("/loop")
	var ir *InfiniteRedirect
	if !errors.As(err, &ir) {
		t.Fatalf("expected InfiniteRedirect, got %v", err)
	}
}

func TestRedirectBadLocation(t *testing.T) {
	app := &recordingApp{pages: map[string]*Response{
		"/start": redirect("http://%zz"),
	}}
	s := New(app.call)
	if err := s.Visit("/start"); err == nil {
		t.Fatalf("expected url parse error for bad location")
	}
}

func TestRedirectRelative(t *testing.T) {
	app := &recordingApp{pages: map[string]*Response{
		"/a/start": redirect("dest"),
		"/a/dest":  page("relative arrived"),
	}}
	s := New(app.call)
	if err := s.Visit("/a/start"); err != nil {
		t.Fatal(err)
	}
	if s.CurrentPath() != "/a/dest" {
		t.Fatalf("relative redirect resolved wrong: %q", s.CurrentPath())
	}
}

func TestParseErrorSeam(t *testing.T) {
	orig := htmlParse
	defer func() { htmlParse = orig }()
	htmlParse = func(io.Reader) (*html.Node, error) { return nil, errors.New("parse boom") }
	s := New(func(*Request) *Response { return page("x") })
	err := s.Visit("/")
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected ParseError, got %v", err)
	}
}

func TestVisitBadPathQuery(t *testing.T) {
	// A path without a query has empty query.
	app := &recordingApp{pages: map[string]*Response{"/p": page("ok")}}
	s := New(app.call)
	if err := s.Visit("/p"); err != nil {
		t.Fatal(err)
	}
	if app.last.Query != "" {
		t.Fatalf("expected empty query, got %q", app.last.Query)
	}
}

func TestFollowLinkBadHref(t *testing.T) {
	s := New(func(*Request) *Response { return page("x") })
	if err := s.Visit("/"); err != nil {
		t.Fatal(err)
	}
	if err := s.followLink("http://%zz"); err == nil {
		t.Fatalf("expected error for bad href")
	}
}

func TestResolveTargetHeaderContentType(t *testing.T) {
	// Ensure POST submission sets the urlencoded content type through the app.
	var seen *Request
	app := func(req *Request) *Response {
		seen = req
		if req.Path == "/form" {
			return page(`<form method="post" action="/create"><input name="a" value="1"><button type="submit">Go</button></form>`)
		}
		return page("done")
	}
	s := New(app)
	if err := s.Visit("/form"); err != nil {
		t.Fatal(err)
	}
	if err := s.ClickButton("Go"); err != nil {
		t.Fatal(err)
	}
	if seen.Method != "POST" {
		t.Fatalf("expected POST, got %s", seen.Method)
	}
	if seen.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Fatalf("missing urlencoded content type")
	}
	if !strings.Contains(seen.Body, "a=1") {
		t.Fatalf("body missing field: %q", seen.Body)
	}
}
