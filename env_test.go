// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"net/http"
	"reflect"
	"testing"
)

func TestRackEnv(t *testing.T) {
	r := &Request{
		Method: "POST",
		Path:   "/submit",
		Query:  "a=1",
		Body:   "x=y",
		Scheme: "http",
		Host:   "example.test",
		Header: http.Header{},
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("X-Custom", "hi")
	env := r.RackEnv()
	if env["REQUEST_METHOD"] != "POST" || env["PATH_INFO"] != "/submit" || env["QUERY_STRING"] != "a=1" {
		t.Fatalf("bad core env: %#v", env)
	}
	if env["CONTENT_TYPE"] != "application/x-www-form-urlencoded" {
		t.Fatalf("content type not mapped: %#v", env)
	}
	if env["HTTP_X_CUSTOM"] != "hi" {
		t.Fatalf("custom header not mapped: %#v", env)
	}
	if env["CONTENT_LENGTH"] != "3" {
		t.Fatalf("content length default wrong: %q", env["CONTENT_LENGTH"])
	}
	if env["rack.url_scheme"] != "http" || env["SERVER_NAME"] != "example.test" {
		t.Fatalf("server env wrong: %#v", env)
	}
}

func TestRackEnvExplicitContentLength(t *testing.T) {
	r := &Request{Method: "POST", Path: "/", Body: "abcd", Header: http.Header{}}
	r.Header.Set("Content-Length", "99")
	env := r.RackEnv()
	if env["CONTENT_LENGTH"] != "99" {
		t.Fatalf("explicit content length overwritten: %q", env["CONTENT_LENGTH"])
	}
}

func TestRackEnvNoBody(t *testing.T) {
	r := &Request{Method: "GET", Path: "/", Header: http.Header{}}
	env := r.RackEnv()
	if _, ok := env["CONTENT_LENGTH"]; ok {
		t.Fatalf("unexpected content length for empty body")
	}
}

func TestNewResponse(t *testing.T) {
	r := NewResponse(200, "", "<p>hi</p>")
	if r.Header.Get("Content-Type") != "text/html;charset=utf-8" {
		t.Fatalf("default content type wrong: %q", r.Header.Get("Content-Type"))
	}
	r2 := NewResponse(201, "application/json", "{}")
	if r2.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("explicit content type wrong")
	}
}

func TestResponseLocation(t *testing.T) {
	var r Response
	if r.location() != "" {
		t.Fatalf("nil header should give empty location")
	}
	r.Header = http.Header{}
	r.Header.Set("Location", "/next")
	if r.location() != "/next" {
		t.Fatalf("location not read")
	}
}

func TestResponseIsRedirect(t *testing.T) {
	for _, code := range []int{301, 302, 303, 307, 308} {
		if !(&Response{Status: code}).isRedirect() {
			t.Fatalf("status %d should be redirect", code)
		}
	}
	if (&Response{Status: 200}).isRedirect() {
		t.Fatalf("200 is not a redirect")
	}
}

func TestSortedHeaderKeys(t *testing.T) {
	r := &Response{Header: http.Header{}}
	r.Header.Set("B", "1")
	r.Header.Set("A", "2")
	got := r.sortedHeaderKeys()
	if !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Fatalf("keys not sorted: %v", got)
	}
}
