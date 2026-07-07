// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"strings"
	"testing"
)

func xpathMatch(t *testing.T, body, expr string) []string {
	t.Helper()
	doc := parseHTML(t, body)
	x, err := parseXPath(expr)
	if err != nil {
		t.Fatalf("parseXPath(%q): %v", expr, err)
	}
	var out []string
	for _, n := range x.eval(doc, doc) {
		out = append(out, strings.ToLower(n.Data)+"#"+getAttr(n, "id"))
	}
	return out
}

func TestXPathParseErrors(t *testing.T) {
	if _, err := parseXPath("  "); err == nil {
		t.Fatal("expected error for blank xpath")
	}
	// unterminated predicate bracket
	if _, err := parseXPath("//a[@href"); err == nil {
		t.Fatal("expected error for unterminated predicate")
	}
	// text after predicate that isn't another predicate
	if _, err := parseXPath("//a[@x]y"); err == nil {
		t.Fatal("expected error for junk after predicate")
	}
}

func TestXPathAxes(t *testing.T) {
	body := `<div id="d"><span id="s"><a id="a" href="#">x</a></span></div>`
	if got := xpathMatch(t, body, "//a"); len(got) != 1 || got[0] != "a#a" {
		t.Fatalf("descendant from root: %v", got)
	}
	if got := xpathMatch(t, body, "/html/body/div"); len(got) != 1 || got[0] != "div#d" {
		t.Fatalf("absolute child path: %v", got)
	}
	if got := xpathMatch(t, body, ".//span"); len(got) != 1 {
		t.Fatalf("relative descendant: %v", got)
	}
	if got := xpathMatch(t, body, "//div/span"); len(got) != 1 || got[0] != "span#s" {
		t.Fatalf("mixed steps: %v", got)
	}
	// wildcard node test
	if got := xpathMatch(t, body, "//span/*"); len(got) != 1 || got[0] != "a#a" {
		t.Fatalf("wildcard: %v", got)
	}
	// name mismatch prunes
	if got := xpathMatch(t, body, "//p"); len(got) != 0 {
		t.Fatalf("no p elements: %v", got)
	}
}

func TestXPathPredicates(t *testing.T) {
	body := `
	<ul>
	  <li id="l1" class="a">one</li>
	  <li id="l2" class="b">two</li>
	  <li id="l3" class="a">three</li>
	</ul>
	<input id="i1" type="text" name="q">
	<input id="i2" type="password" name="p">`
	cases := []struct {
		expr string
		want int
	}{
		{"//li[@class='a']", 2},
		{"//li[@class!='a']", 1},
		{"//li[@id]", 3},
		{"//li[@missing]", 0},
		{"//li[1]", 1},
		{"//li[position()=2]", 1},
		{"//li[last()]", 1},
		{"//li[text()='two']", 1},
		{"//li[contains(text(),'thr')]", 1},
		{"//li[contains(.,'one')]", 1},
		{"//input[@type='text' or @type='password']", 2},
		{"//input[@type='text' and @name='q']", 1},
		{"//li[not(@class='a')]", 1},
		{"//li[starts-with(@id,'l')]", 3},
		{"//li[normalize-space(.)='one']", 1},
		{"//li[normalize-space()='two']", 1},
		{"//input[@type='email']", 0},
	}
	for _, c := range cases {
		got := xpathMatch(t, body, c.expr)
		if len(got) != c.want {
			t.Errorf("expr %q: got %d want %d (%v)", c.expr, len(got), c.want, got)
		}
	}
}

func TestXPathNumericCompare(t *testing.T) {
	body := `<span data-n="3">x</span><span data-n="5">y</span>`
	if got := xpathMatch(t, body, "//span[@data-n=3]"); len(got) != 1 {
		t.Fatalf("numeric eq: %v", got)
	}
	if got := xpathMatch(t, body, "//span[@data-n!=3]"); len(got) != 1 {
		t.Fatalf("numeric ne: %v", got)
	}
}

func TestXPathParenAndPrecedence(t *testing.T) {
	body := `<i id="1" a="1" b="1">x</i><i id="2" a="1">y</i><i id="3" b="1">z</i>`
	if got := xpathMatch(t, body, "//i[(@a='1') and (@b='1')]"); len(got) != 1 || got[0] != "i#1" {
		t.Fatalf("paren and: %v", got)
	}
}

func TestXPathPredicateInternalErrors(t *testing.T) {
	bad := []string{
		"//a[(@x]",         // missing close paren
		"//a['unterم",      // unterminated string (utf8)
		"//a[1.2.3]",       // bad number
		"//a[foo bar]",     // bare name (not function call)
		"//a[]",            // empty predicate -> unexpected end
		"//a[contains(@x]", // missing ) in function
		"//a[=]",           // unexpected token at primary
	}
	for _, expr := range bad {
		if _, err := parseXPath(expr); err == nil {
			t.Errorf("expected parse error for %q", expr)
		}
	}
}

func TestXPathFuncString(t *testing.T) {
	// strOf on bool and number via contains() over function results.
	body := `<b id="x">hi</b>`
	// not() returns bool; contains of bool string "true"
	if got := xpathMatch(t, body, "//b[contains('atrueb', not(@missing))]"); len(got) != 1 {
		t.Fatalf("bool strOf: %v", got)
	}
	// position() number stringified inside contains
	if got := xpathMatch(t, body, "//b[contains('1', position())]"); len(got) != 1 {
		t.Fatalf("num strOf: %v", got)
	}
	// unknown function -> false
	if got := xpathMatch(t, body, "//b[bogus()]"); len(got) != 0 {
		t.Fatalf("unknown func should be false: %v", got)
	}
}

func TestXPathNumOfFallback(t *testing.T) {
	// numOf on a non-numeric string is 0; compare data-n (absent -> bool false)
	body := `<b id="x" data-n="abc">y</b>`
	if got := xpathMatch(t, body, "//b[@data-n=0]"); len(got) != 1 {
		t.Fatalf("numOf non-numeric string should be 0: %v", got)
	}
}

func TestXPathDedup(t *testing.T) {
	// Two ancestors matching then a shared descendant via // should dedup.
	body := `<div class="x"><div class="x"><a id="deep" href="#">z</a></div></div>`
	got := xpathMatch(t, body, "//div//a")
	if len(got) != 1 {
		t.Fatalf("expected dedup to 1, got %v", got)
	}
}
