// Copyright (c) the go-ruby-capybara/capybara authors
//
// SPDX-License-Identifier: BSD-3-Clause

package capybara

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// xpathExpr is a compiled XPath location path. Only the practical subset
// Capybara's default selectors emit is supported: child (/) and descendant
// (//) steps, element or * node tests, and predicates built from @attr,
// text(), '.', string/number literals, the =, != comparisons, the contains,
// starts-with, normalize-space, not, position and last functions, and the
// and / or boolean connectives.
type xpathExpr struct {
	fromRoot bool
	steps    []xstep
}

type xstep struct {
	descendant bool // reached via '//'
	name       string
	preds      []*pnode
}

// parseXPath compiles an XPath location path.
func parseXPath(s string) (*xpathExpr, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, &ElementNotFound{Query: "xpath \"\""}
	}
	rel := false
	if strings.HasPrefix(s, ".") {
		rel = true
		s = s[1:]
	}
	if !strings.HasPrefix(s, "/") {
		rel = true
		s = "/" + s
	}
	expr := &xpathExpr{fromRoot: !rel}
	i := 0
	for i < len(s) && s[i] == '/' {
		descendant := false
		i++
		if i < len(s) && s[i] == '/' {
			descendant = true
			i++
		}
		start := i
		depth := 0
		var quote byte
		for i < len(s) {
			c := s[i]
			if quote != 0 {
				if c == quote {
					quote = 0
				}
				i++
				continue
			}
			switch c {
			case '\'', '"':
				quote = c
			case '[':
				depth++
			case ']':
				depth--
			case '/':
				if depth == 0 {
					goto done
				}
			}
			i++
		}
	done:
		st, err := parseStep(s[start:i], descendant)
		if err != nil {
			return nil, err
		}
		expr.steps = append(expr.steps, st)
	}
	return expr, nil
}

func parseStep(s string, descendant bool) (xstep, error) {
	st := xstep{descendant: descendant}
	// node test up to first '['
	idx := strings.IndexByte(s, '[')
	if idx < 0 {
		st.name = strings.TrimSpace(s)
	} else {
		st.name = strings.TrimSpace(s[:idx])
		rest := s[idx:]
		for len(rest) > 0 {
			if rest[0] != '[' {
				return st, &ElementNotFound{Query: "xpath predicate \"" + s + "\""}
			}
			end := matchBracket(rest)
			if end < 0 {
				return st, &ElementNotFound{Query: "xpath predicate \"" + s + "\""}
			}
			p, err := parsePredicate(rest[1:end])
			if err != nil {
				return st, err
			}
			st.preds = append(st.preds, p)
			rest = rest[end+1:]
		}
	}
	if st.name == "" {
		st.name = "*"
	}
	return st, nil
}

// matchBracket returns the index of the ']' matching the leading '[' of s.
func matchBracket(s string) int {
	depth := 0
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// eval returns all element nodes matching the expression within the subtree
// rooted at ctx (or the document root when fromRoot), in document order.
func (x *xpathExpr) eval(ctx, root *html.Node) []*html.Node {
	var current []*html.Node
	if x.fromRoot {
		current = []*html.Node{root}
	} else {
		current = []*html.Node{ctx}
	}
	for _, st := range x.steps {
		var next []*html.Node
		seen := map[*html.Node]bool{}
		for _, c := range current {
			var cand []*html.Node
			if st.descendant {
				cand = descendants(c)
			} else {
				cand = childElements(c)
			}
			var matched []*html.Node
			for _, e := range cand {
				if st.name != "*" && !strings.EqualFold(e.Data, st.name) {
					continue
				}
				matched = append(matched, e)
			}
			for pos, e := range matched {
				ok := true
				for _, p := range st.preds {
					if !predTrue(p, e, pos+1, len(matched)) {
						ok = false
						break
					}
				}
				if ok && !seen[e] {
					seen[e] = true
					next = append(next, e)
				}
			}
		}
		current = next
	}
	return current
}

func childElements(n *html.Node) []*html.Node {
	var out []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			out = append(out, c)
		}
	}
	return out
}

func descendants(n *html.Node) []*html.Node {
	var out []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			out = append(out, c)
			out = append(out, descendants(c)...)
		}
	}
	return out
}

// nodeString returns the string value of an element: its concatenated
// descendant text, normalized.
func nodeString(n *html.Node) string {
	var b strings.Builder
	collectRawText(n, &b)
	return normalizeSpace(b.String())
}

