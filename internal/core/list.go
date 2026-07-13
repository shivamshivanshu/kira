package core

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/query"
)

type ListOpts struct {
	Type     string
	State    string
	Category string
	Owner    string
	Label    string
	Epic     string
	Priority string
	Sprint   string
	Filter   string
	Query    string
	Tree     bool
	At       string
}

func (s *Store) List(cfg *datamodel.Config, opts ListOpts) (*datamodel.ListResult, error) {
	var items []*datamodel.Item
	var resolver *id.Resolver
	var idxNotes []string
	var err error
	if opts.At != "" {
		items, resolver, cfg, err = s.listView(cfg, opts.At)
	} else {
		items, _, resolver, idxNotes, err = s.indexedLoad(cfg)
	}
	if err != nil {
		return nil, err
	}

	pred, order, notes, err := opts.compile(cfg, s.queryOptions(cfg, resolver))
	if err != nil {
		return nil, errx.User("%v", err)
	}

	matched := make([]*datamodel.Item, 0, len(items))
	for _, it := range items {
		if pred != nil && !pred(it, cfg) {
			continue
		}
		matched = append(matched, it)
	}
	sortMatched(cfg, matched, order)

	rows := make([]datamodel.ListItem, len(matched))
	for i, it := range matched {
		rows[i] = listItemOf(cfg, it)
	}
	res := &datamodel.ListResult{Items: rows, Count: len(rows), StderrNotes: append(idxNotes, notes...)}
	if opts.Tree {
		res.Tree = groupByEpic(rows, items)
	}
	return res, nil
}

func (s *Store) queryOptions(cfg *datamodel.Config, resolver *id.Resolver) query.Options {
	return query.Options{Resolver: resolver, Priorities: cfg.Priorities, ActiveSprint: s.ActiveSprintKey()}
}

func (s *Store) ListWithMatches(cfg *datamodel.Config, expr string) ([]datamodel.ListItem, map[string]bool, error) {
	items, _, resolver, _, err := s.load(cfg)
	if err != nil {
		return nil, nil, err
	}
	pred, _, _, err := ListOpts{Query: expr}.compile(cfg, s.queryOptions(cfg, resolver))
	if err != nil {
		return nil, nil, errx.User("%v", err)
	}
	rows := make([]datamodel.ListItem, len(items))
	matched := make(map[string]bool, len(items))
	for i, it := range items {
		rows[i] = listItemOf(cfg, it)
		if pred == nil || pred(it, cfg) {
			matched[it.ID] = true
		}
	}
	return rows, matched, nil
}

func sortMatched(cfg *datamodel.Config, matched []*datamodel.Item, order *query.Order) {
	if order == nil {
		sortByPrecedence(cfg, matched)
		return
	}
	keyOf := order.Keyer(cfg)
	priorityIndex := query.PriorityIndex(cfg.Priorities)
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

func (opts ListOpts) compile(cfg *datamodel.Config, qopts query.Options) (query.Predicate, *query.Order, []string, error) {
	var preds []query.Predicate
	var order *query.Order
	var notes []string
	flat := []struct{ field, value string }{
		{"type", opts.Type}, {"state", opts.State}, {"category", opts.Category},
		{"owner", opts.Owner}, {"label", opts.Label}, {"epic", opts.Epic},
		{"priority", opts.Priority}, {"sprint", opts.Sprint},
	}
	for _, f := range flat {
		if f.value == "" {
			continue
		}
		p, err := query.Match(f.field, f.value, qopts)
		if err != nil {
			return nil, nil, nil, err
		}
		preds = append(preds, p)
	}
	if opts.Sprint == "active" && qopts.ActiveSprint == "" {
		notes = append(notes, query.NoActiveSprintNote)
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
		notes = append(notes, c.Notes...)
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

func groupByEpic(rows []datamodel.ListItem, items []*datamodel.Item) []datamodel.TreeGroup {
	numOf := map[string]string{}
	for _, it := range items {
		if it.Type == datamodel.TypeEpic {
			numOf[it.ID] = it.Number
		}
	}
	order := make([]string, 0)
	buckets := map[string][]string{}
	ensureGroup := func(key string) {
		if _, seen := buckets[key]; !seen {
			buckets[key] = []string{}
			order = append(order, key)
		}
	}
	for _, r := range rows {
		if r.Type == datamodel.TypeEpic {
			ensureGroup(r.ID)
			continue
		}
		key := ""
		if r.Epic != nil {
			key = *r.Epic
		}
		ensureGroup(key)
		buckets[key] = append(buckets[key], r.ID)
	}
	type dec struct {
		key    string
		k      id.SortKey
		orphan bool
	}
	ds := make([]dec, len(order))
	for i, key := range order {
		if key == "" {
			ds[i] = dec{orphan: true}
			continue
		}
		ds[i] = dec{key: key, k: id.NewSortKey(numOf[key], key)}
	}
	slices.SortStableFunc(ds, func(a, b dec) int {
		if a.orphan || b.orphan {
			switch {
			case !a.orphan:
				return -1
			case !b.orphan:
				return 1
			default:
				return 0
			}
		}
		switch {
		case a.k.Less(b.k):
			return -1
		case b.k.Less(a.k):
			return 1
		default:
			return 0
		}
	})
	groups := make([]datamodel.TreeGroup, 0, len(ds))
	for _, d := range ds {
		g := datamodel.TreeGroup{Items: buckets[d.key]}
		if !d.orphan {
			epic := d.key
			g.Epic = &epic
			if num, ok := numOf[d.key]; ok {
				n := num
				g.EpicNumber = &n
			}
		}
		groups = append(groups, g)
	}
	return groups
}
