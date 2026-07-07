// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func parseHTML(t *testing.T, body string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// cssMatchTags returns the tag names matched by sel over the parsed body.
func cssMatchTags(t *testing.T, body, sel string) []string {
	t.Helper()
	doc := parseHTML(t, body)
	c, err := parseCSS(sel)
	if err != nil {
		t.Fatalf("parseCSS(%q): %v", sel, err)
	}
	var out []string
	for _, n := range c.matchAll(doc, false) {
		out = append(out, strings.ToLower(n.Data)+"#"+getAttr(n, "id"))
	}
	return out
}

func TestCSSParseErrors(t *testing.T) {
	if _, err := parseCSS("   "); err == nil {
		t.Fatal("expected error for blank selector")
	}
	if _, err := parseCSS(","); err == nil {
		t.Fatal("expected error for comma-only selector")
	}
}

func TestCSSTypeIDClass(t *testing.T) {
	body := `<div id="a" class="card big"><p id="p1">x</p><p id="p2" class="card">y</p></div>`
	if got := cssMatchTags(t, body, "p"); len(got) != 2 {
		t.Fatalf("type selector: %v", got)
	}
	if got := cssMatchTags(t, body, "#p1"); len(got) != 1 || got[0] != "p#p1" {
		t.Fatalf("id selector: %v", got)
	}
	if got := cssMatchTags(t, body, ".card"); len(got) != 2 {
		t.Fatalf("class selector: %v", got)
	}
	if got := cssMatchTags(t, body, "div.card.big"); len(got) != 1 {
		t.Fatalf("compound: %v", got)
	}
	if got := cssMatchTags(t, body, ".card.big"); len(got) != 1 {
		t.Fatalf("multi class: %v", got)
	}
	if got := cssMatchTags(t, body, "*"); len(got) == 0 {
		t.Fatalf("universal matched nothing")
	}
}

func TestCSSAttributeOps(t *testing.T) {
	body := `<a href="/foo" data-x="one two" title="Hello world" rel="next" lang="en-US">L</a>`
	cases := []struct {
		sel  string
		want int
	}{
		{"[href]", 1},
		{"[href=/foo]", 1},
		{`[href="/foo"]`, 1},
		{"[data-x~=two]", 1},
		{"[data-x~=three]", 0},
		{"[href^=/f]", 1},
		{"[href^=/z]", 0},
		{"[href$=oo]", 1},
		{"[href$=zz]", 0},
		{"[title*=world]", 1},
		{"[title*=zzz]", 0},
		{"[lang|=en]", 1},
		{"[lang|=fr]", 0},
		{"[missing]", 0},
		{"[href^=]", 0},
		{"[href$=]", 0},
		{"[title*=]", 0},
	}
	for _, c := range cases {
		got := cssMatchTags(t, body, c.sel)
		if len(got) != c.want {
			t.Errorf("sel %q: got %d want %d", c.sel, len(got), c.want)
		}
	}
}

func TestCSSCombinators(t *testing.T) {
	body := `<div class="outer"><div class="inner"><a id="deep" href="#">x</a></div><a id="direct" href="#">y</a></div>`
	if got := cssMatchTags(t, body, "div.outer a"); len(got) != 2 {
		t.Fatalf("descendant: %v", got)
	}
	if got := cssMatchTags(t, body, "div.outer > a"); len(got) != 1 || got[0] != "a#direct" {
		t.Fatalf("child: %v", got)
	}
	if got := cssMatchTags(t, body, "div.inner > a"); len(got) != 1 || got[0] != "a#deep" {
		t.Fatalf("child inner: %v", got)
	}
	// child combinator that cannot be satisfied
	if got := cssMatchTags(t, body, "a > a"); len(got) != 0 {
		t.Fatalf("impossible child: %v", got)
	}
	// descendant that cannot be satisfied
	if got := cssMatchTags(t, body, "span a"); len(got) != 0 {
		t.Fatalf("impossible descendant: %v", got)
	}
}

func TestCSSSelectorList(t *testing.T) {
	body := `<p>a</p><span>b</span><b>c</b>`
	if got := cssMatchTags(t, body, "p, span"); len(got) != 2 {
		t.Fatalf("list: %v", got)
	}
	// list with a blank member (trailing comma) still parses
	if got := cssMatchTags(t, body, "p, "); len(got) != 1 {
		t.Fatalf("trailing comma: %v", got)
	}
	// comma inside attribute is not a separator
	body2 := `<a data-x="a,b" href="#">x</a>`
	if got := cssMatchTags(t, body2, `[data-x="a,b"]`); len(got) != 1 {
		t.Fatalf("comma in attr: %v", got)
	}
}

func TestCSSMatchAllIncludeRoot(t *testing.T) {
	doc := parseHTML(t, `<div id="r"><div id="c"></div></div>`)
	div := elementsByTag(doc, "div")[0] // outer div id=r
	c, _ := parseCSS("div")
	withRoot := c.matchAll(div, true)
	withoutRoot := c.matchAll(div, false)
	if len(withRoot) != len(withoutRoot)+1 {
		t.Fatalf("includeRoot should add the root: %d vs %d", len(withRoot), len(withoutRoot))
	}
}

func TestParentElementNilParent(t *testing.T) {
	doc := parseHTML(t, `<p>x</p>`)
	if parentElement(doc) != nil {
		t.Fatalf("document root has no element parent")
	}
}

func TestUnquote(t *testing.T) {
	if unquote(`"x"`) != "x" || unquote(`'y'`) != "y" || unquote("z") != "z" || unquote(`"`) != `"` {
		t.Fatal("unquote wrong")
	}
}
