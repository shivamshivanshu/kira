package query

import (
	"strconv"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/timex"
)

type Expr interface {
	String() string
}

type orExpr struct{ left, right Expr }
type andExpr struct{ left, right Expr }
type notExpr struct{ x Expr }

type predExpr struct {
	field    string
	op       token
	value    string
	date     time.Time
	num      float64
	valuePos int
}

type inExpr struct {
	field    string
	values   []string
	valuePos []int
	fieldPos int
}

type emptyExpr struct {
	field    string
	notEmpty bool
}

type boolExpr struct {
	field string
	want  bool
}

type termExpr struct{ text string }

func (e *orExpr) String() string   { return "(or " + e.left.String() + " " + e.right.String() + ")" }
func (e *andExpr) String() string  { return "(and " + e.left.String() + " " + e.right.String() + ")" }
func (e *notExpr) String() string  { return "(not " + e.x.String() + ")" }
func (e *predExpr) String() string { return "(" + e.field + " " + e.op.cmpText() + " " + e.value + ")" }
func (e *inExpr) String() string {
	return "(in " + e.field + " " + strings.Join(e.values, " ") + ")"
}
func (e *emptyExpr) String() string {
	if e.notEmpty {
		return "(is-not-empty " + e.field + ")"
	}
	return "(is-empty " + e.field + ")"
}
func (e *boolExpr) String() string {
	if e.want {
		return "(" + e.field + " = true)"
	}
	return "(" + e.field + " = false)"
}
func (e *termExpr) String() string { return "(term " + e.text + ")" }

type Order struct {
	Field         string
	Desc          bool
	pos           int
	priorityIndex map[string]int
}

func (o *Order) String() string {
	dir := "asc"
	if o.Desc {
		dir = "desc"
	}
	return "(order-by " + o.Field + " " + dir + ")"
}

type Parsed struct {
	Expr  Expr
	Order *Order
}

func (q *Parsed) String() string {
	if q.Order == nil {
		return q.Expr.String()
	}
	return q.Expr.String() + " " + q.Order.String()
}

const (
	fieldState      = "state"
	fieldCategory   = "category"
	fieldOwner      = "owner"
	fieldReporter   = "reporter"
	fieldLabel      = "label"
	fieldType       = "type"
	fieldSubtype    = "subtype"
	fieldEpic       = "epic"
	fieldPriority   = "priority"
	fieldRank       = "rank"
	fieldSprint     = "sprint"
	fieldDue        = "due"
	fieldEstimate   = "estimate"
	fieldBlockedBy  = "blocked_by"
	fieldBlocked    = "blocked"
	fieldLinks      = "links"
	fieldResolution = "resolution"
	fieldCreated    = "created"
	fieldUpdated    = "updated"
	fieldActivity   = "activity"
	fieldBoard      = "board"
)

var fields = map[string]bool{
	fieldState: true, fieldCategory: true, fieldOwner: true, fieldReporter: true,
	fieldLabel: true, fieldType: true, fieldSubtype: true, fieldEpic: true,
	fieldPriority: true, fieldRank: true, fieldSprint: true, fieldDue: true,
	fieldEstimate: true, fieldBlockedBy: true, fieldBlocked: true, fieldLinks: true,
	fieldResolution: true, fieldCreated: true, fieldUpdated: true,
	fieldActivity: true, fieldBoard: true,
}

func isDateField(f string) bool {
	return f == fieldCreated || f == fieldUpdated || f == fieldDue || f == fieldActivity
}

func isBoolField(f string) bool { return f == fieldBlocked }

func isListField(f string) bool {
	return f == fieldLabel || f == fieldBlockedBy || f == fieldLinks
}

func allowsOrderedCmp(f string) bool {
	return isDateField(f) || f == fieldEstimate || f == fieldPriority
}

func isAlwaysPresent(f string) bool {
	switch f {
	case fieldState, fieldType, fieldCategory, fieldCreated, fieldUpdated, fieldActivity, fieldBoard:
		return true
	}
	return false
}

