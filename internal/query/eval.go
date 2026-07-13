// Package query implements the kira list-query expression grammar (docs/design/04-cli.md §4).
package query

import (
	"cmp"
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
	Notes []datamodel.WarnCode
}

type Error struct {
	Pos int
	Msg string
}

func (e *Error) Error() string { return fmt.Sprintf("query: %s at position %d", e.Msg, e.Pos) }

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
		return nil, unknownFieldErr(0, field)
	}
	if isDateField(field) {
		return nil, &Error{Pos: 0, Msg: "field " + field + " is not an equality filter"}
	}
	c := &compiler{opts: opts}
	return c.compilePred(&predExpr{field: field, op: token{kind: tokEq, text: "="}, value: value})
}

type compiler struct {
	opts  Options
	notes []datamodel.WarnCode
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
	case fieldState, fieldType, fieldPriority, fieldOwner, fieldReporter,
		fieldSubtype, fieldResolution, fieldRank, fieldCategory:
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
		return datePred(n, func(it *datamodel.Item) string { return it.Created }), nil
	case fieldUpdated:
		return datePred(n, func(it *datamodel.Item) string { return it.Updated }), nil
	case fieldDue:
		return datePred(n, func(it *datamodel.Item) string { return ptr.Deref(it.Due) }), nil
	}
	return nil, unknownFieldErr(n.op.pos, n.field)
}

func unknownFieldErr(pos int, field string) *Error {
	names := make([]string, 0, len(fields))
	for f := range fields {
		names = append(names, f)
	}
	slices.Sort(names)
	return &Error{Pos: pos, Msg: fmt.Sprintf("unknown field %s (available: %s)", quote(field), strings.Join(names, ", "))}
}

func (c *compiler) compileRefPred(n *predExpr, eq bool, matches func(*datamodel.Item, string) bool) (Predicate, error) {
	u, err := c.resolve(n.value, n.valuePos)
	if err != nil {
		return nil, err
	}
	return func(it *datamodel.Item, _ *datamodel.Config) bool { return matches(it, u) == eq }, nil
}

func (c *compiler) compileIn(n *inExpr) (Predicate, error) {
	orPreds := make([]Predicate, len(n.values))
	for i, v := range n.values {
		p, err := c.compilePred(&predExpr{
			field: n.field, op: token{kind: tokEq, text: "=", pos: n.fieldPos},
			value: v, valuePos: n.valuePos[i],
		})
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

var accessors = map[string]func(*datamodel.Item, *datamodel.Config) string{
	fieldState:      func(it *datamodel.Item, _ *datamodel.Config) string { return it.State },
	fieldType:       func(it *datamodel.Item, _ *datamodel.Config) string { return it.Type },
	fieldCategory:   func(it *datamodel.Item, cfg *datamodel.Config) string { return categoryOf(cfg, it.Type, it.State) },
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
}

func scalarGet(field string) func(*datamodel.Item, *datamodel.Config) string {
	if get, ok := accessors[field]; ok {
		return get
	}
	return func(*datamodel.Item, *datamodel.Config) string { return "" }
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
		return it.Estimate != nil && applyCmp(op, *it.Estimate, want)
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
		return applyCmp(op, t.Compare(want), 0)
	}
}
