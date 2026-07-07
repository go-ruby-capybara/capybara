// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// App is the seam between Capybara and the application under test. It is the
// Go analogue of a Rack app: it maps a [Request] to a [Response]. In tests it
// is a plain in-memory func; in the go-embedded-ruby binding it wraps a real
// Ruby Rack app (converting to/from the Rack `env` hash).
type App func(*Request) *Response

// Request is the driver's representation of a single HTTP request to the app.
// It carries enough to build the equivalent Rack `env` hash in the binding.
type Request struct {
	// Method is the upper-case HTTP verb ("GET", "POST", ...).
	Method string
	// Path is the request path (PATH_INFO), always starting with "/".
	Path string
	// Query is the raw query string, without a leading "?".
	Query string
	// Body is the raw request body (rack.input).
	Body string
	// Header holds request headers such as Content-Type.
	Header http.Header
	// Scheme is "http" or "https" (rack.url_scheme).
	Scheme string
	// Host is the server name (SERVER_NAME).
	Host string
}

// RackEnv renders the request as a Rack-style env map. The binding hands this
// to the wrapped Ruby app; the pure-Go tests never need it, but it documents
// exactly how a [Request] maps onto Rack and is convenient for app authors.
func (r *Request) RackEnv() map[string]string {
	env := map[string]string{
		"REQUEST_METHOD":  r.Method,
		"PATH_INFO":       r.Path,
		"QUERY_STRING":    r.Query,
		"SCRIPT_NAME":     "",
		"SERVER_NAME":     r.Host,
		"rack.url_scheme": r.Scheme,
		"rack.input":      r.Body,
	}
	for k := range r.Header {
		v := r.Header.Get(k)
		switch http.CanonicalHeaderKey(k) {
		case "Content-Type":
			env["CONTENT_TYPE"] = v
		case "Content-Length":
			env["CONTENT_LENGTH"] = v
		default:
			env["HTTP_"+strings.ToUpper(strings.ReplaceAll(k, "-", "_"))] = v
		}
	}
	if r.Body != "" {
		if _, ok := env["CONTENT_LENGTH"]; !ok {
			env["CONTENT_LENGTH"] = strconv.Itoa(len(r.Body))
		}
	}
	return env
}

// Response is the app's reply: an HTTP status, headers, and a body. It is the
// Go analogue of a Rack response triple `[status, headers, body]`.
type Response struct {
	Status int
	Header http.Header
	Body   string
}

// NewResponse is a convenience constructor for app authors and tests. The
// Content-Type defaults to text/html when contentType is empty.
func NewResponse(status int, contentType, body string) *Response {
	h := http.Header{}
	if contentType == "" {
		contentType = "text/html;charset=utf-8"
	}
	h.Set("Content-Type", contentType)
	return &Response{Status: status, Header: h, Body: body}
}

// location returns the (possibly empty) Location header, used for redirects.
func (r *Response) location() string {
	if r.Header == nil {
		return ""
	}
	return r.Header.Get("Location")
}

// isRedirect reports whether the status is one Capybara's rack_test driver
// follows automatically.
func (r *Response) isRedirect() bool {
	switch r.Status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	}
	return false
}

// sortedHeaderKeys is a small helper used by tests and diagnostics to render a
// Response deterministically.
func (r *Response) sortedHeaderKeys() []string {
	keys := make([]string, 0, len(r.Header))
	for k := range r.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