func isKeyword(t token, kw string) bool {
	return t.kind == tokWord && strings.EqualFold(t.text, kw)
}

func anyKeyword(t token) bool {
	return isKeyword(t, "AND") || isKeyword(t, "OR") || isKeyword(t, "NOT")
}

func Parse(input string) (*Parsed, error) {
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
	ord, err := p.parseOrder()
	if err != nil {
		return nil, err
	}
	if t := p.peek(); t.kind != tokEOF {
		if ord != nil {
			return nil, &Error{Pos: t.pos, Msg: "ORDER BY must be the trailing clause"}
		}
		return nil, &Error{Pos: t.pos, Msg: "unexpected " + describe(t)}
	}
	return &Parsed{Expr: e, Order: ord}, nil
}

type parser struct {
	toks []token
	i    int
}

func (p *parser) peek() token  { return p.toks[p.i] }
func (p *parser) peek2() token { return p.toks[p.i+1] }
func (p *parser) next() token  { t := p.toks[p.i]; p.i++; return t }

func (p *parser) atOrderBy() bool {
	return isKeyword(p.peek(), "ORDER") && isKeyword(p.peek2(), "BY")
}

func (p *parser) startsFieldPredicate() bool {
	return p.peek2().isCmp() || (isKeyword(p.peek2(), "IN") && p.toks[p.i+2].kind == tokLParen) || isKeyword(p.peek2(), "IS")
}

func (p *parser) parseOrder() (*Order, error) {
	if !p.atOrderBy() {
		return nil, nil
	}
	p.next()
	p.next()
	f := p.peek()
	if f.kind != tokWord || !fields[f.text] {
		return nil, &Error{Pos: f.pos, Msg: "expected a field after ORDER BY"}
	}
	if isListField(f.text) {
		return nil, &Error{Pos: f.pos, Msg: "cannot order by list field " + f.text}
	}
	if isBoolField(f.text) {
		return nil, &Error{Pos: f.pos, Msg: "cannot order by boolean field " + f.text}
	}
	p.next()
	ord := &Order{Field: f.text, pos: f.pos}
	if d := p.peek(); isKeyword(d, "asc") || isKeyword(d, "desc") {
		ord.Desc = strings.EqualFold(d.text, "desc")
		p.next()
	}
	return ord, nil
}

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
			p.next()
		} else if !p.startsOperand(t) {
			return left, nil
		}
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &andExpr{left, right}
	}
}

