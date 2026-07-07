// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"errors"
	"testing"
)

func TestErrorMessages(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{&ElementNotFound{Query: "link \"x\""}, "Unable to find link \"x\""},
		{&Ambiguous{Query: "css \".a\"", Count: 3}, "Ambiguous match, found 3 elements matching css \".a\""},
		{&ExpectationNotMet{Message: "nope"}, "nope"},
		{&InfiniteRedirect{Limit: 5}, "redirected more than 5 times, check for infinite redirects"},
		{&UnselectableError{Message: "bad"}, "bad"},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.want {
			t.Errorf("got %q want %q", got, c.want)
		}
	}
}

func TestParseErrorUnwrap(t *testing.T) {
	inner := errors.New("boom")
	pe := &ParseError{Err: inner}
	if pe.Error() != "capybara: parsing HTML response: boom" {
		t.Fatalf("bad message: %q", pe.Error())
	}
	if !errors.Is(pe, inner) {
		t.Fatalf("unwrap failed")
	}
}
