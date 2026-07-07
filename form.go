// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// formPair is a single name/value pair contributed to a form submission.
type formPair struct {
	k, v string
}

// serializeForm collects the successful controls of a form in document order,
// following the HTML form-submission rules the rack_test driver relies on:
// disabled controls are skipped, unchecked checkboxes/radios contribute
// nothing, a select contributes its selected option(s) (or its first option
// when none is marked and it is single-valued), and buttons contribute nothing
// unless they triggered the submission (handled by the caller).
func serializeForm(form *html.Node) []formPair {
	var pairs []formPair
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "input":
				pairs = appendInput(pairs, n)
			case "textarea":
				if name := getAttr(n, "name"); name != "" && !hasAttrFlag(n, "disabled") {
					var b strings.Builder
					collectRawText(n, &b)
					pairs = append(pairs, formPair{name, b.String()})
				}
			case "select":
				pairs = appendSelect(pairs, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(form)
	return pairs
}

func appendInput(pairs []formPair, n *html.Node) []formPair {
	if hasAttrFlag(n, "disabled") {
		return pairs
	}
	name := getAttr(n, "name")
	if name == "" {
		return pairs
	}
	typ := strings.ToLower(getAttr(n, "type"))
	if typ == "" {
		typ = "text"
	}
	switch typ {
	case "submit", "button", "reset", "image":
		// Only the activating button contributes, handled by the caller.
		return pairs
	case "checkbox", "radio":
		if !hasAttrFlag(n, "checked") {
			return pairs
		}
		val := getAttr(n, "value")
		if val == "" {
			val = "on"
		}
		return append(pairs, formPair{name, val})
	default:
		return append(pairs, formPair{name, getAttr(n, "value")})
	}
}

func appendSelect(pairs []formPair, n *html.Node) []formPair {
	if hasAttrFlag(n, "disabled") {
		return pairs
	}
	name := getAttr(n, "name")
	if name == "" {
		return pairs
	}
	opts := elementsByTag(n, "option")
	multiple := hasAttrFlag(n, "multiple")
	any := false
	for _, opt := range opts {
		if optionSelected(opt) {
			pairs = append(pairs, formPair{name, optionValue(opt)})
			any = true
		}
	}
	if !any && !multiple && len(opts) > 0 {
		pairs = append(pairs, formPair{name, optionValue(opts[0])})
	}
	return pairs
}

// encodePairs URL-encodes the pairs in order, preserving document order (which
// url.Values would not).
func encodePairs(pairs []formPair) string {
	var b strings.Builder
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(p.k))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(p.v))
	}
	return b.String()
}
