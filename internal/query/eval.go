// Package query implements the kira list-query expression grammar.
package query

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

type Predicate func(it *datamodel.Item, cfg *datamodel.Config) bool

const MeToken = "@me"

type Options struct {
	Resolver     *id.Resolver
	Priorities   []string
	ActiveSprint string
	Me           string
	ResolveMe    func() string
	Items        []*datamodel.Item
	Extras       []ExtraField
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
// field name — the one class of parse error whose validity depends on which
// schema-declared extras the caller could supply.
func IsUnknownField(err error) bool {
	e, ok := errors.AsType[*Error](err)
	return ok && e.unknownField
}

func Compile(input string, opts Options) (*Compiled, error) {
	q, err := Parse(input, opts.Extras)
	if err != nil {
		return nil, err
	}
	c := &compiler{opts: opts, spec: specOf(opts.Extras)}
	pred, err := c.compile(q.Expr)
	if err != nil {
		return nil, err
	}
	if q.Order != nil && q.Order.Field == fieldPriority {
		if len(opts.Priorities) == 0 {
			return nil, &Error{Pos: q.Order.pos, Msg: "ORDER BY priority requires configured priorities"}
		}
		q.Order.priorityIndex = PriorityIndex(opts.Priorities)
	}
	return &Compiled{Pred: pred, Order: q.Order, Notes: c.notes}, nil
}

func Match(field, value string, opts Options) (Predicate, []datamodel.WarnCode, error) {
	spec := specOf(opts.Extras)
	if !spec.known(field) {
		return nil, nil, unknownFieldErr(0, field, spec)
	}
	if isDateField(field) {
		return nil, nil, &Error{Pos: 0, Msg: "field " + field + " is not an equality filter"}
	}
	if isBoolField(field) {
		return nil, nil, &Error{Pos: 0, Msg: "field " + field + " is boolean; use the query grammar instead"}
	}
	pred := &predExpr{field: field, op: token{kind: tokEq, text: "="}, value: value}
	if err := typeCheckValue(spec, field, token{kind: tokWord, text: value}, &pred.date, &pred.num); err != nil {
		return nil, nil, err
	}
	c := &compiler{opts: opts, spec: spec}
	p, err := c.compilePred(pred)
	if err != nil {
		return nil, nil, err
	}
	return p, c.notes, nil
}

type compiler struct {
	opts  Options
	spec  extraSpec
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
		return c.compileIsEmpty(n), nil
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
			if c.opts.Me == "" && c.opts.ResolveMe != nil {
				c.opts.Me = c.opts.ResolveMe()
			}
			if c.opts.Me == "" {
				return nil, &Error{Pos: n.valuePos, Msg: "cannot resolve @me: set git user.name or user.email"}
			}
			want = c.opts.Me
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
	case fieldEpic:
		return c.compileRefPred(n, eq, func(it *datamodel.Item, u string) bool { return it.Epic != nil && *it.Epic == u })
	case fieldBlockedBy:
		return c.compileRefPred(n, eq, func(it *datamodel.Item, u string) bool { return slices.Contains(it.BlockedBy, u) })
	case fieldLinks:
		return c.compileRefPred(n, eq, func(it *datamodel.Item, u string) bool { return anyLinkTargets(it.Links, u) })
	case fieldEstimate:
		return estimatePred(n.op.kind, n.num), nil
	case fieldCreated:
		return datePred(n, localDay(func(it *datamodel.Item) string { return it.Created })), nil
	case fieldUpdated:
		return datePred(n, localDay(func(it *datamodel.Item) string { return it.Updated })), nil
	case fieldActivity:
		return datePred(n, localDay(func(it *datamodel.Item) string { return it.Activity })), nil
	case fieldDue:
		return datePred(n, func(it *datamodel.Item) string { return ptr.Deref(it.Due) }), nil
	}
	if ef, ok := c.spec[n.field]; ok {
		return extraPred(n, ef), nil
	}
	return nil, unknownFieldErr(n.op.pos, n.field, c.spec)
}

func extraPred(n *predExpr, ef ExtraField) Predicate {
	eq := n.op.kind == tokEq
	switch {
	case ef.List:
		field, want := n.field, n.value
		return func(it *datamodel.Item, _ *datamodel.Config) bool {
			return slices.ContainsFunc(it.Extras[field].List, func(e string) bool {
				return codec.DecodeScalar(e) == want
			}) == eq
		}
	case ef.Ordered:
		get, op, want := scalarGet(n.field), n.op.kind, n.num
		return func(it *datamodel.Item, cfg *datamodel.Config) bool {
			got, err := strconv.ParseFloat(get(it, cfg), 64)
			return err == nil && applyCmp(op, got, want)
		}
	}
	return scalarPred(eq, n.value, scalarGet(n.field))
}

func unknownFieldErr(pos int, field string, spec extraSpec) *Error {
	return &Error{
		Pos:          pos,
		Msg:          fmt.Sprintf("unknown field %s (available: %s)", quote(field), strings.Join(spec.names(), ", ")),
		unknownField: true,
	}
}

func (c *compiler) compileRefPred(n *predExpr, eq bool, matches func(*datamodel.Item, string) bool) (Predicate, error) {
	u, err := c.resolve(n.value, n.valuePos)
	if err != nil {
		return nil, err
	}
	return func(it *datamodel.Item, _ *datamodel.Config) bool { return matches(it, u) == eq }, nil
}

