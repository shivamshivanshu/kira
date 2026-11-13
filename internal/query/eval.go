// Package query implements the kira list-query expression grammar.
package query

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/ptr"
	"github.com/shivamshivanshu/kira/internal/timex"
)

type Predicate func(it *datamodel.Item, cfg *datamodel.Config) bool

const MeToken = "@me"

type Options struct {
	Resolver *id.Resolver
	// Me resolves the caller's identity for @me. It is called only when a
	// query actually names @me, so the git lookup behind it costs nothing on
	// the queries — the overwhelming majority — that never mention it.
	Me           func() string
	ActiveSprint string
	Items        []*datamodel.Item
}

type Compiled struct {
	Pred  Predicate
	Order *Order
	Notes []datamodel.WarnCode
}

type Error struct {
	Pos          int
	Msg          string
	unknownField bool
}

func (e *Error) Error() string { return fmt.Sprintf("query: %s at position %d", e.Msg, e.Pos) }

// IsUnknownField reports whether err is a query error about an unrecognised
// field name — the one class of compile error config load tolerates in a
// saved filter.
func IsUnknownField(err error) bool {
	e, ok := errors.AsType[*Error](err)
	return ok && e.unknownField
}

func Compile(input string, opts Options) (*Compiled, error) {
	q, err := Parse(input)
	if err != nil {
		return nil, err
	}
	c := &compiler{opts: opts}
	pred, err := c.compile(q.Expr)
	if err != nil {
		return nil, err
	}
	if err := checkOrderField(q.Order); err != nil {
		return nil, err
	}
	return &Compiled{Pred: pred, Order: q.Order, Notes: c.notes}, nil
}

func checkOrderField(o *Order) error {
	if o == nil || fields[o.Field] {
		return nil
	}
	return unknownFieldErr(o.pos, o.Field)
}

func Match(field, value string, opts Options) (Predicate, []datamodel.WarnCode, error) {
	field = canonicalField(field)
	if isDateField(field) {
		return nil, nil, &Error{Pos: 0, Msg: "field " + field + " is a date; use the query grammar instead"}
	}
	if isBoolField(field) {
		return nil, nil, &Error{Pos: 0, Msg: "field " + field + " is boolean; use the query grammar instead"}
	}
	pred := &predExpr{field: field, op: token{kind: tokEq, text: "="}, value: value}
	if fields[field] {
		if err := typeCheckValue(field, token{kind: tokWord, text: value}, &pred.date, &pred.num); err != nil {
			return nil, nil, err
		}
	}
	c := &compiler{opts: opts}
	p, err := c.compilePred(pred)
	if err != nil {
		return nil, nil, err
	}
	return p, c.notes, nil
}

type compiler struct {
	opts  Options
	notes []datamodel.WarnCode
	byID  map[string]*datamodel.Item
}

func (c *compiler) compile(e Expr) (Predicate, error) {
	switch n := e.(type) {
	case *orExpr:
		return c.compileBinary(n.left, n.right, false)
	case *andExpr:
		return c.compileBinary(n.left, n.right, true)
	case *notExpr:
		x, err := c.compile(n.x)
		if err != nil {
			return nil, err
		}
		return func(it *datamodel.Item, cfg *datamodel.Config) bool { return !x(it, cfg) }, nil
	case *termExpr:
		return titleSubstringPred(n.text), nil
	case *predExpr:
		return c.compilePred(n)
	case *inExpr:
		return c.compileIn(n)
	case *emptyExpr:
		return c.compileIsEmpty(n)
	case *boolExpr:
		return c.compileBool(n)
	default:
		return nil, &Error{Pos: 0, Msg: "internal: unknown node"}
	}
}

func titleSubstringPred(text string) Predicate {
	needle := strings.ToLower(text)
	return func(it *datamodel.Item, _ *datamodel.Config) bool {
		return strings.Contains(strings.ToLower(it.Title), needle)
	}
}

func (c *compiler) compileBinary(left, right Expr, and bool) (Predicate, error) {
	l, err := c.compile(left)
	if err != nil {
		return nil, err
	}
	r, err := c.compile(right)
	if err != nil {
		return nil, err
	}
	if and {
		return func(it *datamodel.Item, cfg *datamodel.Config) bool { return l(it, cfg) && r(it, cfg) }, nil
	}
	return func(it *datamodel.Item, cfg *datamodel.Config) bool { return l(it, cfg) || r(it, cfg) }, nil
}

