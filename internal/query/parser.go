package query

import (
	"strings"
	"time"
)

// Expr is a query AST node. String renders a fully-parenthesized canonical
// form (prefix notation) so precedence and associativity are testable as exact
// strings, independent of the source spacing.
type Expr interface {
	String() string
}

type orExpr struct{ left, right Expr }
type andExpr struct{ left, right Expr }
type notExpr struct{ x Expr }

// predExpr is a `field cmp value` comparison. For a date field (created,
// updated) date holds the parsed value; valuePos is the value token's offset,
// used when epic resolution fails at compile time.
type predExpr struct {
	field    string
	op       token
	value    string
	date     time.Time // parsed value, valid only for a date field
	valuePos int
}

// termExpr is a bare word or quoted string that falls through to a
// case-insensitive title substring match.
type termExpr struct{ text string }

func (e *orExpr) String() string   { return "(or " + e.left.String() + " " + e.right.String() + ")" }
func (e *andExpr) String() string  { return "(and " + e.left.String() + " " + e.right.String() + ")" }
func (e *notExpr) String() string  { return "(not " + e.x.String() + ")" }
func (e *predExpr) String() string { return "(" + e.field + " " + e.op.cmpText() + " " + e.value + ")" }
func (e *termExpr) String() string { return "(term " + e.text + ")" }

// The fields the grammar recognizes on the left of a comparison
// (docs/design/04-cli.md §4).
const (
	fieldState    = "state"
	fieldCategory = "category"
	fieldOwner    = "owner"
	fieldLabel    = "label"
	fieldType     = "type"
	fieldEpic     = "epic"
	fieldPriority = "priority"
	fieldCreated  = "created"
	fieldUpdated  = "updated"
)

var fields = map[string]bool{
	fieldState: true, fieldCategory: true, fieldOwner: true, fieldLabel: true,
	fieldType: true, fieldEpic: true, fieldPriority: true, fieldCreated: true, fieldUpdated: true,
}

// isDateField reports whether f compares as a date and thus admits the ordering
// operators; every other field admits only = and !=.
func isDateField(f string) bool { return f == fieldCreated || f == fieldUpdated }

// keyword reports whether an unquoted word is the given logical keyword,
// case-insensitively. A quoted string is never a keyword, so a title term of
// "and"/"or"/"not" is written with quotes.
func isKeyword(t token, kw string) bool {
	return t.kind == tokWord && strings.EqualFold(t.text, kw)
}

func anyKeyword(t token) bool {
	return isKeyword(t, "AND") || isKeyword(t, "OR") || isKeyword(t, "NOT")
}

// Parse lexes and parses input into an AST, validating field/operator legality
// and date literals. It does not resolve epic references (that needs the item
// set) — see Compile. Errors are *Error with a source position.
func Parse(input string) (Expr, error) {
	toks, err := lex(input)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	if p.peek().kind == tokEOF {
		return nil, &Error{Pos: 0, Msg: "empty query"}
	}
	e, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if t := p.peek(); t.kind != tokEOF {
		return nil, &Error{Pos: t.pos, Msg: "unexpected " + describe(t)}
	}
	return e, nil
}

type parser struct {
	toks []token
	i    int
}

func (p *parser) peek() token { return p.toks[p.i] }
func (p *parser) next() token { t := p.toks[p.i]; p.i++; return t }

func (p *parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for isKeyword(p.peek(), "OR") {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &orExpr{left, right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Expr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if isKeyword(t, "AND") {
			p.next() // explicit AND
		} else if !p.startsOperand(t) {
			return left, nil // adjacency: another operand follows, or we stop
		}
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &andExpr{left, right}
	}
}

// startsOperand reports whether t can begin another and-operand: a '(', a
// string, or a word other than the OR/AND keywords (AND is consumed explicitly
// by the caller; NOT legitimately begins a not_expr).
func (p *parser) startsOperand(t token) bool {
	switch t.kind {
	case tokLParen, tokString:
		return true
	case tokWord:
		return !isKeyword(t, "OR") && !isKeyword(t, "AND")
	default:
		return false
	}
}

func (p *parser) parseNot() (Expr, error) {
	if isKeyword(p.peek(), "NOT") {
		p.next()
		x, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &notExpr{x}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Expr, error) {
	t := p.peek()
	switch t.kind {
	case tokLParen:
		p.next()
		e, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRParen {
			return nil, &Error{Pos: p.peek().pos, Msg: "expected ')'"}
		}
		p.next()
		return e, nil
	case tokWord:
		if anyKeyword(t) {
			return nil, &Error{Pos: t.pos, Msg: "unexpected keyword " + strings.ToUpper(t.text)}
		}
		if fields[t.text] && p.toks[p.i+1].isCmp() {
			return p.parsePredicate()
		}
		p.next()
		return &termExpr{t.text}, nil
	case tokString:
		p.next()
		return &termExpr{t.text}, nil
	default:
		return nil, &Error{Pos: t.pos, Msg: "expected a term, field comparison, or '('"}
	}
}

func (p *parser) parsePredicate() (Expr, error) {
	field := p.next()
	op := p.next()
	if !isDateField(field.text) && op.kind != tokEq && op.kind != tokNe {
		return nil, &Error{Pos: op.pos, Msg: "operator " + op.cmpText() + " is only valid on created/updated"}
	}
	val := p.peek()
	if val.kind != tokWord && val.kind != tokString {
		return nil, &Error{Pos: val.pos, Msg: "expected a value after " + op.cmpText()}
	}
	p.next()
	pred := &predExpr{field: field.text, op: op, value: val.text, valuePos: val.pos}
	if isDateField(field.text) {
		d, err := parseDate(val.text)
		if err != nil {
			return nil, &Error{Pos: val.pos, Msg: "invalid date " + quote(val.text)}
		}
		pred.date = d
	}
	return pred, nil
}

// parseDate accepts a full RFC3339 timestamp or a bare YYYY-MM-DD date (taken
// as midnight UTC). Comparisons operate on the resulting instant.
func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

func describe(t token) string {
	switch t.kind {
	case tokEOF:
		return "end of input"
	case tokString:
		return "string " + quote(t.text)
	case tokLParen, tokRParen:
		return quote(t.text)
	default:
		if t.isCmp() {
			return "operator " + t.cmpText()
		}
		return quote(t.text)
	}
}

func quote(s string) string { return "\"" + s + "\"" }
