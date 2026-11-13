package core

import (
	"fmt"
	"slices"
	"sync"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/query"
)

// FieldFilter is one `--field key=value` equality filter. It desugars to the
// same query predicate a native flag or a `key = value` expression produces.
type FieldFilter struct {
	Key   string
	Value string
}

type ListOpts struct {
	Type     string
	Subtype  string
	State    string
	Category string
	Owner    string
	Label    string
	Parent   string
	Priority string
	Sprint   string
	Board    string
	Filter   string
	Query    string
	Fields   []FieldFilter
	Tree     bool
	At       string
}

func (s *Store) List(cfg *datamodel.Config, opts ListOpts) (*datamodel.ListResult, error) {
	ld, err := s.read(cfg, loadOpts{at: opts.At, useIndex: true})
	if err != nil {
		return nil, err
	}
	return listResult(ld, opts, s.queryOptions(ld))
}

func listResult(ld *loaded, opts ListOpts, qopts query.Options) (*datamodel.ListResult, error) {
	cfg, items := ld.cfg, ld.items

	pred, order, notes, err := opts.compile(cfg, qopts)
	if err != nil {
		return nil, queryError(err)
	}

	matched := ld.match(opts, pred, order)

	parentNumbers := parentNumberMap(items)
	rows := make([]datamodel.ListItem, len(matched))
	for i, it := range matched {
		rows[i] = listItemOf(it, parentNumbers)
	}
	res := &datamodel.ListResult{
		Items:       rows,
		Count:       len(rows),
		StderrNotes: slices.Concat(ld.notes, notes),
	}
	if opts.Tree {
		res.Tree = groupByParent(rows, parentNumbers)
	}
	return res, nil
}

func matchSorted(ld *loaded, opts ListOpts, qopts query.Options) ([]*datamodel.Item, error) {
	pred, order, _, err := opts.compile(ld.cfg, qopts)
	if err != nil {
		return nil, queryError(err)
	}
	return ld.match(opts, pred, order), nil
}

func (ld *loaded) match(opts ListOpts, pred query.Predicate, order *query.Order) []*datamodel.Item {
	if ld.matches == nil {
		return filterSort(ld.cfg, ld.items, pred, order)
	}
	// ListOpts carries a slice, so it cannot key a map directly; %#v renders
	// every field unambiguously.
	key := fmt.Sprintf("%#v", opts)
	if matched, ok := ld.matches[key]; ok {
		return matched
	}
	matched := filterSort(ld.cfg, ld.items, pred, order)
	ld.matches[key] = matched
	return matched
}

func filterSort(cfg *datamodel.Config, items []*datamodel.Item, pred query.Predicate, order *query.Order) []*datamodel.Item {
	matched := make([]*datamodel.Item, 0, len(items))
	for _, it := range items {
		if pred != nil && !pred(it, cfg) {
			continue
		}
		matched = append(matched, it)
	}
	sortMatched(cfg, matched, order)
	return matched
}

func (s *Store) queryOptions(ld *loaded) query.Options {
	cfg := ld.cfg
	me := sync.OnceValue(func() string { value, _ := s.identity(cfg); return value })
	return query.Options{
		Resolver:     ld.resolver,
		ActiveSprint: s.ActiveSprintKey(),
		Me:           me,
		Items:        ld.items,
	}
}

func FilterableFields() []string {
	return query.FieldNames()
}

func (s *Store) ListWithMatches(cfg *datamodel.Config, expr string) ([]datamodel.ListItem, map[string]bool, error) {
	ld, err := s.read(cfg, loadOpts{})
	if err != nil {
		return nil, nil, err
	}
	return listWithMatches(cfg, ld.items, expr, s.queryOptions(ld))
}

func listWithMatches(cfg *datamodel.Config, items []*datamodel.Item, expr string, qopts query.Options) ([]datamodel.ListItem, map[string]bool, error) {
	pred, _, _, err := ListOpts{Query: expr}.compile(cfg, qopts)
	if err != nil {
		return nil, nil, queryError(err)
	}
	parentNumbers := parentNumberMap(items)
	rows := make([]datamodel.ListItem, len(items))
	matched := make(map[string]bool, len(items))
	for i, it := range items {
		rows[i] = listItemOf(it, parentNumbers)
		if pred == nil || pred(it, cfg) {
			matched[it.ID] = true
		}
	}
	return rows, matched, nil
}

func sortMatched(cfg *datamodel.Config, matched []*datamodel.Item, order *query.Order) {
	if order == nil {
		sortByPrecedence(matched)
		return
	}
	keyOf := order.Keyer(cfg)
	priorityIndex := query.PriorityIndex(datamodel.Priorities)
	sortByKey(matched, func(it *datamodel.Item) orderedKey {
		return orderedKey{order: order, key: keyOf(it), tie: precedenceKeyOf(priorityIndex, it)}
	})
}

type orderedKey struct {
	order *query.Order
	key   query.OrderKey
	tie   precedenceKey
}

func (a orderedKey) Less(b orderedKey) bool {
	if a.order.Less(a.key, b.key) {
		return true
	}
	if a.order.Less(b.key, a.key) {
		return false
	}
	return a.tie.Less(b.tie)
}

