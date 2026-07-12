// Package query implements the kira list-query language: the expression grammar
// in docs/design/04-cli.md §4. It lexes and parses an expression into an AST
// (Parse) and compiles that AST into a Predicate over items (Compile). It is a
// leaf package over internal/{item,config,id}; core drives it and never the
// other way round, so the same engine backs `kira query`, `kira list --query`,
// and any future frontend.
package query

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// Predicate reports whether an item matches a compiled query. cfg supplies the
// state->category mapping a category comparison needs.
type Predicate func(it *item.Item, cfg *config.Config) bool

// Error is a query lex/parse/compile failure. Pos is the byte offset in the
// source expression where the problem was detected, for a caret-style message.
type Error struct {
	Pos int
	Msg string
}

func (e *Error) Error() string { return fmt.Sprintf("query: %s at position %d", e.Msg, e.Pos) }

// Compile parses input and binds it to a Predicate. resolver (required)
// resolves an `epic=<ref>` value (ULID | prefix | number | alias) to its ULID;
// a reference that does not resolve is an *Error at the value's position.
func Compile(input string, resolver *id.Resolver) (Predicate, error) {
	e, err := Parse(input)
	if err != nil {
		return nil, err
	}
	return compile(e, resolver)
}

// Match builds the predicate for a single `field = value` comparison — the
// programmatic equivalent of one grammar predicate. It is how the flat CLI
// filters (--state, --owner, --epic, …) are lowered into this one comparison
// engine instead of a parallel hand-written one. Date fields are rejected.
func Match(field, value string, resolver *id.Resolver) (Predicate, error) {
	if !fields[field] {
		return nil, &Error{Pos: 0, Msg: "unknown field " + field}
	}
	if isDateField(field) {
		return nil, &Error{Pos: 0, Msg: "field " + field + " is not an equality filter"}
	}
	n := &predExpr{field: field, op: token{kind: tokEq, text: "="}, value: value}
	return compilePred(n, resolver)
}

func compile(e Expr, resolver *id.Resolver) (Predicate, error) {
	switch n := e.(type) {
	case *orExpr:
		return compileBinary(n.left, n.right, resolver, func(a, b bool) bool { return a || b })
	case *andExpr:
		return compileBinary(n.left, n.right, resolver, func(a, b bool) bool { return a && b })
	case *notExpr:
		x, err := compile(n.x, resolver)
		if err != nil {
			return nil, err
		}
		return func(it *item.Item, cfg *config.Config) bool { return !x(it, cfg) }, nil
	case *termExpr:
		needle := strings.ToLower(n.text)
		return func(it *item.Item, _ *config.Config) bool {
			return strings.Contains(strings.ToLower(it.Title), needle)
		}, nil
	case *predExpr:
		return compilePred(n, resolver)
	default:
		return nil, &Error{Pos: 0, Msg: "internal: unknown node"}
	}
}

// compileBinary compiles both operands of an or/and node and combines their
// results with join.
func compileBinary(left, right Expr, resolver *id.Resolver, join func(a, b bool) bool) (Predicate, error) {
	l, err := compile(left, resolver)
	if err != nil {
		return nil, err
	}
	r, err := compile(right, resolver)
	if err != nil {
		return nil, err
	}
	return func(it *item.Item, cfg *config.Config) bool { return join(l(it, cfg), r(it, cfg)) }, nil
}

func compilePred(n *predExpr, resolver *id.Resolver) (Predicate, error) {
	eq := n.op.kind == tokEq
	want := n.value
	switch n.field {
	case fieldState:
		return scalarPred(eq, want, func(it *item.Item) string { return it.State }), nil
	case fieldType:
		return scalarPred(eq, want, func(it *item.Item) string { return it.Type }), nil
	case fieldPriority:
		return scalarPred(eq, want, func(it *item.Item) string { return deref(it.Priority) }), nil
	case fieldOwner:
		return scalarPred(eq, want, func(it *item.Item) string { return deref(it.Owner) }), nil
	case fieldCategory:
		return func(it *item.Item, cfg *config.Config) bool {
			return (categoryOf(cfg, it.Type, it.State) == want) == eq
		}, nil
	case fieldLabel:
		return func(it *item.Item, _ *config.Config) bool {
			return slices.Contains(it.Labels, want) == eq
		}, nil
	case fieldEpic:
		u, err := resolver.Resolve(n.value)
		if err != nil {
			return nil, &Error{Pos: n.valuePos, Msg: err.Error()}
		}
		want = u
		return func(it *item.Item, _ *config.Config) bool {
			return (it.Epic != nil && *it.Epic == want) == eq
		}, nil
	case fieldCreated:
		return datePred(n, func(it *item.Item) string { return it.Created }), nil
	case fieldUpdated:
		return datePred(n, func(it *item.Item) string { return it.Updated }), nil
	default:
		return nil, &Error{Pos: n.op.pos, Msg: "unknown field " + n.field}
	}
}

// scalarPred builds an =/!= predicate comparing want against a string
// projection of the item.
func scalarPred(eq bool, want string, get func(*item.Item) string) Predicate {
	return func(it *item.Item, _ *config.Config) bool { return (get(it) == want) == eq }
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// categoryOf returns the configured category string for a state within a type's
// workflow, or "" if unknown. It reads config's exported workflow fields so the
// query package needs no dependency on core.
func categoryOf(cfg *config.Config, typ, state string) string {
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

// datePred builds a date comparison predicate. An item whose timestamp does not
// parse never matches (a malformed stored date is caught by validation, not by
// silently satisfying a filter).
func datePred(n *predExpr, get func(*item.Item) string) Predicate {
	op := n.op.kind
	want := n.date
	return func(it *item.Item, _ *config.Config) bool {
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
