// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"strings"

	"golang.org/x/net/html"
)

// cssSelector is a compiled comma-separated list of complex selectors. A node
// matches the list if it matches any of its members.
type cssSelector struct {
	groups []cssComplex
}

// cssComplex is a sequence of compound selectors joined by combinators, e.g.
// `div.card > a.link`. The final compound is the "subject": a node matches the
// complex selector when it satisfies the subject and the combinator chain
// walking leftward is satisfiable.
type cssComplex struct {
	subject cssCompound
	// ancestry lists the preceding (compound, combinator) pairs, nearest first.
	ancestry []cssStep
}

type cssStep struct {
	comb     byte // ' ' descendant, '>' child
	compound cssCompound
}

// cssCompound is a set of simple conditions that all apply to one element,
// e.g. `a.link#main[href]`.
type cssCompound struct {
	tag     string // "" means universal
	id      string
	classes []string
	attrs   []cssAttr
}

type cssAttr struct {
	key string
	op  string // "" (presence), "=", "~=", "^=", "$=", "*=", "|="
	val string
}

// parseCSS compiles a CSS selector string. It returns an error only for
// structurally empty input; unknown constructs degrade to never-match rather
// than panicking, matching how the driver tolerates odd generated selectors.
func parseCSS(sel string) (*cssSelector, error) {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return nil, &ElementNotFound{Query: "css \"\""}
	}
	out := &cssSelector{}
	for _, group := range splitTopLevel(sel, ',') {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		out.groups = append(out.groups, parseComplex(group))
	}
	if len(out.groups) == 0 {
		return nil, &ElementNotFound{Query: "css \"" + sel + "\""}
	}
	return out, nil
}

// splitTopLevel splits s on sep, ignoring occurrences inside [...] brackets.
func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// parseComplex parses a single complex selector into subject + ancestry.
func parseComplex(s string) cssComplex {
	toks := tokenizeCombinators(s)
	// toks alternates compound, combinator, compound, ...
	var compounds []cssCompound
	var combs []byte
	for i := 0; i < len(toks); i++ {
		if i%2 == 0 {
			compounds = append(compounds, parseCompound(toks[i]))
		} else {
			combs = append(combs, toks[i][0])
		}
	}
	cx := cssComplex{subject: compounds[len(compounds)-1]}
	// Build ancestry nearest-first.
	for i := len(compounds) - 2; i >= 0; i-- {
		cx.ancestry = append(cx.ancestry, cssStep{comb: combs[i], compound: compounds[i]})
	}
	return cx
}

// tokenizeCombinators splits a complex selector into alternating compound and
// combinator tokens. A combinator token is a single-char string "  " (space)
// or ">".
func tokenizeCombinators(s string) []string {
	var toks []string
	var cur strings.Builder
	depth := 0
	flush := func() {
		if cur.Len() > 0 {
			toks = append(toks, cur.String())
			cur.Reset()
		}
	}
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '[':
			depth++
			cur.WriteByte(c)
		case c == ']':
			if depth > 0 {
				depth--
			}
			cur.WriteByte(c)
		case depth == 0 && c == '>':
			flush()
			toks = append(toks, ">")
			i++
			// swallow surrounding spaces
			for i < len(s) && s[i] == ' ' {
				i++
			}
			continue
		case depth == 0 && (c == ' ' || c == '\t' || c == '\n'):
			// collapse run of whitespace into one descendant combinator, but
			// only if it separates two compounds (not trailing).
			j := i
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n') {
				j++
			}
			if j < len(s) && s[j] != '>' && cur.Len() > 0 {
				flush()
				toks = append(toks, " ")
			}
			i = j
			continue
		default:
			cur.WriteByte(c)
		}
		i++
	}
	flush()
	return toks
}