func (opts ListOpts) compile(cfg *datamodel.Config, qopts query.Options) (query.Predicate, *query.Order, []datamodel.Warning, error) {
	var preds []query.Predicate
	var order *query.Order
	var notes []datamodel.Warning
	flat := []FieldFilter{
		{"type", opts.Type}, {"subtype", opts.Subtype}, {"state", opts.State}, {"category", opts.Category},
		{"owner", opts.Owner}, {"label", opts.Label}, {"parent", opts.Parent},
		{"priority", opts.Priority}, {"sprint", opts.Sprint}, {"board", opts.Board},
	}
	addMatch := func(f FieldFilter) error {
		if f.Value == "" {
			return nil
		}
		p, warns, err := query.Match(f.Key, f.Value, qopts)
		if err != nil {
			return err
		}
		preds = append(preds, p)
		for _, w := range warns {
			notes = append(notes, datamodel.Warning{Code: w})
		}
		return nil
	}
	for _, f := range flat {
		if err := addMatch(f); err != nil {
			return nil, nil, nil, err
		}
	}
	for _, f := range opts.Fields {
		if err := addMatch(f); err != nil {
			return nil, nil, nil, fieldFilterErr(f.Key, err)
		}
	}

	exprs := make([]string, 0, 2)
	if opts.Filter != "" {
		expr, ok := cfg.Filters[opts.Filter]
		if !ok {
			return nil, nil, nil, unknownFilterErr(cfg, opts.Filter)
		}
		exprs = append(exprs, expr)
	}
	if opts.Query != "" {
		exprs = append(exprs, opts.Query)
	}
	for _, expr := range exprs {
		c, err := query.Compile(expr, qopts)
		if err != nil {
			return nil, nil, nil, err
		}
		if c.Order != nil {
			if order != nil {
				return nil, nil, nil, errx.User("only one ORDER BY clause is allowed across --filter and the query")
			}
			order = c.Order
		}
		preds = append(preds, c.Pred)
		for _, n := range c.Notes {
			notes = append(notes, datamodel.Warning{Code: n})
		}
	}

	if len(preds) == 0 {
		return nil, order, notes, nil
	}
	pred := func(it *datamodel.Item, cfg *datamodel.Config) bool {
		for _, p := range preds {
			if !p(it, cfg) {
				return false
			}
		}
		return true
	}
	return pred, order, notes, nil
}

// groupByParent buckets rows under their immediate parent. A container heads a
// group of its own, and a container that itself has a parent additionally
// appears as a row inside that parent's group — the flat two-level shape has
// nowhere else to show the nesting.
func groupByParent(rows []datamodel.ListItem, parentNumbers map[string]string) []datamodel.TreeGroup {
	order := make([]string, 0)
	buckets := map[string][]string{}
	ensureGroup := func(key string) {
		if _, seen := buckets[key]; !seen {
			buckets[key] = []string{}
			order = append(order, key)
		}
	}
	for _, r := range rows {
		isContainer := datamodel.IsContainer(r.Type)
		if isContainer {
			ensureGroup(r.ID)
		}
		key := ""
		if r.Parent != nil {
			key = *r.Parent
		}
		if isContainer && key == "" {
			continue
		}
		ensureGroup(key)
		buckets[key] = append(buckets[key], r.ID)
	}
	sortByKey(order, func(key string) parentGroupKey {
		if key == "" {
			return parentGroupKey{orphan: true}
		}
		return parentGroupKey{k: id.NewSortKey(parentNumbers[key], key)}
	})
	groups := make([]datamodel.TreeGroup, 0, len(order))
	for _, key := range order {
		g := datamodel.TreeGroup{Items: buckets[key]}
		if key != "" {
			parent := key
			g.Parent = &parent
			if num, ok := parentNumbers[key]; ok {
				n := num
				g.ParentNumber = &n
			}
		}
		groups = append(groups, g)
	}
	return groups
}

type parentGroupKey struct {
	orphan bool
	k      id.SortKey
}

func (a parentGroupKey) Less(b parentGroupKey) bool {
	if a.orphan != b.orphan {
		return !a.orphan
	}
	return a.k.Less(b.k)
}

// Callers only ever look this up by an item's parent id, so it needs no notion
// of which types may be parents.
func parentNumberMap(items []*datamodel.Item) map[string]string {
	m := make(map[string]string, len(items))
	for _, it := range items {
		m[it.ID] = it.Number
	}
	return m
}

func listItemOf(it *datamodel.Item, parentNumbers map[string]string) datamodel.ListItem {
	li := datamodel.ListItem{
		ID:         it.ID,
		Number:     it.Number,
		Board:      boardKeyOf(it.Number),
		Title:      it.Title,
		Type:       it.Type,
		Subtype:    it.Subtype,
		State:      it.State,
		Category:   categoryString(it.Type, it.State),
		Owner:      it.Owner,
		Labels:     nonNil(it.Labels),
		Parent:     it.Parent,
		Priority:   it.Priority,
		Resolution: it.Resolution,
		Due:        it.Due,
	}
	if it.Parent != nil {
		if num, ok := parentNumbers[*it.Parent]; ok {
			li.ParentNumber = &num
		}
	}
	return li
}