func (c *compiler) compilePred(n *predExpr) (Predicate, error) {
	if n.field == fieldPriority && n.op.isOrderedCmp() {
		return c.rankedPriorityPred(n)
	}
	eq := n.op.kind == tokEq
	want := n.value
	switch n.field {
	case fieldState, fieldType, fieldPriority, fieldOwner, fieldReporter,
		fieldSubtype, fieldResolution, fieldRank, fieldCategory:
		if (n.field == fieldOwner || n.field == fieldReporter) && want == MeToken {
			me, err := c.resolveMe(n.valuePos)
			if err != nil {
				return nil, err
			}
			want = me
		}
		return scalarPred(eq, want, accessors[n.field]), nil
	case fieldSprint:
		if n.value == "active" {
			if c.opts.ActiveSprint == "" {
				c.note(datamodel.WarnNoActiveSprint)
				return func(*datamodel.Item, *datamodel.Config) bool { return !eq }, nil
			}
			want = c.opts.ActiveSprint
		}
		return scalarPred(eq, want, accessors[fieldSprint]), nil
	case fieldBoard:
		return scalarPred(eq, strings.ToUpper(want), accessors[fieldBoard]), nil
	case fieldLabel:
		return func(it *datamodel.Item, _ *datamodel.Config) bool {
			return slices.Contains(it.Labels, want) == eq
		}, nil
	case fieldParent:
		return c.compileRefPred(n, eq, func(it *datamodel.Item, u string) bool { return it.Parent != nil && *it.Parent == u })
	case fieldBlockedBy:
		return c.compileRefPred(n, eq, func(it *datamodel.Item, u string) bool { return slices.Contains(it.BlockedBy, u) })
	case fieldLinks:
		return c.compileRefPred(n, eq, func(it *datamodel.Item, u string) bool { return anyLinkTargets(it.Links, u) })
	case fieldEstimate:
		return estimatePred(n.op.kind, n.num), nil
	case fieldCreated:
		return datePred(n, recordedDay(func(it *datamodel.Item) string { return it.Created })), nil
	case fieldUpdated:
		return datePred(n, recordedDay(func(it *datamodel.Item) string { return it.Updated })), nil
	case fieldActivity:
		return datePred(n, recordedDay(func(it *datamodel.Item) string { return it.Activity })), nil
	case fieldDue:
		return datePred(n, func(it *datamodel.Item) string { return ptr.Deref(it.Due) }), nil
	}
	return nil, unknownFieldErr(n.op.pos, n.field)
}

func unknownFieldErr(pos int, field string) *Error {
	return &Error{
		Pos:          pos,
		Msg:          fmt.Sprintf("unknown field %s (available: %s)", quote(field), strings.Join(fieldNames(), ", ")),
		unknownField: true,
	}
}

func fieldNames() []string {
	return slices.Sorted(maps.Keys(fields))
}

func (c *compiler) compileRefPred(n *predExpr, eq bool, matches func(*datamodel.Item, string) bool) (Predicate, error) {
	u, err := c.resolve(n.value, n.valuePos)
	if err != nil {
		return nil, err
	}
	return func(it *datamodel.Item, _ *datamodel.Config) bool { return matches(it, u) == eq }, nil
}

func (c *compiler) resolveMe(pos int) (string, error) {
	me := ""
	if c.opts.Me != nil {
		me = c.opts.Me()
	}
	if me == "" {
		return "", &Error{Pos: pos, Msg: "cannot resolve @me: set git user.name or user.email"}
	}
	return me, nil
}

func (c *compiler) compileIn(n *inExpr) (Predicate, error) {
	// Large scalar sets need one field extraction per item, including extras
	// whose quoted values otherwise get YAML-decoded once per alternative.
	if get, ok := c.scalarInGet(n.field); len(n.values) >= 8 && ok {
		values := make(map[string]bool, len(n.values))
		for i, value := range n.values {
			if (n.field == fieldOwner || n.field == fieldReporter) && value == MeToken {
				me, err := c.resolveMe(n.valuePos[i])
				if err != nil {
					return nil, err
				}
				value = me
			}
			if n.field == fieldBoard {
				value = strings.ToUpper(value)
			}
			values[value] = true
		}
		return func(it *datamodel.Item, cfg *datamodel.Config) bool { return values[get(it, cfg)] }, nil
	}
	orPreds := make([]Predicate, len(n.values))
	for i, v := range n.values {
		pred := &predExpr{
			field: n.field, op: token{kind: tokEq, text: "=", pos: n.fieldPos},
			value: v, valuePos: n.valuePos[i],
		}
		if fields[n.field] {
			if err := typeCheckValue(n.field, token{text: v, pos: n.valuePos[i]}, &pred.date, &pred.num); err != nil {
				return nil, err
			}
		}
		p, err := c.compilePred(pred)
		if err != nil {
			return nil, err
		}
		orPreds[i] = p
	}
	return func(it *datamodel.Item, cfg *datamodel.Config) bool {
		for _, p := range orPreds {
			if p(it, cfg) {
				return true
			}
		}
		return false
	}, nil
}

