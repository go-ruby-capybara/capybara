// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"strings"

	"golang.org/x/net/html"
)

// Node is a single element found in the current document. It mirrors
// Capybara::Node::Element: it wraps a DOM node and a back-reference to the
// Session so that interactions (click/set/...) can issue new requests.
type Node struct {
	node    *html.Node
	session *Session
}

// TagName returns the lower-case tag name, e.g. "a", "input". It mirrors
// Node#tag_name.
func (n *Node) TagName() string {
	return strings.ToLower(n.node.Data)
}

// Attr returns the value of the named attribute and whether it was present. It
// backs the Ruby Node#[] accessor.
func (n *Node) Attr(name string) (string, bool) {
	return lookupAttr(n.node, name)
}

// AttrOr returns the attribute value or a fallback when absent.
func (n *Node) AttrOr(name, fallback string) string {
	if v, ok := lookupAttr(n.node, name); ok {
		return v
	}
	return fallback
}

// Text returns the normalized visible text of the node, mirroring Node#text:
// descendant text nodes concatenated, runs of whitespace collapsed, trimmed.
// Text inside <script>/<style> and non-visible elements is excluded.
func (n *Node) Text() string {
	var b strings.Builder
	collectText(n.node, &b)
	return normalizeSpace(b.String())
}

// Value returns the node's value the way Node#value does: the value attribute
// for inputs, the concatenated text for a textarea, and the selected option's
// value (or values) for a select.
func (n *Node) Value() string {
	switch n.TagName() {
	case "textarea":
		var b strings.Builder
		collectRawText(n.node, &b)
		return b.String()
	case "select":
		if hasAttrFlag(n.node, "multiple") {
			var vals []string
			for _, opt := range n.options() {
				if optionSelected(opt) {
					vals = append(vals, optionValue(opt))
				}
			}
			return strings.Join(vals, ",")
		}
		for _, opt := range n.options() {
			if optionSelected(opt) {
				return optionValue(opt)
			}
		}
		// HTML default: first option when none marked selected.
		if opts := n.options(); len(opts) > 0 {
			return optionValue(opts[0])
		}
		return ""
	default:
		return n.AttrOr("value", "")
	}
}

// Checked reports whether a checkbox/radio is checked (Node#checked?).
func (n *Node) Checked() bool {
	return hasAttrFlag(n.node, "checked")
}

// Selected reports whether an option is selected (Node#selected?).
func (n *Node) Selected() bool {
	return optionSelected(n.node)
}

// Visible reports whether the node is visible, applying the same coarse rules
// as the rack_test driver: hidden inputs, type=hidden, display:none and the
// hidden attribute are invisible; everything else is visible.
func (n *Node) Visible() bool { return isVisible(n.node) }

// isVisible applies the rack_test driver's coarse visibility rules to a raw
// node: type=hidden inputs, the hidden attribute, display:none and
// script/style — on the node or any ancestor — make it invisible.
func isVisible(n *html.Node) bool {
	for cur := n; cur != nil && cur.Type == html.ElementNode; cur = parentElement(cur) {
		if strings.EqualFold(cur.Data, "input") && strings.EqualFold(getAttr(cur, "type"), "hidden") {
			return false
		}
		if hasAttrFlag(cur, "hidden") {
			return false
		}
		style := strings.ToLower(getAttr(cur, "style"))
		if strings.Contains(style, "display:none") || strings.Contains(style, "display: none") {
			return false
		}
		if strings.EqualFold(cur.Data, "script") || strings.EqualFold(cur.Data, "style") {
			return false
		}
	}
	return true
}

// Set assigns a value, mirroring Node#set. For text inputs and textareas it
// updates the value; for checkboxes/radios a bool-ish value toggles checked.
func (n *Node) Set(value string) error {
	switch n.TagName() {
	case "textarea":
		setTextareaValue(n.node, value)
		return nil
	case "input":
		typ := strings.ToLower(n.AttrOr("type", "text"))
		switch typ {
		case "checkbox", "radio":
			if value == "" || value == "false" {
				n.setChecked(false)
			} else {
				n.setChecked(true)
			}
			return nil
		default:
			setAttr(n.node, "value", value)
			return nil
		}
	default:
		return &UnselectableError{Message: "cannot set value on <" + n.TagName() + ">"}
	}
}

