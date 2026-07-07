// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"errors"
	"testing"
)

const richPage = `
<div id="wrap">
  <nav>
    <a id="home" href="/" title="Home page">Home</a>
    <a href="/about">About Us</a>
    <a href="/logo"><img alt="Logo" src="/l.png"></a>
  </nav>
  <form action="/create" method="post">
    <label for="email">Email address</label>
    <input type="email" id="email" name="email" placeholder="you@example.com">
    <button type="submit" id="save" name="commit" value="Save" title="Save form">Save changes</button>
    <input type="submit" name="draft" value="Save draft">
    <input type="image" name="img" alt="Submit image">
  </form>
  <p class="msg">Welcome back</p>
  <p class="msg">Second message</p>
</div>`

func TestFindAllFirst(t *testing.T) {
	s := sessionWith(t, richPage)
	if _, err := s.Find(".msg"); err == nil {
		t.Fatal("expected Ambiguous for two .msg")
	} else {
		var amb *Ambiguous
		if !errors.As(err, &amb) || amb.Count != 2 {
			t.Fatalf("expected Ambiguous count 2, got %v", err)
		}
	}
	if _, err := s.Find(".nope"); err == nil {
		t.Fatal("expected ElementNotFound")
	}
	all, err := s.All(".msg")
	if err != nil || len(all) != 2 {
		t.Fatalf("All wrong: %v %d", err, len(all))
	}
	first, err := s.First(".msg")
	if err != nil || first.Text() != "Welcome back" {
		t.Fatalf("First wrong: %v", err)
	}
	if _, err := s.First(".nope"); err == nil {
		t.Fatal("First should error when none")
	}
	one, err := s.Find("#home")
	if err != nil || one.TagName() != "a" {
		t.Fatalf("single find failed: %v", err)
	}
}

func TestFindXPathAndAllXPath(t *testing.T) {
	s := sessionWith(t, richPage)
	n, err := s.FindXPath("//a[@id='home']")
	if err != nil || n.AttrOr("id", "") != "home" {
		t.Fatalf("FindXPath failed: %v", err)
	}
	all, err := s.AllXPath("//p[@class='msg']")
	if err != nil || len(all) != 2 {
		t.Fatalf("AllXPath wrong: %v %d", err, len(all))
	}
}

func TestQueryParseErrorPaths(t *testing.T) {
	s := sessionWith(t, richPage)
	if _, err := s.Find(""); err == nil {
		t.Fatal("Find empty should error")
	}
	if _, err := s.FindXPath(""); err == nil {
		t.Fatal("FindXPath empty should error")
	}
	if _, err := s.All(""); err == nil {
		t.Fatal("All empty should error")
	}
	if _, err := s.AllXPath(""); err == nil {
		t.Fatal("AllXPath empty should error")
	}
	if _, err := s.First(""); err == nil {
		t.Fatal("First empty should error")
	}
	if s.HasSelector("") {
		t.Fatal("HasSelector empty should be false")
	}
	if s.HasXPath("") {
		t.Fatal("HasXPath empty should be false")
	}
}

func TestMatchers(t *testing.T) {
	s := sessionWith(t, richPage)
	if !s.HasSelector(".msg") || s.HasNoSelector(".msg") {
		t.Fatal("HasSelector/.HasNoSelector wrong")
	}
	if !s.HasSelector("#home") {
		t.Fatal("HasSelector #home")
	}
	if !s.HasXPath("//nav/a") || s.HasNoXPath("//nav/a") {
		t.Fatal("HasXPath wrong")
	}
	if !s.HasContent("Welcome back") || !s.HasText("Second message") {
		t.Fatal("HasContent/HasText wrong")
	}
	if s.HasContent("nonexistent phrase") || !s.HasNoContent("nonexistent phrase") {
		t.Fatal("HasNoContent wrong")
	}
	if !s.HasLink("Home") || s.HasNoLink("Home") {
		t.Fatal("HasLink wrong")
	}
	if !s.HasButton("Save changes") || s.HasNoButton("Save changes") {
		t.Fatal("HasButton wrong")
	}
	if !s.HasField("email") || s.HasNoField("email") {
		t.Fatal("HasField wrong")
	}
	if s.HasNoField("Email address") {
		t.Fatal("HasField by label wrong")
	}
}

func TestAssertions(t *testing.T) {
	s := sessionWith(t, richPage)
	if err := s.AssertSelector(".msg"); err != nil {
		t.Fatalf("AssertSelector: %v", err)
	}
	if err := s.AssertSelector(".nope"); err == nil {
		t.Fatal("AssertSelector should fail")
	}
	if err := s.AssertNoSelector(".nope"); err != nil {
		t.Fatalf("AssertNoSelector: %v", err)
	}
	if err := s.AssertNoSelector(".msg"); err == nil {
		t.Fatal("AssertNoSelector should fail")
	}
	if err := s.AssertText("Welcome back"); err != nil {
		t.Fatalf("AssertText: %v", err)
	}
	if err := s.AssertText("absent text"); err == nil {
		t.Fatal("AssertText should fail")
	}
	if err := s.AssertNoText("absent text"); err != nil {
		t.Fatalf("AssertNoText: %v", err)
	}
	if err := s.AssertNoText("Welcome back"); err == nil {
		t.Fatal("AssertNoText should fail")
	}
	var enm *ExpectationNotMet
	if err := s.AssertSelector(".nope"); !errors.As(err, &enm) {
		t.Fatalf("expected ExpectationNotMet, got %v", err)
	}
}

