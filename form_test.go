// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"errors"
	"strings"
	"testing"
)

// captureApp serves a form page at /form and echoes the follow-up request.
func captureApp(t *testing.T, formHTML string) (*Session, *Request) {
	t.Helper()
	last := &Request{}
	app := func(req *Request) *Response {
		if req.Path == "/form" {
			return page(formHTML)
		}
		*last = *req
		return page("submitted")
	}
	s := New(app)
	if err := s.Visit("/form"); err != nil {
		t.Fatal(err)
	}
	return s, last
}

func TestFillInAndSubmitGET(t *testing.T) {
	form := `<form action="/search" method="get">
	  <input type="text" name="q" value="">
	  <input type="submit" value="Search">
	</form>`
	s, last := captureApp(t, form)
	if err := s.FillIn("q", "golang"); err != nil {
		t.Fatal(err)
	}
	if err := s.ClickButton("Search"); err != nil {
		t.Fatal(err)
	}
	if last.Method != "GET" {
		t.Fatalf("expected GET, got %s", last.Method)
	}
	if !strings.Contains(last.Query, "q=golang") {
		t.Fatalf("query missing field: %q", last.Query)
	}
	if s.CurrentPath() != "/search" {
		t.Fatalf("action path wrong: %q", s.CurrentPath())
	}
}

func TestFillInByLabelAndSubmitPOST(t *testing.T) {
	form := `<form action="/create" method="post">
	  <label for="name">Full Name</label>
	  <input type="text" id="name" name="name">
	  <label>Bio <textarea name="bio"></textarea></label>
	  <input type="hidden" name="token" value="abc">
	  <button type="submit" name="commit" value="Save">Save</button>
	</form>`
	s, last := captureApp(t, form)
	if err := s.FillIn("Full Name", "Ada"); err != nil {
		t.Fatal(err)
	}
	if err := s.FillIn("Bio", "hacker"); err != nil {
		t.Fatal(err)
	}
	if err := s.ClickButton("Save"); err != nil {
		t.Fatal(err)
	}
	if last.Method != "POST" {
		t.Fatalf("expected POST, got %s", last.Method)
	}
	for _, want := range []string{"name=Ada", "bio=hacker", "token=abc", "commit=Save"} {
		if !strings.Contains(last.Body, want) {
			t.Fatalf("body %q missing %q", last.Body, want)
		}
	}
}

func TestCheckUncheckChooseSelect(t *testing.T) {
	form := `<form action="/create" method="post">
	  <input type="checkbox" name="agree" id="agree" value="yes">
	  <input type="checkbox" name="news" id="news" value="1" checked>
	  <input type="radio" name="plan" id="free" value="free">
	  <input type="radio" name="plan" id="pro" value="pro">
	  <select name="color" id="color">
	    <option value="r">Red</option>
	    <option value="g">Green</option>
	  </select>
	  <input type="submit" value="Go">
	</form>`
	s, last := captureApp(t, form)
	if err := s.Check("agree"); err != nil {
		t.Fatal(err)
	}
	if err := s.Uncheck("news"); err != nil {
		t.Fatal(err)
	}
	if err := s.Choose("pro"); err != nil {
		t.Fatal(err)
	}
	if err := s.Select("Green", "color"); err != nil {
		t.Fatal(err)
	}
	if err := s.ClickButton("Go"); err != nil {
		t.Fatal(err)
	}
	body := last.Body
	if !strings.Contains(body, "agree=yes") {
		t.Fatalf("agree not submitted: %q", body)
	}
	if strings.Contains(body, "news=") {
		t.Fatalf("news should be unchecked: %q", body)
	}
	if !strings.Contains(body, "plan=pro") {
		t.Fatalf("plan not pro: %q", body)
	}
	if !strings.Contains(body, "color=g") {
		t.Fatalf("color not green: %q", body)
	}
}

func TestSelectByValueAttr(t *testing.T) {
	s, _ := captureApp(t, `<form action="/c"><select name="s" id="s"><option value="x">Ex</option><option value="y">Why</option></select></form>`)
	if err := s.Select("y", "s"); err != nil {
		t.Fatal(err)
	}
	n, _ := s.Find("#s")
	if n.Value() != "y" {
		t.Fatalf("select by value failed: %q", n.Value())
	}
}

func TestSelectOptionNotFound(t *testing.T) {
	s, _ := captureApp(t, `<form><select name="s" id="s"><option>A</option></select></form>`)
	err := s.Select("Nonexistent", "s")
	var nf *ElementNotFound
	if !errors.As(err, &nf) {
		t.Fatalf("expected ElementNotFound, got %v", err)
	}
}

func TestSelectMultipleKeepsOthers(t *testing.T) {
	s, _ := captureApp(t, `<form><select name="s" id="s" multiple><option value="a" selected>A</option><option value="b">B</option></select></form>`)
	if err := s.Select("B", "s"); err != nil {
		t.Fatal(err)
	}
	n, _ := s.Find("#s")
	if n.Value() != "a,b" {
		t.Fatalf("multiple select should keep prior selection: %q", n.Value())
	}
}