// immediateText returns the concatenation of the node's direct text children.
func immediateText(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	}
	return b.String()
}

// --- predicate expressions ---

type pkind int

const (
	pOr pkind = iota
	pAnd
	pEq
	pNe
	pLit
	pNum
	pAttr
	pText
	pDot
	pFunc
)

type pnode struct {
	kind pkind
	// binary
	left, right *pnode
	// leaf
	str string
	num float64
	// function
	fn   string
	args []*pnode
}

// parsePredicate parses a predicate expression string.
func parsePredicate(s string) (*pnode, error) {
	p := &pparser{src: s}
	p.skipSpace()
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos != len(p.src) {
		return nil, &ElementNotFound{Query: "xpath predicate \"" + s + "\""}
	}
	return node, nil
}

type pparser struct {
	src string
	pos int
}

func (p *pparser) skipSpace() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t') {
		p.pos++
	}
}

func (p *pparser) eatWord(w string) bool {
	p.skipSpace()
	if strings.HasPrefix(p.src[p.pos:], w) {
		end := p.pos + len(w)
		// word boundary
		if end == len(p.src) || !isNameByte(p.src[end]) {
			p.pos = end
			return true
		}
	}
	return false
}

func isNameByte(b byte) bool {
	return b == '-' || b == '_' || b == ':' ||
		(b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func (p *pparser) parseOr() (*pnode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.eatWord("or") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &pnode{kind: pOr, left: left, right: right}
	}
	return left, nil
}

func (p *pparser) parseAnd() (*pnode, error) {
	left, err := p.parseCmp()
	if err != nil {
		return nil, err
	}
	for p.eatWord("and") {
		right, err := p.parseCmp()
		if err != nil {
			return nil, err
		}
		left = &pnode{kind: pAnd, left: left, right: right}
	}
	return left, nil
}

func (p *pparser) parseCmp() (*pnode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos < len(p.src) {
		if p.src[p.pos] == '=' {
			p.pos++
			right, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			return &pnode{kind: pEq, left: left, right: right}, nil
		}
		if strings.HasPrefix(p.src[p.pos:], "!=") {
			p.pos += 2
			right, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			return &pnode{kind: pNe, left: left, right: right}, nil
		}
	}
	return left, nil
}

func (p *pparser) parsePrimary() (*pnode, error) {
	p.skipSpace()
	if p.pos >= len(p.src) {
		return nil, &ElementNotFound{Query: "xpath predicate: unexpected end"}
	}
	c := p.src[p.pos]
	switch {
	case c == '(':
		p.pos++
		node, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if p.pos >= len(p.src) || p.src[p.pos] != ')' {
			return nil, &ElementNotFound{Query: "xpath predicate: missing )"}
		}
		p.pos++
		return node, nil
	case c == '\'' || c == '"':
		return p.parseString()
	case c == '@':
		p.pos++
		name := p.parseName()
		return &pnode{kind: pAttr, str: name}, nil
	case c == '.':
		p.pos++
		return &pnode{kind: pDot}, nil
	case c >= '0' && c <= '9':
		return p.parseNumber()
	default:
		name := p.parseName()
		if name == "" {
			return nil, &ElementNotFound{Query: "xpath predicate: unexpected '" + string(c) + "'"}
		}
		p.skipSpace()
		if p.pos < len(p.src) && p.src[p.pos] == '(' {
			return p.parseFunc(name)
		}
		return nil, &ElementNotFound{Query: "xpath predicate: bare name \"" + name + "\""}
	}
}

func (p *pparser) parseString() (*pnode, error) {
	quote := p.src[p.pos]
	p.pos++
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != quote {
		p.pos++
	}
	if p.pos >= len(p.src) {
		return nil, &ElementNotFound{Query: "xpath predicate: unterminated string"}
	}
	str := p.src[start:p.pos]
	p.pos++ // closing quote
	return &pnode{kind: pLit, str: str}, nil
}

func (p *pparser) parseNumber() (*pnode, error) {
	start := p.pos
	for p.pos < len(p.src) && ((p.src[p.pos] >= '0' && p.src[p.pos] <= '9') || p.src[p.pos] == '.') {
		p.pos++
	}
	n, err := strconv.ParseFloat(p.src[start:p.pos], 64)
	if err != nil {
		return nil, &ElementNotFound{Query: "xpath predicate: bad number"}
	}
	return &pnode{kind: pNum, num: n}, nil
}

func (p *pparser) parseName() string {
	start := p.pos
	for p.pos < len(p.src) && isNameByte(p.src[p.pos]) {
		p.pos++
	}
	return p.src[start:p.pos]
}

func (p *pparser) parseFunc(name string) (*pnode, error) {
	p.pos++ // consume '('
	fn := &pnode{kind: pFunc, fn: name}
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] == ')' {
		p.pos++
		// special: text() behaves like a node accessor
		if name == "text" {
			return &pnode{kind: pText}, nil
		}
		return fn, nil
	}
	for {
		arg, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		fn.args = append(fn.args, arg)
		p.skipSpace()
		if p.pos < len(p.src) && p.src[p.pos] == ',' {
			p.pos++
			continue
		}
		break
	}
	p.skipSpace()
	if p.pos >= len(p.src) || p.src[p.pos] != ')' {
		return nil, &ElementNotFound{Query: "xpath predicate: missing ) in " + name}
	}
	p.pos++
	return fn, nil
}