func TestWithin(t *testing.T) {
	body := `
	<div id="a"><p class="hit">alpha</p></div>
	<div id="b"><p class="hit">beta</p><p class="hit">beta2</p></div>`
	s := sessionWith(t, body)
	// Inside #a there is exactly one .hit.
	err := s.Within("#a", func() error {
		n, err := s.Find(".hit")
		if err != nil {
			return err
		}
		if n.Text() != "alpha" {
			t.Fatalf("within scope wrong node: %q", n.Text())
		}
		if !s.HasContent("alpha") || s.HasContent("beta") {
			t.Fatal("within content scoping wrong")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Scope restored: now .hit is ambiguous across the whole document.
	if _, err := s.Find(".hit"); err == nil {
		t.Fatal("scope should be restored (ambiguous)")
	}
}

func TestWithinFindError(t *testing.T) {
	s := sessionWith(t, `<div>x</div>`)
	called := false
	err := s.Within("#missing", func() error { called = true; return nil })
	if err == nil || called {
		t.Fatal("Within should error before calling fn when scope not found")
	}
}

func TestWithinPropagatesFnError(t *testing.T) {
	s := sessionWith(t, `<div id="a">x</div>`)
	sentinel := errors.New("boom")
	if err := s.Within("#a", func() error { return sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("expected fn error, got %v", err)
	}
}

func TestLinkMatchingBranches(t *testing.T) {
	s := sessionWith(t, richPage)
	// by id
	if n, err := s.FindLink("home"); err != nil || n.AttrOr("id", "") != "home" {
		t.Fatalf("link by id: %v", err)
	}
	// by title
	if _, err := s.FindLink("Home page"); err != nil {
		t.Fatalf("link by title: %v", err)
	}
	// by text (substring)
	if _, err := s.FindLink("About"); err != nil {
		t.Fatalf("link by text: %v", err)
	}
	// by image alt
	if _, err := s.FindLink("Logo"); err != nil {
		t.Fatalf("link by image alt: %v", err)
	}
	// empty locator matches any -> ambiguous
	if _, err := s.FindLink(""); err == nil {
		t.Fatal("empty link locator should be ambiguous")
	}
	// ClickLink success
	if err := s.ClickLink("About"); err != nil {
		t.Fatalf("ClickLink: %v", err)
	}
	if s.CurrentPath() != "/about" {
		t.Fatalf("ClickLink navigation wrong: %q", s.CurrentPath())
	}
}

func TestButtonMatchingBranches(t *testing.T) {
	s := sessionWith(t, richPage)
	if _, err := s.FindButton("save"); err != nil { // id
		t.Fatalf("button by id: %v", err)
	}
	if _, err := s.FindButton("commit"); err != nil { // name
		t.Fatalf("button by name: %v", err)
	}
	if _, err := s.FindButton("Save form"); err != nil { // title
		t.Fatalf("button by title: %v", err)
	}
	if _, err := s.FindButton("Save draft"); err != nil { // input value
		t.Fatalf("button by input value: %v", err)
	}
	if _, err := s.FindButton("Submit image"); err != nil { // image alt
		t.Fatalf("button by image alt: %v", err)
	}
	if _, err := s.FindButton("Save changes"); err != nil { // button text
		t.Fatalf("button by text: %v", err)
	}
	if _, err := s.FindButton(""); err == nil { // ambiguous
		t.Fatal("empty button locator should be ambiguous")
	}
}

func TestClickOnLinkOrButton(t *testing.T) {
	s := sessionWith(t, richPage)
	// Matches a link
	if err := s.ClickOn("About"); err != nil {
		t.Fatalf("ClickOn link: %v", err)
	}
	if s.CurrentPath() != "/about" {
		t.Fatalf("ClickOn did not follow link: %q", s.CurrentPath())
	}
}

func TestClickOnButton(t *testing.T) {
	form := `<form action="/go" method="post"><button type="submit">Proceed</button></form>`
	s, last := captureApp(t, form)
	if err := s.ClickOn("Proceed"); err != nil {
		t.Fatalf("ClickOn button: %v", err)
	}
	if last.Method != "POST" {
		t.Fatalf("ClickOn button did not submit: %s", last.Method)
	}
}

func TestFieldByWrappingLabel(t *testing.T) {
	s := sessionWith(t, `<label>Username <input type="text" name="user"></label>`)
	if _, err := s.FindField("Username"); err != nil {
		t.Fatalf("field by wrapping label: %v", err)
	}
}

func TestFieldByPlaceholder(t *testing.T) {
	s := sessionWith(t, `<input type="text" name="q" placeholder="Search here">`)
	if _, err := s.FindField("Search here"); err != nil {
		t.Fatalf("field by placeholder: %v", err)
	}
}

func TestFieldAmbiguous(t *testing.T) {
	s := sessionWith(t, `<input name="dup"><input name="dup">`)
	if _, err := s.FindField("dup"); err == nil {
		t.Fatal("expected ambiguous field")
	} else {
		var amb *Ambiguous
		if !errors.As(err, &amb) {
			t.Fatalf("expected Ambiguous, got %v", err)
		}
	}
}

func TestVisibleTextSkipsHidden(t *testing.T) {
	s := sessionWith(t, `<p>shown</p><script>var x=1;</script><p style="display:none">gone</p>`)
	if !s.HasContent("shown") {
		t.Fatal("visible text missing shown")
	}
	if s.HasContent("var x") {
		t.Fatal("script text should be excluded")
	}
	if s.HasContent("gone") {
		t.Fatal("hidden text should be excluded")
	}
}
