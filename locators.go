// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"strings"

	"golang.org/x/net/html"
)

// allElements returns every element in the current scope, in document order,
// excluding the scope root itself.
func (s *Session) allElements() []*html.Node {
	root := s.root()
	var out []*html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n != root {
			out = append(out, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return out
}

// --- links ---

func (s *Session) linkNodes(locator string) []*html.Node {
	var out []*html.Node
	for _, n := range s.allElements() {
		if !strings.EqualFold(n.Data, "a") {
			continue
		}
		if !hasAttrFlag(n, "href") {
			continue
		}
		if !isVisible(n) {
			continue
		}
		if linkMatches(n, locator) {
			out = append(out, n)
		}
	}
	return out
}

func linkMatches(n *html.Node, locator string) bool {
	if locator == "" {
		return true
	}
	if getAttr(n, "id") == locator || getAttr(n, "title") == locator {
		return true
	}
	if strings.Contains(normalizeSpace(nodeString(n)), locator) {
		return true
	}
	for _, img := range elementsByTag(n, "img") {
		if getAttr(img, "alt") == locator {
			return true
		}
	}
	return false
}

// FindLink returns the single link matching locator (id, title, text or image
// alt), mirroring find(:link, locator).
func (s *Session) FindLink(locator string) (*Node, error) {
	return s.findOne("link "+quote(locator), s.linkNodes(locator), nil)
}

// ClickLink finds a link by locator and follows its href (Session#click_link).
func (s *Session) ClickLink(locator string) error {
	node, err := s.FindLink(locator)
	if err != nil {
		return err
	}
	return node.Click()
}

// --- buttons ---

func (s *Session) buttonNodes(locator string) []*html.Node {
	var out []*html.Node
	for _, n := range s.allElements() {
		if !isButton(n) || !isVisible(n) {
			continue
		}
		if buttonMatches(n, locator) {
			out = append(out, n)
		}
	}
	return out
}

func isButton(n *html.Node) bool {
	switch strings.ToLower(n.Data) {
	case "button":
		return true
	case "input":
		switch strings.ToLower(getAttr(n, "type")) {
		case "submit", "reset", "button", "image":
			return true
		}
	}
	return false
}

func buttonMatches(n *html.Node, locator string) bool {
	if locator == "" {
		return true
	}
	if getAttr(n, "id") == locator || getAttr(n, "name") == locator || getAttr(n, "title") == locator {
		return true
	}
	if strings.EqualFold(n.Data, "input") {
		if getAttr(n, "value") == locator || getAttr(n, "alt") == locator {
			return true
		}
	}
	if strings.EqualFold(n.Data, "button") {
		if strings.Contains(normalizeSpace(nodeString(n)), locator) {
			return true
		}
	}
	return false
}

// FindButton returns the single button matching locator, mirroring
// find(:button, locator).
func (s *Session) FindButton(locator string) (*Node, error) {
	return s.findOne("button "+quote(locator), s.buttonNodes(locator), nil)
}

// ClickButton finds a button by locator and submits its form
// (Session#click_button).
func (s *Session) ClickButton(locator string) error {
	node, err := s.FindButton(locator)
	if err != nil {
		return err
	}
	return node.Click()
}

// ClickOn clicks a link or a button matching locator, whichever exists,
// mirroring Session#click_on / #click_link_or_button.
func (s *Session) ClickOn(locator string) error {
	nodes := append(s.linkNodes(locator), s.buttonNodes(locator)...)
	node, err := s.findOne("link or button "+quote(locator), nodes, nil)
	if err != nil {
		return err
	}
	return node.Click()
}

// --- fields ---

type fieldPred func(*html.Node) bool

func isAnyField(n *html.Node) bool {
	switch strings.ToLower(n.Data) {
	case "textarea", "select":
		return true
	case "input":
		return !strings.EqualFold(getAttr(n, "type"), "hidden")
	}
	return false
}

func isFillableField(n *html.Node) bool {
	switch strings.ToLower(n.Data) {
	case "textarea":
		return true
	case "input":
		switch strings.ToLower(inputType(n)) {
		case "checkbox", "radio", "submit", "reset", "button", "image", "file", "hidden":
			return false
		default:
			return true
		}
	}
	return false
}

func isCheckbox(n *html.Node) bool {
	return strings.EqualFold(n.Data, "input") && strings.EqualFold(inputType(n), "checkbox")
}

func isRadio(n *html.Node) bool {
	return strings.EqualFold(n.Data, "input") && strings.EqualFold(inputType(n), "radio")
}

func isSelect(n *html.Node) bool { return strings.EqualFold(n.Data, "select") }

func isFileField(n *html.Node) bool {
	return strings.EqualFold(n.Data, "input") && strings.EqualFold(inputType(n), "file")
}

func inputType(n *html.Node) string {
	if t := getAttr(n, "type"); t != "" {
		return t
	}
	return "text"
}

func (s *Session) fieldNodes(locator string, pred fieldPred) []*html.Node {
	var out []*html.Node
	for _, n := range s.allElements() {
		if !pred(n) {
			continue
		}
		if s.fieldMatches(n, locator) {
			out = append(out, n)
		}
	}
	return out
}

func (s *Session) fieldMatches(n *html.Node, locator string) bool {
	if locator == "" {
		return true
	}
	if getAttr(n, "id") == locator || getAttr(n, "name") == locator || getAttr(n, "placeholder") == locator {
		return true
	}
	return s.labelMatch(n, locator)
}

// labelMatch reports whether the field is associated with a <label> whose text
// contains locator, either via for=id or by wrapping the field.
func (s *Session) labelMatch(field *html.Node, locator string) bool {
	if id := getAttr(field, "id"); id != "" {
		for _, lbl := range elementsByTag(s.root(), "label") {
			if getAttr(lbl, "for") == id && strings.Contains(normalizeSpace(nodeString(lbl)), locator) {
				return true
			}
		}
	}
	for p := parentElement(field); p != nil; p = parentElement(p) {
		if strings.EqualFold(p.Data, "label") && strings.Contains(normalizeSpace(nodeString(p)), locator) {
			return true
		}
	}
	return false
}

// FindField returns the single field matching locator (id, name, placeholder
// or label), mirroring find(:field, locator).
func (s *Session) FindField(locator string) (*Node, error) {
	return s.findOne("field "+quote(locator), s.fieldNodes(locator, isAnyField), nil)
}

// FillIn finds a fillable field by locator and sets its value to with,
// mirroring Session#fill_in(locator, with: value).
func (s *Session) FillIn(locator, with string) error {
	node, err := s.findOne("fillable field "+quote(locator), s.fieldNodes(locator, isFillableField), nil)
	if err != nil {
		return err
	}
	return node.Set(with)
}

// Choose selects a radio button by locator (Session#choose).
func (s *Session) Choose(locator string) error {
	node, err := s.findOne("radio button "+quote(locator), s.fieldNodes(locator, isRadio), nil)
	if err != nil {
		return err
	}
	node.chooseRadio()
	return nil
}

// Check checks a checkbox by locator (Session#check).
func (s *Session) Check(locator string) error {
	node, err := s.findOne("checkbox "+quote(locator), s.fieldNodes(locator, isCheckbox), nil)
	if err != nil {
		return err
	}
	node.setChecked(true)
	return nil
}

// Uncheck unchecks a checkbox by locator (Session#uncheck).
func (s *Session) Uncheck(locator string) error {
	node, err := s.findOne("checkbox "+quote(locator), s.fieldNodes(locator, isCheckbox), nil)
	if err != nil {
		return err
	}
	node.setChecked(false)
	return nil
}

// AttachFile finds a file input by locator and sets its value to path,
// mirroring Session#attach_file. Multipart encoding of the upload is out of
// scope for this subset; the path is submitted as the field value.
func (s *Session) AttachFile(locator, path string) error {
	node, err := s.findOne("file field "+quote(locator), s.fieldNodes(locator, isFileField), nil)
	if err != nil {
		return err
	}
	setAttr(node.node, "value", path)
	return nil
}

// Select chooses the option labelled value within the select identified by
// from, mirroring Session#select(value, from: locator). When the select is
// single-valued, previously selected options are cleared.
func (s *Session) Select(value, from string) error {
	node, err := s.findOne("select "+quote(from), s.fieldNodes(from, isSelect), nil)
	if err != nil {
		return err
	}
	sel := node.node
	var target *html.Node
	for _, opt := range elementsByTag(sel, "option") {
		if optionLabel(opt) == value || optionValue(opt) == value {
			target = opt
			break
		}
	}
	if target == nil {
		return &ElementNotFound{Query: "option " + quote(value)}
	}
	if !hasAttrFlag(sel, "multiple") {
		for _, opt := range elementsByTag(sel, "option") {
			removeAttr(opt, "selected")
		}
	}
	setAttr(target, "selected", "selected")
	return nil
}