// --- predicate evaluation ---

type pval struct {
	kind   pkind // pLit(string), pNum, or "bool" via pAnd sentinel
	str    string
	num    float64
	b      bool
	isBool bool
	isNum  bool
}

func predTrue(p *pnode, n *html.Node, pos, size int) bool {
	v := evalP(p, n, pos, size)
	if v.isBool {
		return v.b
	}
	if v.isNum {
		return v.num == float64(pos)
	}
	return v.str != ""
}

func evalP(p *pnode, n *html.Node, pos, size int) pval {
	switch p.kind {
	case pOr:
		return boolVal(truthy(evalP(p.left, n, pos, size)) || truthy(evalP(p.right, n, pos, size)))
	case pAnd:
		return boolVal(truthy(evalP(p.left, n, pos, size)) && truthy(evalP(p.right, n, pos, size)))
	case pEq:
		return boolVal(cmpEq(evalP(p.left, n, pos, size), evalP(p.right, n, pos, size)))
	case pNe:
		return boolVal(!cmpEq(evalP(p.left, n, pos, size), evalP(p.right, n, pos, size)))
	case pLit:
		return pval{str: p.str}
	case pNum:
		return pval{num: p.num, isNum: true}
	case pAttr:
		v, ok := lookupAttr(n, p.str)
		if !ok {
			return pval{isBool: true, b: false}
		}
		return pval{str: v}
	case pText:
		return pval{str: immediateText(n)}
	case pDot:
		return pval{str: nodeString(n)}
	case pFunc:
		return evalFunc(p, n, pos, size)
	}
	return pval{}
}

func evalFunc(p *pnode, n *html.Node, pos, size int) pval {
	switch p.fn {
	case "contains":
		a := strOf(evalP(p.args[0], n, pos, size))
		b := strOf(evalP(p.args[1], n, pos, size))
		return boolVal(strings.Contains(a, b))
	case "starts-with":
		a := strOf(evalP(p.args[0], n, pos, size))
		b := strOf(evalP(p.args[1], n, pos, size))
		return boolVal(strings.HasPrefix(a, b))
	case "not":
		return boolVal(!truthy(evalP(p.args[0], n, pos, size)))
	case "normalize-space":
		if len(p.args) == 0 {
			return pval{str: normalizeSpace(nodeString(n))}
		}
		return pval{str: normalizeSpace(strOf(evalP(p.args[0], n, pos, size)))}
	case "position":
		return pval{num: float64(pos), isNum: true}
	case "last":
		return pval{num: float64(size), isNum: true}
	}
	return pval{isBool: true, b: false}
}

func truthy(v pval) bool {
	if v.isBool {
		return v.b
	}
	if v.isNum {
		return v.num != 0
	}
	return v.str != ""
}

func strOf(v pval) string {
	if v.isBool {
		if v.b {
			return "true"
		}
		return "false"
	}
	if v.isNum {
		return strconv.FormatFloat(v.num, 'g', -1, 64)
	}
	return v.str
}

func cmpEq(a, b pval) bool {
	if a.isNum || b.isNum {
		return numOf(a) == numOf(b)
	}
	return strOf(a) == strOf(b)
}

func numOf(v pval) float64 {
	if v.isNum {
		return v.num
	}
	if v.isBool {
		if v.b {
			return 1
		}
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v.str), 64)
	if err != nil {
		return 0
	}
	return f
}

func boolVal(b bool) pval {
	return pval{isBool: true, b: b}
}
