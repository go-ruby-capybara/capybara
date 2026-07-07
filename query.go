// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"strings"

	"golang.org/x/net/html"
)

// root returns the current search root: the active within-scope element, or
// the whole document.
func (s *Session) root() *html.Node {
	if s.scope != nil {
		return s.scope
	}
	return s.doc
}

func (s *Session) wrap(n *html.Node) *Node { return &Node{node: n, session: s} }

// --- low-level queries ---

func (s *Session) queryCSS(sel string) ([]*html.Node, error) {
	c, err := parseCSS(sel)
	if err != nil {
		return nil, err
	}
	return c.matchAll(s.root(), false), nil
}

func (s *Session) queryXPath(sel string) ([]*html.Node, error) {
	x, err := parseXPath(sel)
	if err != nil {
		return nil, err
	}
	return x.eval(s.root(), documentRoot(s.root())), nil
}

func visibleNodes(nodes []*html.Node) []*html.Node {
	out := nodes[:0:0]
	for _, n := range nodes {
		if isVisible(n) {
			out = append(out, n)
		}
	}
	return out
}

func (s *Session) findOne(desc string, nodes []*html.Node, err error) (*Node, error) {
	if err != nil {
		return nil, err
	}
	nodes = visibleNodes(nodes)
	switch len(nodes) {
	case 0:
		return nil, &ElementNotFound{Query: desc}
	case 1:
		return s.wrap(nodes[0]), nil
	default:
		return nil, &Ambiguous{Query: desc, Count: len(nodes)}
	}
}

// --- public CSS / XPath finders ---

// Find returns the single visible element matching the CSS selector, or an
// error if none (ElementNotFound) or several (Ambiguous) match. It mirrors
// Session#find with the default :css selector.
func (s *Session) Find(sel string) (*Node, error) {
	nodes, err := s.queryCSS(sel)
	return s.findOne("css "+quote(sel), nodes, err)
}

// FindXPath is Find using an XPath selector (Session#find(:xpath, ...)).
func (s *Session) FindXPath(sel string) (*Node, error) {
	nodes, err := s.queryXPath(sel)
	return s.findOne("xpath "+quote(sel), nodes, err)
}

// All returns every visible element matching the CSS selector (Session#all).
func (s *Session) All(sel string) ([]*Node, error) {
	nodes, err := s.queryCSS(sel)
	if err != nil {
		return nil, err
	}
	return s.wrapAll(visibleNodes(nodes)), nil
}

// AllXPath is All using an XPath selector.
func (s *Session) AllXPath(sel string) ([]*Node, error) {
	nodes, err := s.queryXPath(sel)
	if err != nil {
		return nil, err
	}
	return s.wrapAll(visibleNodes(nodes)), nil
}

// First returns the first visible element matching the CSS selector, or an
// error if none match (Session#first).
func (s *Session) First(sel string) (*Node, error) {
	nodes, err := s.All(sel)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &ElementNotFound{Query: "css " + quote(sel)}
	}
	return nodes[0], nil
}

func (s *Session) wrapAll(nodes []*html.Node) []*Node {
	out := make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, s.wrap(n))
	}
	return out
}

// --- matchers ---

// HasSelector reports whether at least one visible element matches the CSS
// selector (Session#has_selector? / #has_css?).
func (s *Session) HasSelector(sel string) bool {
	nodes, err := s.queryCSS(sel)
	if err != nil {
		return false
	}
	return len(visibleNodes(nodes)) > 0
}

// HasNoSelector is the negation of HasSelector (Session#has_no_selector?).
func (s *Session) HasNoSelector(sel string) bool { return !s.HasSelector(sel) }

// HasXPath reports whether at least one visible element matches the XPath.
func (s *Session) HasXPath(sel string) bool {
	nodes, err := s.queryXPath(sel)
	if err != nil {
		return false
	}
	return len(visibleNodes(nodes)) > 0
}

// HasNoXPath is the negation of HasXPath.
func (s *Session) HasNoXPath(sel string) bool { return !s.HasXPath(sel) }

// HasContent reports whether the current scope's visible text contains text,
// after whitespace normalization (Session#has_content? / #has_text?).
func (s *Session) HasContent(text string) bool {
	return strings.Contains(visibleText(s.root()), normalizeSpace(text))
}

// HasText is an alias for HasContent.
func (s *Session) HasText(text string) bool { return s.HasContent(text) }

// HasNoContent is the negation of HasContent (Session#has_no_content?).
func (s *Session) HasNoContent(text string) bool { return !s.HasContent(text) }

// HasLink reports whether a matching link is present (Session#has_link?).
func (s *Session) HasLink(locator string) bool { return len(s.linkNodes(locator)) > 0 }

// HasNoLink is the negation of HasLink.
func (s *Session) HasNoLink(locator string) bool { return !s.HasLink(locator) }

// HasButton reports whether a matching button is present (Session#has_button?).
func (s *Session) HasButton(locator string) bool { return len(s.buttonNodes(locator)) > 0 }

// HasNoButton is the negation of HasButton.
func (s *Session) HasNoButton(locator string) bool { return !s.HasButton(locator) }

// HasField reports whether a matching field is present (Session#has_field?).
func (s *Session) HasField(locator string) bool {
	return len(s.fieldNodes(locator, isAnyField)) > 0
}

// HasNoField is the negation of HasField.
func (s *Session) HasNoField(locator string) bool { return !s.HasField(locator) }

// --- assertions ---

// AssertSelector returns an ExpectationNotMet error unless a visible element
// matches the CSS selector (Session#assert_selector).
func (s *Session) AssertSelector(sel string) error {
	if !s.HasSelector(sel) {
		return &ExpectationNotMet{Message: "expected to find css " + quote(sel) + " but there were no matches"}
	}
	return nil
}

// AssertNoSelector returns an error if any visible element matches
// (Session#assert_no_selector).
func (s *Session) AssertNoSelector(sel string) error {
	if s.HasSelector(sel) {
		return &ExpectationNotMet{Message: "expected not to find css " + quote(sel)}
	}
	return nil
}

// AssertText returns an ExpectationNotMet error unless the scope contains text
// (Session#assert_text).
func (s *Session) AssertText(text string) error {
	if !s.HasContent(text) {
		return &ExpectationNotMet{Message: "expected to find text " + quote(text) + " in " + quote(visibleText(s.root()))}
	}
	return nil
}

// AssertNoText returns an error if the scope contains text
// (Session#assert_no_text).
func (s *Session) AssertNoText(text string) error {
	if s.HasContent(text) {
		return &ExpectationNotMet{Message: "expected not to find text " + quote(text)}
	}
	return nil
}

// --- within ---

// Within restricts subsequent queries inside fn to the subtree of the single
// element matching the CSS selector, mirroring Session#within(scope){ }. The
// previous scope is restored afterwards, even if fn returns an error.
func (s *Session) Within(sel string, fn func() error) error {
	node, err := s.Find(sel)
	if err != nil {
		return err
	}
	prev := s.scope
	s.scope = node.node
	defer func() { s.scope = prev }()
	return fn()
}

// --- visible text ---

// visibleText returns the normalized visible text of the subtree, skipping
// hidden subtrees and script/style.
func visibleText(root *html.Node) string {
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			b.WriteString(n.Data)
			return
		case html.ElementNode:
			if strings.EqualFold(n.Data, "script") || strings.EqualFold(n.Data, "style") {
				return
			}
			if !isVisible(n) {
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return normalizeSpace(b.String())
}

func quote(s string) string { return "\"" + s + "\"" }