func (c *compiler) compileIn(n *inExpr) (Predicate, error) {
	// Large scalar sets need one field extraction per item, including extras
	// whose quoted values otherwise get YAML-decoded once per alternative.
	if len(n.values) >= 8 && c.scalarInField(n.field) {
		values := make(map[string]bool, len(n.values))
		for i, value := range n.values {
			if (n.field == fieldOwner || n.field == fieldReporter) && value == MeToken {
				if c.opts.Me == "" && c.opts.ResolveMe != nil {
					c.opts.Me = c.opts.ResolveMe()
				}
				if c.opts.Me == "" {
					return nil, &Error{Pos: n.valuePos[i], Msg: "cannot resolve @me: set git user.name or user.email"}
				}
				value = c.opts.Me
			}
			if n.field == fieldBoard {
				value = strings.ToUpper(value)
			}
			values[value] = true
		}
		get := scalarGet(n.field)
		return func(it *datamodel.Item, cfg *datamodel.Config) bool { return values[get(it, cfg)] }, nil
	}
	orPreds := make([]Predicate, len(n.values))
	for i, v := range n.values {
		pred := &predExpr{
			field: n.field, op: token{kind: tokEq, text: "=", pos: n.fieldPos},
			value: v, valuePos: n.valuePos[i],
		}
		if err := typeCheckValue(c.spec, n.field, token{text: v, pos: n.valuePos[i]}, &pred.date, &pred.num); err != nil {
			return nil, err
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

func (c *compiler) scalarInField(field string) bool {
	switch field {
	case fieldState, fieldType, fieldPriority, fieldOwner, fieldReporter,
		fieldSubtype, fieldResolution, fieldRank, fieldCategory, fieldBoard:
		return true
	}
	f, ok := c.spec[field]
	return ok && !f.List && !f.Ordered
}

func (c *compiler) compileIsEmpty(n *emptyExpr) Predicate {
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
		field := n.field
		if c.spec[field].List {
			isEmpty = func(it *datamodel.Item, _ *datamodel.Config) bool { return len(it.Extras[field].List) == 0 }
			break
		}
		get := scalarGet(field)
		isEmpty = func(it *datamodel.Item, cfg *datamodel.Config) bool { return get(it, cfg) == "" }
	}
	want := !n.notEmpty
	return func(it *datamodel.Item, cfg *datamodel.Config) bool { return isEmpty(it, cfg) == want }
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
	return func(it *datamodel.Item, cfg *datamodel.Config) bool {
		for _, id := range it.BlockedBy {
			if blocker := byID[id]; blocker != nil {
				if cat, known := cfg.CategoryOf(blocker.Type, blocker.State); known && cat != datamodel.CategoryDone {
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
func OpenBlockers(cfg *datamodel.Config, it *datamodel.Item, byID map[string]*datamodel.Item) (open []*datamodel.Item, skipped []SkippedBlocker) {
	for _, b := range it.BlockedBy {
		blocker, ok := byID[b]
		if !ok {
			skipped = append(skipped, SkippedBlocker{ID: b, Reason: "resolves to no item"})
			continue
		}
		cat, known := cfg.CategoryOf(blocker.Type, blocker.State)
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
	fieldCategory: func(it *datamodel.Item, cfg *datamodel.Config) string {
		cat, _ := cfg.CategoryOf(it.Type, it.State)
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
	fieldEpic:       func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Epic) },
	fieldCreated:    func(it *datamodel.Item, _ *datamodel.Config) string { return it.Created },
	fieldUpdated:    func(it *datamodel.Item, _ *datamodel.Config) string { return it.Updated },
	fieldActivity:   func(it *datamodel.Item, _ *datamodel.Config) string { return it.Activity },
	fieldBoard:      func(it *datamodel.Item, _ *datamodel.Config) string { return strings.ToUpper(id.KeyOf(it.Number)) },
}

func scalarGet(field string) func(*datamodel.Item, *datamodel.Config) string {
	if get, ok := accessors[field]; ok {
		return get
	}
	return func(it *datamodel.Item, _ *datamodel.Config) string {
		if v := it.Extras[field].Scalar; v != nil {
			return codec.DecodeScalar(*v)
		}
		return ""
	}
}

func PriorityIndex(priorities []string) map[string]int {
	index := make(map[string]int, len(priorities))
	for i, p := range priorities {
		index[p] = i
	}
	return index
}

func (c *compiler) rankedPriorityPred(n *predExpr) (Predicate, error) {
	if len(c.opts.Priorities) == 0 {
		return nil, &Error{Pos: n.op.pos,
			Msg: "ordered comparison on priority requires configured priorities (only = and != apply)"}
	}
	index := PriorityIndex(c.opts.Priorities)
	want, known := index[n.value]
	if !known {
		return nil, &Error{Pos: n.valuePos, Msg: "unknown priority " + quote(n.value)}
	}
	op := n.op.kind
	return func(it *datamodel.Item, _ *datamodel.Config) bool {
		got, ok := index[ptr.Deref(it.Priority)]
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
		return it.Estimate != nil && applyCmp(op, *it.Estimate, want)
	}
}

func datePred(n *predExpr, get func(*datamodel.Item) string) Predicate {
	op := n.op.kind
	want := n.date.Format(time.DateOnly)
	return func(it *datamodel.Item, _ *datamodel.Config) bool {
		day := get(it)
		if day == "" {
			return false
		}
		return applyCmp(op, day, want)
	}
}

func localDay(get func(*datamodel.Item) string) func(*datamodel.Item) string {
	return func(it *datamodel.Item) string {
		t, err := time.Parse(time.RFC3339, get(it))
		if err != nil {
			return ""
		}
		return t.Local().Format(time.DateOnly)
	}
}