func (c *compiler) scalarInGet(field string) (func(*datamodel.Item, *datamodel.Config) string, bool) {
	switch field {
	case fieldState, fieldType, fieldPriority, fieldOwner, fieldReporter,
		fieldSubtype, fieldResolution, fieldRank, fieldCategory, fieldBoard:
		return accessors[field], true
	}
	return nil, false
}

func (c *compiler) compileIsEmpty(n *emptyExpr) (Predicate, error) {
	if !fields[n.field] {
		return nil, unknownFieldErr(0, n.field)
	}
	var isEmpty func(it *datamodel.Item, cfg *datamodel.Config) bool
	switch n.field {
	case fieldLabel:
		isEmpty = func(it *datamodel.Item, _ *datamodel.Config) bool { return len(it.Labels) == 0 }
	case fieldBlockedBy:
		isEmpty = func(it *datamodel.Item, _ *datamodel.Config) bool { return len(it.BlockedBy) == 0 }
	case fieldLinks:
		isEmpty = func(it *datamodel.Item, _ *datamodel.Config) bool { return !anyLinkPresent(it.Links) }
	case fieldEstimate:
		isEmpty = func(it *datamodel.Item, _ *datamodel.Config) bool { return it.Estimate == nil }
	default:
		get := scalarGet(n.field)
		isEmpty = func(it *datamodel.Item, cfg *datamodel.Config) bool { return get(it, cfg) == "" }
	}
	want := !n.notEmpty
	return func(it *datamodel.Item, cfg *datamodel.Config) bool { return isEmpty(it, cfg) == want }, nil
}

func (c *compiler) compileBool(n *boolExpr) (Predicate, error) {
	pred := c.blockedPred()
	want := n.want
	return func(it *datamodel.Item, cfg *datamodel.Config) bool { return pred(it, cfg) == want }, nil
}

func (c *compiler) blockedPred() Predicate {
	if c.byID == nil {
		c.byID = datamodel.IndexByID(c.opts.Items)
	}
	byID := c.byID
	return func(it *datamodel.Item, _ *datamodel.Config) bool {
		for _, id := range it.BlockedBy {
			if blocker := byID[id]; blocker != nil {
				if cat, known := datamodel.CategoryOf(blocker.Type, blocker.State); known && cat != datamodel.CategoryDone {
					return true
				}
			}
		}
		return false
	}
}

// SkippedBlocker is a blocked_by reference whose open/closed status could not
// be determined — either it resolves to no known item, or the item's
// type/state has no configured category. Both are treated as satisfied.
type SkippedBlocker struct {
	ID      string
	Blocker *datamodel.Item // nil if the ID resolves to no known item
	Reason  string
}

// OpenBlockers classifies it.BlockedBy against byID (a lookup of ULID to
// item): open holds blockers whose category is known and not Done; skipped
// holds blockers that could not be evaluated (and so are treated as
// satisfied, same as a closed blocker).
func OpenBlockers(it *datamodel.Item, byID map[string]*datamodel.Item) (open []*datamodel.Item, skipped []SkippedBlocker) {
	for _, b := range it.BlockedBy {
		blocker, ok := byID[b]
		if !ok {
			skipped = append(skipped, SkippedBlocker{ID: b, Reason: "resolves to no item"})
			continue
		}
		cat, known := datamodel.CategoryOf(blocker.Type, blocker.State)
		if !known {
			skipped = append(skipped, SkippedBlocker{ID: b, Blocker: blocker, Reason: "has no known state category"})
			continue
		}
		if cat != datamodel.CategoryDone {
			open = append(open, blocker)
		}
	}
	return open, skipped
}