// Click performs the element's default action, mirroring Node#click: following
// a link's href, or submitting the form a button belongs to. Checkboxes and
// radios toggle. It returns an error if the element is not clickable.
func (n *Node) Click() error {
	switch n.TagName() {
	case "a":
		href, ok := n.Attr("href")
		if !ok {
			return &UnselectableError{Message: "link has no href"}
		}
		return n.session.followLink(href)
	case "button":
		bn, bv, bok := n.buttonNameValue()
		return n.session.submitFrom(n.node, bn, bv, bok)
	case "input":
		typ := strings.ToLower(n.AttrOr("type", "text"))
		switch typ {
		case "submit", "image":
			bn, bv, bok := n.buttonNameValue()
			return n.session.submitFrom(n.node, bn, bv, bok)
		case "checkbox":
			n.setChecked(!n.Checked())
			return nil
		case "radio":
			n.chooseRadio()
			return nil
		default:
			return &UnselectableError{Message: "input type " + typ + " is not clickable"}
		}
	default:
		return &UnselectableError{Message: "<" + n.TagName() + "> is not clickable"}
	}
}

// buttonNameValue returns the name=value pair a submit control contributes to
// the form data when it is the control that triggered submission.
func (n *Node) buttonNameValue() (string, string, bool) {
	name, ok := n.Attr("name")
	if !ok || name == "" {
		return "", "", false
	}
	return name, n.AttrOr("value", ""), true
}

func (n *Node) setChecked(v bool) {
	if v {
		setAttr(n.node, "checked", "checked")
	} else {
		removeAttr(n.node, "checked")
	}
}

// chooseRadio checks this radio and unchecks every other radio sharing its
// name within the same form.
func (n *Node) chooseRadio() {
	name := getAttr(n.node, "name")
	form := enclosingForm(n.node)
	root := form
	if root == nil {
		root = documentRoot(n.node)
	}
	for _, r := range elementsByTag(root, "input") {
		if strings.EqualFold(getAttr(r, "type"), "radio") && getAttr(r, "name") == name {
			removeAttr(r, "checked")
		}
	}
	setAttr(n.node, "checked", "checked")
}

func (n *Node) options() []*html.Node {
	return elementsByTag(n.node, "option")
}

// --- attribute + tree helpers shared across the package ---

func lookupAttr(n *html.Node, name string) (string, bool) {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val, true
		}
	}
	return "", false
}

func getAttr(n *html.Node, name string) string {
	v, _ := lookupAttr(n, name)
	return v
}

func hasAttrFlag(n *html.Node, name string) bool {
	_, ok := lookupAttr(n, name)
	return ok
}

func setAttr(n *html.Node, name, val string) {
	for i := range n.Attr {
		if strings.EqualFold(n.Attr[i].Key, name) {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: name, Val: val})
}

func removeAttr(n *html.Node, name string) {
	out := n.Attr[:0]
	for _, a := range n.Attr {
		if !strings.EqualFold(a.Key, name) {
			out = append(out, a)
		}
	}
	n.Attr = out
}

// collectText appends visible descendant text of n into b, skipping
// script/style and hidden subtrees.
func collectText(n *html.Node, b *strings.Builder) {
	if n.Type == html.TextNode {
		b.WriteString(n.Data)
		return
	}
	if n.Type == html.ElementNode {
		if strings.EqualFold(n.Data, "script") || strings.EqualFold(n.Data, "style") {
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, b)
	}
}

// collectRawText appends all descendant text without filtering (used for
// textarea/option values).
func collectRawText(n *html.Node, b *strings.Builder) {
	if n.Type == html.TextNode {
		b.WriteString(n.Data)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectRawText(c, b)
	}
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// setTextareaValue replaces a textarea's text content with value.
func setTextareaValue(n *html.Node, value string) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		n.RemoveChild(c)
		c = next
	}
	n.AppendChild(&html.Node{Type: html.TextNode, Data: value})
}

func optionValue(opt *html.Node) string {
	if v, ok := lookupAttr(opt, "value"); ok {
		return v
	}
	var b strings.Builder
	collectRawText(opt, &b)
	return normalizeSpace(b.String())
}

func optionLabel(opt *html.Node) string {
	var b strings.Builder
	collectRawText(opt, &b)
	return normalizeSpace(b.String())
}

func optionSelected(opt *html.Node) bool {
	return hasAttrFlag(opt, "selected")
}

// elementsByTag returns descendant elements (and root itself) with the tag.
func elementsByTag(root *html.Node, tag string) []*html.Node {
	var out []*html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, tag) {
			out = append(out, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return out
}

// enclosingForm returns the nearest ancestor <form>, or nil.
func enclosingForm(n *html.Node) *html.Node {
	for p := parentElement(n); p != nil; p = parentElement(p) {
		if strings.EqualFold(p.Data, "form") {
			return p
		}
	}
	return nil
}

// documentRoot walks to the top of the tree.
func documentRoot(n *html.Node) *html.Node {
	for n.Parent != nil {
		n = n.Parent
	}
	return n
}