func (p *parser) startsOperand(t token) bool {
	switch t.kind {
	case tokLParen, tokString:
		return true
	case tokWord:
		return !isKeyword(t, "OR") && !isKeyword(t, "AND") && !p.atOrderBy()
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
		if p.atOrderBy() {
			return nil, &Error{Pos: t.pos, Msg: "expected an expression before ORDER BY"}
		}
		if fields[t.text] {
			switch {
			case p.peek2().isCmp():
				if isBoolField(t.text) {
					return p.parseBool()
				}
				return p.parsePredicate()
			case isKeyword(p.peek2(), "IN") && p.toks[p.i+2].kind == tokLParen:
				if isBoolField(t.text) {
					return nil, &Error{Pos: t.pos, Msg: "field " + t.text + " is boolean; use " + t.text + " or NOT " + t.text}
				}
				return p.parseIn()
			case isKeyword(p.peek2(), "IS"):
				return p.parseIsEmpty()
			}
			if isBoolField(t.text) {
				p.next()
				return &boolExpr{field: t.text, want: true}, nil
			}
		} else if p.startsFieldPredicate() {
			return nil, unknownFieldErr(t.pos, t.text)
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
	if !allowsOrderedCmp(field.text) && op.kind != tokEq && op.kind != tokNe {
		return nil, &Error{Pos: op.pos, Msg: "operator " + op.cmpText() + " is only valid on " +
			"created/updated/due/activity, estimate, and priority"}
	}
	val := p.peek()
	if val.kind != tokWord && val.kind != tokString {
		return nil, &Error{Pos: val.pos, Msg: "expected a value after " + op.cmpText()}
	}
	p.next()
	pred := &predExpr{field: field.text, op: op, value: val.text, valuePos: val.pos}
	if err := typeCheckValue(field.text, val, &pred.date, &pred.num); err != nil {
		return nil, err
	}
	return pred, nil
}

func (p *parser) parseBool() (Expr, error) {
	field := p.next()
	op := p.next()
	if op.kind != tokEq && op.kind != tokNe {
		return nil, &Error{Pos: op.pos, Msg: "operator " + op.cmpText() + " is not valid on boolean field " + field.text}
	}
	val := p.peek()
	want, ok := boolValue(val)
	if !ok {
		return nil, &Error{Pos: val.pos, Msg: "expected true or false after " + op.cmpText()}
	}
	p.next()
	if op.kind == tokNe {
		want = !want
	}
	return &boolExpr{field: field.text, want: want}, nil
}

func boolValue(t token) (bool, bool) {
	if t.kind != tokWord && t.kind != tokString {
		return false, false
	}
	switch strings.ToLower(t.text) {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}

func typeCheckValue(field string, val token, date *time.Time, num *float64) error {
	if isDateField(field) {
		d, err := parseDate(val.text)
		if err != nil {
			return &Error{Pos: val.pos, Msg: "invalid date " + quote(val.text)}
		}
		if field != fieldDue {
			if _, err := time.Parse(time.DateOnly, val.text); err != nil {
				return &Error{Pos: val.pos, Msg: "field " + field + " compares by calendar day; use YYYY-MM-DD, not " + quote(val.text)}
			}
		}
		*date = d
	}
	if field == fieldEstimate {
		n, err := strconv.ParseFloat(val.text, 64)
		if err != nil {
			return &Error{Pos: val.pos, Msg: "invalid number " + quote(val.text)}
		}
		*num = n
	}
	return nil
}

func (p *parser) parseIn() (Expr, error) {
	field := p.next()
	p.next()
	p.next()
	e := &inExpr{field: field.text, fieldPos: field.pos}
	for {
		val := p.peek()
		if val.kind != tokWord && val.kind != tokString {
			return nil, &Error{Pos: val.pos, Msg: "expected a value in IN (…)"}
		}
		p.next()
		var date time.Time
		var num float64
		if err := typeCheckValue(field.text, val, &date, &num); err != nil {
			return nil, err
		}
		e.values = append(e.values, val.text)
		e.valuePos = append(e.valuePos, val.pos)
		switch p.peek().kind {
		case tokComma:
			p.next()
		case tokRParen:
			p.next()
			return e, nil
		default:
			return nil, &Error{Pos: p.peek().pos, Msg: "expected ',' or ')' in IN (…)"}
		}
	}
}

func (p *parser) parseIsEmpty() (Expr, error) {
	field := p.next()
	is := p.next()
	e := &emptyExpr{field: field.text}
	if isKeyword(p.peek(), "NOT") {
		e.notEmpty = true
		p.next()
	}
	if !isKeyword(p.peek(), "EMPTY") {
		return nil, &Error{Pos: p.peek().pos, Msg: "expected EMPTY after IS"}
	}
	p.next()
	if isBoolField(field.text) {
		return nil, &Error{Pos: is.pos, Msg: "field " + field.text + " is boolean; use " + field.text + " or NOT " + field.text}
	}
	if isAlwaysPresent(field.text) {
		return nil, &Error{Pos: is.pos, Msg: "field " + field.text + " is never empty"}
	}
	return e, nil
}

func parseDate(s string) (time.Time, error) {
	return timex.ParseFlexible(s)
}

func describe(t token) string {
	switch t.kind {
	case tokEOF:
		return "end of input"
	case tokString:
		return "string " + quote(t.text)
	case tokLParen, tokRParen, tokComma:
		return quote(t.text)
	default:
		if t.isCmp() {
			return "operator " + t.cmpText()
		}
		return quote(t.text)
	}
}

func quote(s string) string { return "\"" + s + "\"" }
