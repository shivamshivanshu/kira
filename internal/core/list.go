package core

import (
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
	ld, err := s.read(cfg, loadOpts{at: opts.At, useIndex: true})
	if err != nil {
		return nil, err
	}
	cfg = ld.cfg
	items, resolver, idxNotes := ld.items, ld.resolver, ld.notes

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

func (opts ListOpts) compile(cfg *datamodel.Config, qopts query.Options) (query.Predicate, *query.Order, []datamodel.Warning, error) {
	var preds []query.Predicate
	var order *query.Order
	var notes []datamodel.Warning
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
		notes = append(notes, datamodel.Warning{Code: datamodel.WarnNoActiveSprint})
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
	sortByKey(order, func(key string) epicGroupKey {
		if key == "" {
			return epicGroupKey{orphan: true}
		}
		return epicGroupKey{k: id.NewSortKey(numOf[key], key)}
	})
	groups := make([]datamodel.TreeGroup, 0, len(order))
	for _, key := range order {
		g := datamodel.TreeGroup{Items: buckets[key]}
		if key != "" {
			epic := key
			g.Epic = &epic
			if num, ok := numOf[key]; ok {
				n := num
				g.EpicNumber = &n
			}
		}
		groups = append(groups, g)
	}
	return groups
}

type epicGroupKey struct {
	orphan bool
	k      id.SortKey
}

func (a epicGroupKey) Less(b epicGroupKey) bool {
	if a.orphan != b.orphan {
		return !a.orphan
	}
	return a.k.Less(b.k)
}

func listItemOf(cfg *datamodel.Config, it *datamodel.Item) datamodel.ListItem {
	return datamodel.ListItem{
		ID:       it.ID,
		Number:   it.Number,
		Title:    it.Title,
		Type:     it.Type,
		State:    it.State,
		Category: categoryString(cfg, it.Type, it.State),
		Owner:    it.Owner,
		Labels:   nonNil(it.Labels),
		Epic:     it.Epic,

		Priority:   it.Priority,
		Resolution: it.Resolution,
	}
}