// parseCompound parses one compound selector such as `a.b#c[href^=/x]`.
// Pseudo-classes (a leading ':') are tolerated and ignored: the driver's
// generated selectors rarely use them, and dropping them degrades gracefully
// to matching on the remaining simple selectors.
func parseCompound(s string) cssCompound {
	var c cssCompound
	i := 0
	// leading type/universal
	if i < len(s) && s[i] != '.' && s[i] != '#' && s[i] != '[' && s[i] != ':' {
		j := i
		for j < len(s) && s[j] != '.' && s[j] != '#' && s[j] != '[' && s[j] != ':' {
			j++
		}
		name := s[i:j]
		if name != "*" {
			c.tag = strings.ToLower(name)
		}
		i = j
	}
	for i < len(s) {
		switch s[i] {
		case '.':
			j := i + 1
			for j < len(s) && s[j] != '.' && s[j] != '#' && s[j] != '[' && s[j] != ':' {
				j++
			}
			c.classes = append(c.classes, s[i+1:j])
			i = j
		case '#':
			j := i + 1
			for j < len(s) && s[j] != '.' && s[j] != '#' && s[j] != '[' && s[j] != ':' {
				j++
			}
			c.id = s[i+1 : j]
			i = j
		case '[':
			j := i + 1
			for j < len(s) && s[j] != ']' {
				j++
			}
			c.attrs = append(c.attrs, parseAttr(s[i+1:j]))
			if j < len(s) {
				j++ // consume ']'
			}
			i = j
		default:
			i++
		}
	}
	return c
}

// parseAttr parses the inside of an attribute selector such as `href^="/x"`.
func parseAttr(s string) cssAttr {
	s = strings.TrimSpace(s)
	for _, op := range []string{"~=", "^=", "$=", "*=", "|=", "="} {
		if idx := strings.Index(s, op); idx >= 0 {
			key := strings.TrimSpace(s[:idx])
			val := strings.TrimSpace(s[idx+len(op):])
			val = unquote(val)
			return cssAttr{key: strings.ToLower(key), op: op, val: val}
		}
	}
	return cssAttr{key: strings.ToLower(s)}
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// matchAll returns every element in the subtree rooted at root (excluding root
// itself unless includeRoot) that matches the selector, in document order.
func (sel *cssSelector) matchAll(root *html.Node, includeRoot bool) []*html.Node {
	var out []*html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (includeRoot || n != root) {
			if sel.matches(n) {
				out = append(out, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return out
}

func (sel *cssSelector) matches(n *html.Node) bool {
	for i := range sel.groups {
		if sel.groups[i].matches(n) {
			return true
		}
	}
	return false
}

func (cx *cssComplex) matches(n *html.Node) bool {
	if !cx.subject.matches(n) {
		return false
	}
	return matchAncestry(cx.ancestry, n)
}

// matchAncestry verifies the combinator chain leftward from node n.
func matchAncestry(steps []cssStep, n *html.Node) bool {
	if len(steps) == 0 {
		return true
	}
	step := steps[0]
	rest := steps[1:]
	switch step.comb {
	case '>':
		p := parentElement(n)
		if p != nil && step.compound.matches(p) && matchAncestry(rest, p) {
			return true
		}
		return false
	default: // ' ' descendant
		for p := parentElement(n); p != nil; p = parentElement(p) {
			if step.compound.matches(p) && matchAncestry(rest, p) {
				return true
			}
		}
		return false
	}
}

func parentElement(n *html.Node) *html.Node {
	if p := n.Parent; p != nil && p.Type == html.ElementNode {
		return p
	}
	return nil
}

func (c *cssCompound) matches(n *html.Node) bool {
	if c.tag != "" && !strings.EqualFold(n.Data, c.tag) {
		return false
	}
	if c.id != "" && getAttr(n, "id") != c.id {
		return false
	}
	if len(c.classes) > 0 {
		have := strings.Fields(getAttr(n, "class"))
		for _, want := range c.classes {
			if !containsStr(have, want) {
				return false
			}
		}
	}
	for _, a := range c.attrs {
		if !matchAttr(n, a) {
			return false
		}
	}
	return true
}

func matchAttr(n *html.Node, a cssAttr) bool {
	v, ok := lookupAttr(n, a.key)
	switch a.op {
	case "":
		return ok
	case "=":
		return ok && v == a.val
	case "~=":
		return ok && containsStr(strings.Fields(v), a.val)
	case "^=":
		return ok && a.val != "" && strings.HasPrefix(v, a.val)
	case "$=":
		return ok && a.val != "" && strings.HasSuffix(v, a.val)
	case "*=":
		return ok && a.val != "" && strings.Contains(v, a.val)
	case "|=":
		return ok && (v == a.val || strings.HasPrefix(v, a.val+"-"))
	}
	return false
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
