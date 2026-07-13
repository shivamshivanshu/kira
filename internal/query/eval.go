// Package query implements the kira list-query expression grammar (docs/design/04-cli.md §4).
package query

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

type Predicate func(it *datamodel.Item, cfg *datamodel.Config) bool

type Options struct {
	Resolver     *id.Resolver
	Priorities   []string
	ActiveSprint string
}

type Compiled struct {
	Pred  Predicate
	Order *Order
	Notes []string
}

type Error struct {
	Pos int
	Msg string
}

func (e *Error) Error() string { return fmt.Sprintf("query: %s at position %d", e.Msg, e.Pos) }

const NoActiveSprintNote = "no active sprint set; sprint=active matches nothing (run `kira sprint activate <key>`)"

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
	if q.Order != nil && q.Order.Field == fieldPriority && len(opts.Priorities) == 0 {
		return nil, &Error{Pos: q.Order.pos, Msg: "ORDER BY priority requires configured priorities"}
	}
	return &Compiled{Pred: pred, Order: q.Order, Notes: c.notes}, nil
}

func Match(field, value string, opts Options) (Predicate, error) {
	if !fields[field] {
		return nil, &Error{Pos: 0, Msg: "unknown field " + field}
	}
	if isDateField(field) {
		return nil, &Error{Pos: 0, Msg: "field " + field + " is not an equality filter"}
	}
	c := &compiler{opts: opts}
	return c.compilePred(&predExpr{field: field, op: token{kind: tokEq, text: "="}, value: value})
}

type compiler struct {
	opts  Options
	notes []string
}

func (c *compiler) compile(e Expr) (Predicate, error) {
	switch n := e.(type) {
	case *orExpr:
		return c.compileBinary(n.left, n.right, func(a, b bool) bool { return a || b })
	case *andExpr:
		return c.compileBinary(n.left, n.right, func(a, b bool) bool { return a && b })
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
		return compileIsEmpty(n), nil
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

func (c *compiler) compileBinary(left, right Expr, join func(a, b bool) bool) (Predicate, error) {
	l, err := c.compile(left)
	if err != nil {
		return nil, err
	}
	r, err := c.compile(right)
	if err != nil {
		return nil, err
	}
	return func(it *datamodel.Item, cfg *datamodel.Config) bool { return join(l(it, cfg), r(it, cfg)) }, nil
}

func (c *compiler) compilePred(n *predExpr) (Predicate, error) {
	if n.field == fieldPriority && n.op.isOrderedCmp() {
		return c.rankedPriorityPred(n)
	}
	eq := n.op.kind == tokEq
	want := n.value
	switch n.field {
	case fieldState:
		return scalarPred(eq, want, func(it *datamodel.Item) string { return it.State }), nil
	case fieldType:
		return scalarPred(eq, want, func(it *datamodel.Item) string { return it.Type }), nil
	case fieldPriority:
		return scalarPred(eq, want, func(it *datamodel.Item) string { return ptr.Deref(it.Priority) }), nil
	case fieldOwner:
		return scalarPred(eq, want, func(it *datamodel.Item) string { return ptr.Deref(it.Owner) }), nil
	case fieldReporter:
		return scalarPred(eq, want, func(it *datamodel.Item) string { return ptr.Deref(it.Reporter) }), nil
	case fieldSubtype:
		return scalarPred(eq, want, func(it *datamodel.Item) string { return ptr.Deref(it.Subtype) }), nil
	case fieldResolution:
		return scalarPred(eq, want, func(it *datamodel.Item) string { return ptr.Deref(it.Resolution) }), nil
	case fieldRank:
		return scalarPred(eq, want, func(it *datamodel.Item) string { return ptr.Deref(it.Rank) }), nil
	case fieldSprint:
		if n.value == "active" {
			if c.opts.ActiveSprint == "" {
				c.note(NoActiveSprintNote)
				return func(*datamodel.Item, *datamodel.Config) bool { return !eq }, nil
			}
			want = c.opts.ActiveSprint
		}
		return scalarPred(eq, want, func(it *datamodel.Item) string { return ptr.Deref(it.Sprint) }), nil
	case fieldCategory:
		return func(it *datamodel.Item, cfg *datamodel.Config) bool {
			return (categoryOf(cfg, it.Type, it.State) == want) == eq
		}, nil
	case fieldLabel:
		return func(it *datamodel.Item, _ *datamodel.Config) bool {
			return slices.Contains(it.Labels, want) == eq
		}, nil
	case fieldEpic:
		u, err := c.resolve(n.value, n.valuePos)
		if err != nil {
			return nil, err
		}
		return func(it *datamodel.Item, _ *datamodel.Config) bool {
			return (it.Epic != nil && *it.Epic == u) == eq
		}, nil
	case fieldBlockedBy:
		u, err := c.resolve(n.value, n.valuePos)
		if err != nil {
			return nil, err
		}
		return func(it *datamodel.Item, _ *datamodel.Config) bool {
			return slices.Contains(it.BlockedBy, u) == eq
		}, nil
	case fieldLinks:
		u, err := c.resolve(n.value, n.valuePos)
		if err != nil {
			return nil, err
		}
		return func(it *datamodel.Item, _ *datamodel.Config) bool {
			return anyLinkTargets(it.Links, u) == eq
		}, nil
	case fieldEstimate:
		return estimatePred(n.op.kind, n.num), nil
	case fieldCreated:
		return datePred(n, func(it *datamodel.Item) string { return it.Created }), nil
	case fieldUpdated:
		return datePred(n, func(it *datamodel.Item) string { return it.Updated }), nil
	case fieldDue:
		return datePred(n, func(it *datamodel.Item) string { return ptr.Deref(it.Due) }), nil
	default:
		return nil, &Error{Pos: n.op.pos, Msg: "unknown field " + n.field}
	}
}

func (c *compiler) compileIn(n *inExpr) (Predicate, error) {
	equalsAnyValue := make([]Predicate, len(n.values))
	for i, v := range n.values {
		p, err := c.compilePred(&predExpr{
			field: n.field, op: token{kind: tokEq, text: "=", pos: n.fieldPos},
			value: v, valuePos: n.valuePos[i],
		})
		if err != nil {
			return nil, err
		}
		equalsAnyValue[i] = p
	}
	return func(it *datamodel.Item, cfg *datamodel.Config) bool {
		for _, p := range equalsAnyValue {
			if p(it, cfg) {
				return true
			}
		}
		return false
	}, nil
}

func compileIsEmpty(n *emptyExpr) Predicate {
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
	return func(it *datamodel.Item, cfg *datamodel.Config) bool { return isEmpty(it, cfg) == want }
}

func scalarGet(field string) func(*datamodel.Item, *datamodel.Config) string {
	switch field {
	case fieldState:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return it.State }
	case fieldType:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return it.Type }
	case fieldCategory:
		return func(it *datamodel.Item, cfg *datamodel.Config) string { return categoryOf(cfg, it.Type, it.State) }
	case fieldOwner:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Owner) }
	case fieldReporter:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Reporter) }
	case fieldSubtype:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Subtype) }
	case fieldResolution:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Resolution) }
	case fieldPriority:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Priority) }
	case fieldRank:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Rank) }
	case fieldSprint:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Sprint) }
	case fieldDue:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Due) }
	case fieldEpic:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return ptr.Deref(it.Epic) }
	case fieldCreated:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return it.Created }
	case fieldUpdated:
		return func(it *datamodel.Item, _ *datamodel.Config) string { return it.Updated }
	default:
		return func(*datamodel.Item, *datamodel.Config) string { return "" }
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
		return ok && cmpInts(op, got, want)
	}, nil
}

