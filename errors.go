// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import "fmt"

// ElementNotFound is returned when a finder that expects exactly one match
// finds none. It mirrors Capybara::ElementNotFound.
type ElementNotFound struct {
	// Query is a human description of what was searched for, e.g.
	// `link "Sign in"` or `css ".btn"`.
	Query string
}

func (e *ElementNotFound) Error() string {
	return fmt.Sprintf("Unable to find %s", e.Query)
}

// Ambiguous is returned when a singular finder matches more than one element.
// It mirrors Capybara::Ambiguous.
type Ambiguous struct {
	Query string
	Count int
}

func (e *Ambiguous) Error() string {
	return fmt.Sprintf("Ambiguous match, found %d elements matching %s", e.Count, e.Query)
}

// ExpectationNotMet is returned by the assert_* helpers when a positive or
// negative expectation fails. It mirrors Capybara::ExpectationNotMet.
type ExpectationNotMet struct {
	Message string
}

func (e *ExpectationNotMet) Error() string { return e.Message }

// InfiniteRedirect is returned when following redirects exceeds the driver's
// redirect limit. It mirrors Capybara::InfiniteRedirectError.
type InfiniteRedirect struct {
	Limit int
}

func (e *InfiniteRedirect) Error() string {
	return fmt.Sprintf("redirected more than %d times, check for infinite redirects", e.Limit)
}

// UnselectableError is returned when set/choose/check is called on a node whose
// tag cannot receive that interaction (e.g. filling in a <div>).
type UnselectableError struct {
	Message string
}

func (e *UnselectableError) Error() string { return e.Message }

// ParseError wraps a failure to parse an HTML response body.
type ParseError struct {
	Err error
}

func (e *ParseError) Error() string { return "capybara: parsing HTML response: " + e.Err.Error() }

func (e *ParseError) Unwrap() error { return e.Err }
