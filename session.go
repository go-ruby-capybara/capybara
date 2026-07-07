// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// htmlParse is the HTML parser seam. It is a package variable so tests can
// substitute a failing parser to exercise the ParseError path; strings.Reader
// itself never surfaces a parse error.
var htmlParse = html.Parse

// Session drives an application under test, mirroring Capybara::Session with
// the default rack_test driver. Construct one with [New].
type Session struct {
	app App

	// RedirectLimit bounds automatic redirect following (Capybara default 5).
	RedirectLimit int
	// Host is the SERVER_NAME used for requests (Capybara default
	// "www.example.com"; here trimmed to "www.example.com").
	Host string
	// Scheme is the URL scheme for requests.
	Scheme string

	resp     *Response
	doc      *html.Node
	curPath  string
	curQuery string
	scope    *html.Node
}

// New returns a Session driving app with Capybara's default configuration.
func New(app App) *Session {
	return &Session{
		app:           app,
		RedirectLimit: 5,
		Host:          "www.example.com",
		Scheme:        "http",
	}
}

// Response returns the most recent raw [Response], or nil before the first
// request. It backs Capybara's page.status_code / page.response_headers /
// page.body helpers.
func (s *Session) Response() *Response { return s.resp }

// Body returns the raw body of the last response.
func (s *Session) Body() string {
	if s.resp == nil {
		return ""
	}
	return s.resp.Body
}

// StatusCode returns the last response's HTTP status.
func (s *Session) StatusCode() int {
	if s.resp == nil {
		return 0
	}
	return s.resp.Status
}

// Visit issues a GET request for path (which may include a query string) and
// makes the response the current page, mirroring Session#visit.
func (s *Session) Visit(path string) error {
	p, q := splitPathQuery(path)
	return s.request("GET", p, q, "", nil)
}

// CurrentPath returns the path of the current page (Session#current_path).
func (s *Session) CurrentPath() string { return s.curPath }

// CurrentURL returns the full URL of the current page (Session#current_url).
func (s *Session) CurrentURL() string {
	u := &url.URL{Scheme: s.Scheme, Host: s.Host, Path: s.curPath, RawQuery: s.curQuery}
	return u.String()
}

// request performs a single request, follows redirects up to RedirectLimit,
// then parses and installs the resulting document as the current page.
func (s *Session) request(method, path, query, body string, header http.Header) error {
	req := &Request{
		Method: method, Path: path, Query: query, Body: body,
		Header: header, Scheme: s.Scheme, Host: s.Host,
	}
	resp := s.app(req)
	redirects := 0
	for resp.isRedirect() {
		redirects++
		if redirects > s.RedirectLimit {
			return &InfiniteRedirect{Limit: s.RedirectLimit}
		}
		loc := resp.location()
		np, nq, err := s.resolveTarget(path, loc)
		if err != nil {
			return err
		}
		path, query = np, nq
		req = &Request{Method: "GET", Path: path, Query: query, Scheme: s.Scheme, Host: s.Host}
		resp = s.app(req)
	}
	doc, err := parseDocument(resp.Body)
	if err != nil {
		return err
	}
	s.resp = resp
	s.doc = doc
	s.curPath = path
	s.curQuery = query
	s.scope = nil
	return nil
}

// parseDocument parses an HTML body into a document tree via the htmlParse
// seam.
func parseDocument(body string) (*html.Node, error) {
	doc, err := htmlParse(strings.NewReader(body))
	if err != nil {
		return nil, &ParseError{Err: err}
	}
	return doc, nil
}

// resolveTarget resolves href relative to the current path, returning the
// target path and query.
func (s *Session) resolveTarget(current, href string) (string, string, error) {
	base := &url.URL{Scheme: s.Scheme, Host: s.Host, Path: current}
	ref, err := url.Parse(href)
	if err != nil {
		return "", "", err
	}
	u := base.ResolveReference(ref)
	return u.Path, u.RawQuery, nil
}

// followLink issues a GET for a link's href, resolved against the current
// path.
func (s *Session) followLink(href string) error {
	p, q, err := s.resolveTarget(s.curPath, href)
	if err != nil {
		return err
	}
	return s.request("GET", p, q, "", nil)
}

// submitFrom submits the form enclosing control, optionally contributing the
// control's name/value pair (for submit buttons).
func (s *Session) submitFrom(control *html.Node, name, value string, has bool) error {
	form := enclosingForm(control)
	if form == nil {
		return &UnselectableError{Message: "control is not inside a form"}
	}
	method := strings.ToUpper(getAttr(form, "method"))
	if method == "" {
		method = "GET"
	}
	action := getAttr(form, "action")
	p, q, err := s.resolveTarget(s.curPath, action)
	if err != nil {
		return err
	}
	pairs := serializeForm(form)
	if has {
		pairs = append(pairs, formPair{name, value})
	}
	encoded := encodePairs(pairs)
	if method == "GET" {
		return s.request("GET", p, encoded, "", nil)
	}
	h := http.Header{}
	h.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.request(method, p, q, encoded, h)
}

// splitPathQuery splits "path?query" into its two parts.
func splitPathQuery(s string) (string, string) {
	if i := strings.IndexByte(s, '?'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