func (t token) isOrderedCmp() bool { return t.kind >= tokLt && t.kind <= tokGe }

func cmpInts(op tokKind, a, b int) bool {
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

func (c *compiler) note(msg string) {
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

func scalarPred(eq bool, want string, get func(*datamodel.Item) string) Predicate {
	return func(it *datamodel.Item, _ *datamodel.Config) bool { return (get(it) == want) == eq }
}

func categoryOf(cfg *datamodel.Config, typ, state string) string {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return ""
	}
	for _, st := range wf.States {
		if st.Key == state {
			return string(st.Category)
		}
	}
	return ""
}

func estimatePred(op tokKind, want float64) Predicate {
	return func(it *datamodel.Item, _ *datamodel.Config) bool {
		if it.Estimate == nil {
			return false
		}
		got := *it.Estimate
		switch op {
		case tokEq:
			return got == want
		case tokNe:
			return got != want
		case tokLt:
			return got < want
		case tokLe:
			return got <= want
		case tokGt:
			return got > want
		case tokGe:
			return got >= want
		default:
			return false
		}
	}
}

func datePred(n *predExpr, get func(*datamodel.Item) string) Predicate {
	op := n.op.kind
	want := n.date
	return func(it *datamodel.Item, _ *datamodel.Config) bool {
		t, err := parseDate(get(it))
		if err != nil {
			return false
		}
		switch op {
		case tokEq:
			return t.Equal(want)
		case tokNe:
			return !t.Equal(want)
		case tokLt:
			return t.Before(want)
		case tokLe:
			return t.Before(want) || t.Equal(want)
		case tokGt:
			return t.After(want)
		case tokGe:
			return t.After(want) || t.Equal(want)
		default:
			return false
		}
	}
}
