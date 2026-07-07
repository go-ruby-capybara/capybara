// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"errors"
	"testing"
)

// sessionWith returns a session whose current document is the given body.
func sessionWith(t *testing.T, body string) *Session {
	t.Helper()
	s := New(func(*Request) *Response { return page(body) })
	if err := s.Visit("/"); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNodeBasics(t *testing.T) {
	s := sessionWith(t, `<a id="lnk" href="/x" data-role="nav">Click <b>me</b></a>`)
	n, err := s.Find("a")
	if err != nil {
		t.Fatal(err)
	}
	if n.TagName() != "a" {
		t.Fatalf("tag %q", n.TagName())
	}
	if v, ok := n.Attr("href"); !ok || v != "/x" {
		t.Fatalf("attr href %q %v", v, ok)
	}
	if _, ok := n.Attr("nope"); ok {
		t.Fatalf("missing attr reported present")
	}
	if n.AttrOr("data-role", "d") != "nav" || n.AttrOr("absent", "d") != "d" {
		t.Fatalf("AttrOr wrong")
	}
	if n.Text() != "Click me" {
		t.Fatalf("text %q", n.Text())
	}
}

func TestNodeValueInput(t *testing.T) {
	s := sessionWith(t, `<input id="i" value="hello">`)
	n, _ := s.Find("#i")
	if n.Value() != "hello" {
		t.Fatalf("input value %q", n.Value())
	}
}

func TestNodeValueTextarea(t *testing.T) {
	s := sessionWith(t, `<textarea id="t">line1
line2</textarea>`)
	n, _ := s.Find("#t")
	if n.Value() != "line1\nline2" {
		t.Fatalf("textarea value %q", n.Value())
	}
}

func TestNodeValueSelect(t *testing.T) {
	s := sessionWith(t, `<select id="s"><option value="a">A</option><option value="b" selected>B</option></select>`)
	n, _ := s.Find("#s")
	if n.Value() != "b" {
		t.Fatalf("select value %q", n.Value())
	}
}

func TestNodeValueSelectDefaultFirst(t *testing.T) {
	s := sessionWith(t, `<select id="s"><option value="a">A</option><option value="b">B</option></select>`)
	n, _ := s.Find("#s")
	if n.Value() != "a" {
		t.Fatalf("expected first option default, got %q", n.Value())
	}
}

func TestNodeValueSelectEmpty(t *testing.T) {
	s := sessionWith(t, `<select id="s"></select>`)
	n, _ := s.Find("#s")
	if n.Value() != "" {
		t.Fatalf("empty select value %q", n.Value())
	}
}

func TestNodeValueSelectMultiple(t *testing.T) {
	s := sessionWith(t, `<select id="s" multiple><option value="a" selected>A</option><option value="b" selected>B</option><option value="c">C</option></select>`)
	n, _ := s.Find("#s")
	if n.Value() != "a,b" {
		t.Fatalf("multiple value %q", n.Value())
	}
}

func TestNodeOptionValueFromText(t *testing.T) {
	s := sessionWith(t, `<select id="s"><option selected>Plain Text</option></select>`)
	n, _ := s.Find("#s")
	if n.Value() != "Plain Text" {
		t.Fatalf("option without value attr should use text, got %q", n.Value())
	}
}

func TestNodeCheckedSelected(t *testing.T) {
	s := sessionWith(t, `<input id="c" type="checkbox" checked><select><option id="o" selected>x</option></select>`)
	c, _ := s.Find("#c")
	if !c.Checked() {
		t.Fatal("checkbox should be checked")
	}
	o, _ := s.Find("#o")
	if !o.Selected() {
		t.Fatal("option should be selected")
	}
}

func TestNodeVisible(t *testing.T) {
	s := sessionWith(t, `
	  <p id="vis">visible</p>
	  <input id="hid" type="hidden" value="x">
	  <p id="disp" style="display: none">hidden</p>
	  <p id="attr" hidden>hidden</p>
	  <div style="display:none"><span id="nested">deep</span></div>`)
	if n, _ := s.First("#vis"); !n.Visible() {
		t.Fatal("#vis should be visible")
	}
	for _, sel := range []string{"#hid", "#disp", "#attr", "#nested"} {
		nodes, _ := s.queryCSS(sel)
		if len(nodes) != 1 {
			t.Fatalf("expected one node for %s", sel)
		}
		if isVisible(nodes[0]) {
			t.Fatalf("%s should be invisible", sel)
		}
	}
}

func TestNodeSet(t *testing.T) {
	s := sessionWith(t, `
	  <input id="txt" type="text" value="old">
	  <textarea id="ta">old</textarea>
	  <input id="cb" type="checkbox">
	  <div id="d"></div>`)
	txt, _ := s.Find("#txt")
	if err := txt.Set("new"); err != nil {
		t.Fatal(err)
	}
	if txt.Value() != "new" {
		t.Fatalf("text set failed: %q", txt.Value())
	}
	ta, _ := s.Find("#ta")
	if err := ta.Set("fresh"); err != nil {
		t.Fatal(err)
	}
	if ta.Value() != "fresh" {
		t.Fatalf("textarea set failed: %q", ta.Value())
	}
	cb, _ := s.Find("#cb")
	if err := cb.Set("true"); err != nil {
		t.Fatal(err)
	}
	if !cb.Checked() {
		t.Fatal("checkbox should be checked after set true")
	}
	if err := cb.Set("false"); err != nil {
		t.Fatal(err)
	}
	if cb.Checked() {
		t.Fatal("checkbox should be unchecked after set false")
	}
	d, _ := s.Find("#d")
	var ue *UnselectableError
	if err := d.Set("x"); !errors.As(err, &ue) {
		t.Fatalf("expected UnselectableError for div, got %v", err)
	}
}

func TestNodeClickErrors(t *testing.T) {
	s := sessionWith(t, `
	  <a id="nohref">no href</a>
	  <input id="txtclick" type="text">
	  <span id="span">x</span>`)
	nohref, _ := s.Find("#nohref")
	if err := nohref.Click(); err == nil {
		t.Fatal("expected error clicking hrefless link")
	}
	txt, _ := s.Find("#txtclick")
	if err := txt.Click(); err == nil {
		t.Fatal("expected error clicking text input")
	}
	span, _ := s.Find("#span")
	if err := span.Click(); err == nil {
		t.Fatal("expected error clicking span")
	}
}

func TestNodeClickCheckboxRadio(t *testing.T) {
	s := sessionWith(t, `
	  <input id="cb" type="checkbox">
	  <input id="r1" type="radio" name="g" value="1">
	  <input id="r2" type="radio" name="g" value="2">`)
	cb, _ := s.Find("#cb")
	if err := cb.Click(); err != nil {
		t.Fatal(err)
	}
	if !cb.Checked() {
		t.Fatal("checkbox toggle on")
	}
	if err := cb.Click(); err != nil {
		t.Fatal(err)
	}
	if cb.Checked() {
		t.Fatal("checkbox toggle off")
	}
	r1, _ := s.Find("#r1")
	if err := r1.Click(); err != nil {
		t.Fatal(err)
	}
	r2, _ := s.Find("#r2")
	if err := r2.Click(); err != nil {
		t.Fatal(err)
	}
	if r1.Checked() {
		t.Fatal("r1 should be unchecked after r2 chosen")
	}
	if !r2.Checked() {
		t.Fatal("r2 should be checked")
	}
}

func TestChooseRadioNoForm(t *testing.T) {
	// radios not wrapped in a form still deselect siblings via document root.
	s := sessionWith(t, `<input id="r1" type="radio" name="g" checked><input id="r2" type="radio" name="g">`)
	r2, _ := s.Find("#r2")
	r2.chooseRadio()
	r1, _ := s.Find("#r1")
	if r1.Checked() || !r2.Checked() {
		t.Fatal("radio group without form not handled")
	}
}

func TestButtonWithoutName(t *testing.T) {
	s := sessionWith(t, `<form action="/go"><button type="submit">Go</button></form>`)
	b, _ := s.Find("button")
	_, _, ok := b.buttonNameValue()
	if ok {
		t.Fatal("button without name should not contribute")
	}
}