func TestAttachFile(t *testing.T) {
	s, last := captureApp(t, `<form action="/upload" method="post"><input type="file" name="doc" id="doc"><input type="submit" value="Up"></form>`)
	if err := s.AttachFile("doc", "/etc/hosts"); err != nil {
		t.Fatal(err)
	}
	if err := s.ClickButton("Up"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(last.Body, "doc=") {
		t.Fatalf("file field not submitted: %q", last.Body)
	}
}

func TestFindFieldAndErrors(t *testing.T) {
	s := sessionWith(t, `<input id="only" name="only" type="text">`)
	n, err := s.FindField("only")
	if err != nil || n.TagName() != "input" {
		t.Fatalf("find field failed: %v", err)
	}
	if _, err := s.FindField("missing"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestInteractionNotFoundErrors(t *testing.T) {
	s := sessionWith(t, `<div>nothing</div>`)
	if err := s.FillIn("x", "y"); err == nil {
		t.Fatal("fill_in should fail")
	}
	if err := s.Check("x"); err == nil {
		t.Fatal("check should fail")
	}
	if err := s.Uncheck("x"); err == nil {
		t.Fatal("uncheck should fail")
	}
	if err := s.Choose("x"); err == nil {
		t.Fatal("choose should fail")
	}
	if err := s.Select("v", "x"); err == nil {
		t.Fatal("select should fail")
	}
	if err := s.AttachFile("x", "/p"); err == nil {
		t.Fatal("attach should fail")
	}
	if err := s.ClickLink("x"); err == nil {
		t.Fatal("click_link should fail")
	}
	if err := s.ClickButton("x"); err == nil {
		t.Fatal("click_button should fail")
	}
	if err := s.ClickOn("x"); err == nil {
		t.Fatal("click_on should fail")
	}
}

func TestSubmitNoForm(t *testing.T) {
	// A submit button not inside a form yields an error on click.
	s := sessionWith(t, `<button type="submit" id="b">Orphan</button>`)
	b, _ := s.Find("#b")
	var ue *UnselectableError
	if err := b.Click(); !errors.As(err, &ue) {
		t.Fatalf("expected UnselectableError, got %v", err)
	}
}

func TestSubmitBadAction(t *testing.T) {
	s := sessionWith(t, `<form action="http://%zz"><button type="submit" id="b">Go</button></form>`)
	b, _ := s.Find("#b")
	if err := b.Click(); err == nil {
		t.Fatal("expected error for bad action url")
	}
}

func TestSerializeFormBranches(t *testing.T) {
	form := `<form>
	  <input type="text" name="a" value="1">
	  <input type="text" value="noname">
	  <input type="text" name="dis" value="x" disabled>
	  <input type="checkbox" name="c1" value="on1">
	  <input type="checkbox" name="c2" checked>
	  <input type="radio" name="r" value="r1">
	  <input type="submit" name="btn" value="Send">
	  <textarea name="ta">body</textarea>
	  <textarea disabled>skip</textarea>
	  <select name="sel"><option value="x" selected>X</option></select>
	  <select name="seldef"><option value="first">F</option><option value="second">S</option></select>
	  <select name="seldis" disabled><option selected>D</option></select>
	  <select><option>noname</option></select>
	</form>`
	doc := parseHTML(t, form)
	f := elementsByTag(doc, "form")[0]
	pairs := serializeForm(f)
	got := map[string]string{}
	for _, p := range pairs {
		got[p.k] = p.v
	}
	if got["a"] != "1" {
		t.Errorf("text field: %v", got)
	}
	if _, ok := got["dis"]; ok {
		t.Errorf("disabled input included")
	}
	if _, ok := got["c1"]; ok {
		t.Errorf("unchecked checkbox included")
	}
	if got["c2"] != "on" {
		t.Errorf("checked checkbox default value: %v", got)
	}
	if _, ok := got["r"]; ok {
		t.Errorf("unchecked radio included")
	}
	if _, ok := got["btn"]; ok {
		t.Errorf("submit button auto-included")
	}
	if got["ta"] != "body" {
		t.Errorf("textarea: %v", got)
	}
	if got["sel"] != "x" {
		t.Errorf("select selected: %v", got)
	}
	if got["seldef"] != "first" {
		t.Errorf("select default first: %v", got)
	}
	if _, ok := got["seldis"]; ok {
		t.Errorf("disabled select included")
	}
}

func TestEncodePairsEscaping(t *testing.T) {
	got := encodePairs([]formPair{{"a b", "c&d"}, {"x", "y"}})
	if got != "a+b=c%26d&x=y" {
		t.Fatalf("encodePairs wrong: %q", got)
	}
	if encodePairs(nil) != "" {
		t.Fatalf("empty encode should be empty")
	}
}