var accessors = map[string]func(*datamodel.Item, *datamodel.Config) string{
	fieldState: func(it *datamodel.Item, _ *datamodel.Config) string { return it.State },
	fieldType:  func(it *datamodel.Item, _ *datamodel.Config) string { return it.Type },
	fieldCategory: func(it *datamodel.Item, _ *datamodel.Config) string {
		cat, _ := datamodel.CategoryOf(it.Type, it.State)
		return string(cat)
	},
	fieldOwner:      func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Owner) },
	fieldReporter:   func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Reporter) },
	fieldSubtype:    func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Subtype) },
	fieldResolution: func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Resolution) },
	fieldPriority:   func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Priority) },
	fieldRank:       func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Rank) },
	fieldSprint:     func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Sprint) },
	fieldDue:        func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Due) },
	fieldParent:     func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Parent) },
	fieldCreated:    func(it *datamodel.Item, _ *datamodel.Config) string { return it.Created },
	fieldUpdated:    func(it *datamodel.Item, _ *datamodel.Config) string { return it.Updated },
	fieldActivity:   func(it *datamodel.Item, _ *datamodel.Config) string { return it.Activity },
	fieldBoard:      func(it *datamodel.Item, _ *datamodel.Config) string { return strings.ToUpper(id.KeyOf(it.Number)) },
}

func scalarGet(field string) func(*datamodel.Item, *datamodel.Config) string {
	if get, ok := accessors[field]; ok {
		return get
	}
	return func(*datamodel.Item, *datamodel.Config) string { return "" }
}

var PriorityRank = func() map[string]int {
	rank := make(map[string]int, len(datamodel.Priorities))
	for i, p := range datamodel.Priorities {
		rank[p] = i
	}
	return rank
}()

func (c *compiler) rankedPriorityPred(n *predExpr) (Predicate, error) {
	want, known := PriorityRank[n.value]
	if !known {
		return nil, &Error{Pos: n.valuePos, Msg: "unknown priority " + quote(n.value)}
	}
	op := n.op.kind
	return func(it *datamodel.Item, _ *datamodel.Config) bool {
		got, ok := PriorityRank[ptr.Deref(it.Priority)]
		return ok && applyCmp(op, got, want)
	}, nil
}

func (t token) isOrderedCmp() bool {
	switch t.kind {
	case tokLt, tokLe, tokGt, tokGe:
		return true
	}
	return false
}

func applyCmp[T cmp.Ordered](op tokKind, a, b T) bool {
	switch op {
	case tokEq:
		return a == b
	case tokNe:
		return a != b
	case tokLt:
		return a < b
	case tokLe:
		return a <= b
	case tokGt:
		return a > b
	case tokGe:
		return a >= b
	default:
		return false
	}
}

func (c *compiler) resolve(ref string, pos int) (string, error) {
	u, err := c.opts.Resolver.Resolve(ref)
	if err != nil {
		return "", &Error{Pos: pos, Msg: err.Error()}
	}
	return u, nil
}

func (c *compiler) note(msg datamodel.WarnCode) {
	if !slices.Contains(c.notes, msg) {
		c.notes = append(c.notes, msg)
	}
}

func anyLinkTargets(links map[string][]string, ulid string) bool {
	for _, targets := range links {
		if slices.Contains(targets, ulid) {
			return true
		}
	}
	return false
}

func anyLinkPresent(links map[string][]string) bool {
	for _, targets := range links {
		if len(targets) > 0 {
			return true
		}
	}
	return false
}

func scalarPred(eq bool, want string, get func(*datamodel.Item, *datamodel.Config) string) Predicate {
	return func(it *datamodel.Item, cfg *datamodel.Config) bool { return (get(it, cfg) == want) == eq }
}

func estimatePred(op tokKind, want float64) Predicate {
	return func(it *datamodel.Item, _ *datamodel.Config) bool {
		if it.Estimate == nil {
			return op == tokNe
		}
		return applyCmp(op, *it.Estimate, want)
	}
}

func datePred(n *predExpr, get func(*datamodel.Item) string) Predicate {
	op := n.op.kind
	want := n.date.Format(time.DateOnly)
	return func(it *datamodel.Item, _ *datamodel.Config) bool {
		day := get(it)
		if day == "" {
			return op == tokNe
		}
		return applyCmp(op, day, want)
	}
}

func recordedDay(get func(*datamodel.Item) string) func(*datamodel.Item) string {
	return func(it *datamodel.Item) string {
		day, _ := timex.Day(get(it))
		return day
	}
}
