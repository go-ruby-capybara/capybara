// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"testing"
)

func TestCSSPseudoIgnored(t *testing.T) {
	// A pseudo-class is tolerated and dropped, matching on the rest.
	if got := cssMatchTags(t, `<a href="#" id="k">x</a>`, "a:hover"); len(got) != 1 {
		t.Fatalf("pseudo-class should be ignored: %v", got)
	}
	if got := cssMatchTags(t, `<input required id="k">`, "input:required"); len(got) != 1 {
		t.Fatalf("pseudo on input: %v", got)
	}
}

func TestMatchAttrUnknownOp(t *testing.T) {
	doc := parseHTML(t, `<a href="/x">y</a>`)
	a := elementsByTag(doc, "a")[0]
	if matchAttr(a, cssAttr{key: "href", op: "??", val: "/x"}) {
		t.Fatal("unknown operator should not match")
	}
}

func TestClassMismatchBranch(t *testing.T) {
	// element has class "card" but selector wants ".card.big" -> excluded.
	if got := cssMatchTags(t, `<p class="card">x</p>`, ".card.big"); len(got) != 0 {
		t.Fatalf("class mismatch should exclude: %v", got)
	}
}

func TestBodyAfterVisit(t *testing.T) {
	s := sessionWith(t, `<h1>Title</h1>`)
	if s.Body() == "" {
		t.Fatal("Body after visit should be non-empty")
	}
	if s.Response() == nil {
		t.Fatal("Response after visit should be set")
	}
}

func TestLinkNoHrefAndHidden(t *testing.T) {
	s := sessionWith(t, `
	  <a>No href here</a>
	  <a href="/vis">Visible</a>
	  <a href="/hid" style="display:none">Hidden</a>`)
	// "No href" anchor is skipped (no href), so no match.
	if s.HasLink("No href here") {
		t.Fatal("anchor without href should not be a link")
	}
	// hidden link is skipped.
	if s.HasLink("Hidden") {
		t.Fatal("hidden link should be skipped")
	}
	if !s.HasLink("Visible") {
		t.Fatal("visible link should match")
	}
}

func TestFieldTextareaSelectAndHidden(t *testing.T) {
	s := sessionWith(t, `
	  <textarea name="bio">x</textarea>
	  <select name="pick"><option>a</option></select>
	  <input type="hidden" name="secret" value="s">`)
	if _, err := s.FindField("bio"); err != nil {
		t.Fatalf("textarea field: %v", err)
	}
	if _, err := s.FindField("pick"); err != nil {
		t.Fatalf("select field: %v", err)
	}
	if s.HasField("secret") {
		t.Fatal("hidden input should not count as a field")
	}
}

func TestFillInInputNoType(t *testing.T) {
	// An input without a type attribute defaults to text and is fillable.
	s := sessionWith(t, `<input name="plain">`)
	if err := s.FillIn("plain", "hi"); err != nil {
		t.Fatalf("fill_in default text input: %v", err)
	}
	n, _ := s.Find("input")
	if n.Value() != "hi" {
		t.Fatalf("value not set: %q", n.Value())
	}
}

func TestHasFieldEmptyLocator(t *testing.T) {
	// An empty locator matches any field.
	s := sessionWith(t, `<input name="x"><input name="y">`)
	if !s.HasField("") {
		t.Fatal("empty field locator should match any field")
	}
}

func TestIsVisibleScript(t *testing.T) {
	s := sessionWith(t, `<script>var x=1;</script>`)
	nodes, _ := s.queryCSS("script")
	if len(nodes) != 1 || isVisible(nodes[0]) {
		t.Fatal("script element should be invisible")
	}
}

func TestNodeTextSkipsScript(t *testing.T) {
	s := sessionWith(t, `<div id="d">hi<script>var x=1;</script>bye</div>`)
	n, _ := s.Find("#d")
	if n.Text() != "hibye" {
		t.Fatalf("Node.Text should skip script: %q", n.Text())
	}
}

// --- xpath predicate coverage ---

func TestXPathEmptyNodeTestPredicate(t *testing.T) {
	// "//[@id]" -> empty node test becomes "*".
	if got := xpathMatch(t, `<a id="k" href="#">x</a>`, "//[@id='k']"); len(got) != 1 {
		t.Fatalf("empty node test should become wildcard: %v", got)
	}
}

func TestXPathTrailingJunkPredicate(t *testing.T) {
	if _, err := parseXPath("//a[@x @y]"); err == nil {
		t.Fatal("expected trailing-junk predicate error")
	}
}

func TestXPathBinaryRHSErrors(t *testing.T) {
	bad := []string{
		"//a[@x or ]",  // or with bad rhs
		"//a[@x and ]", // and with bad rhs
		"//a[@x=]",     // = with missing rhs
		"//a[@x!=]",    // != with missing rhs
	}
	for _, expr := range bad {
		if _, err := parseXPath(expr); err == nil {
			t.Errorf("expected error for %q", expr)
		}
	}
}

func TestXPathTruthyKinds(t *testing.T) {
	body := `<li id="a" data-n="1">x</li><li id="b">y</li>`
	// number truthy via 'and'
	if got := xpathMatch(t, body, "//li[1 and @id]"); len(got) == 0 {
		t.Fatalf("number truthy failed: %v", got)
	}
	// string truthy via normalize-space
	if got := xpathMatch(t, body, "//li[@id and normalize-space()]"); len(got) != 2 {
		t.Fatalf("string truthy failed: %v", got)
	}
}

func TestXPathNumOfBool(t *testing.T) {
	body := `<b id="x">y</b>`
	// not(@missing) -> bool true, compared numerically to 1.
	if got := xpathMatch(t, body, "//b[not(@missing)=1]"); len(got) != 1 {
		t.Fatalf("bool numeric coercion failed: %v", got)
	}
}

func TestXPathBareRelativePath(t *testing.T) {
	// A path with neither a leading '/' nor '.' is treated as relative.
	body := `<div id="d"><span id="s">x</span></div>`
	if got := xpathMatch(t, body, "html/body/div"); len(got) != 1 || got[0] != "div#d" {
		t.Fatalf("bare relative path: %v", got)
	}
}

func TestXPathParenBadContent(t *testing.T) {
	if _, err := parseXPath("//a[(=)]"); err == nil {
		t.Fatal("expected error for bad paren content")
	}
}

func TestXPathFuncBadFirstArg(t *testing.T) {
	if _, err := parseXPath("//a[contains(=,1)]"); err == nil {
		t.Fatal("expected error for bad function argument")
	}
}

func TestParsePredicateUnterminatedString(t *testing.T) {
	// Reached only via a direct predicate parse: matchBracket would otherwise
	// swallow the unterminated quote.
	if _, err := parsePredicate("'abc"); err == nil {
		t.Fatal("expected unterminated string error")
	}
}

func TestXPathNumOfBoolFalse(t *testing.T) {
	body := `<b id="x" k="v">y</b>`
	// not(@k) -> false (k present); compared numerically to 0.
	if got := xpathMatch(t, body, "//b[not(@k)=0]"); len(got) != 1 {
		t.Fatalf("bool-false numeric coercion failed: %v", got)
	}
}

func TestEvalPUnknownKind(t *testing.T) {
	doc := parseHTML(t, `<b>x</b>`)
	b := elementsByTag(doc, "b")[0]
	v := evalP(&pnode{kind: pkind(99)}, b, 1, 1)
	if v.isBool || v.isNum || v.str != "" {
		t.Fatalf("unknown kind should yield zero value: %#v", v)
	}
}
