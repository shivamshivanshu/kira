package core

import (
	"sort"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
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
}

func (s *Store) List(cfg *config.Config, opts ListOpts) (*ListResult, error) {
	items, _, resolver, err := s.load(cfg)
	if err != nil {
		return nil, err
	}

	qopts := query.Options{
		Resolver:     resolver,
		Priorities:   cfg.Priorities,
		ActiveSprint: s.ActiveSprintKey(),
	}
	pred, order, notes, err := opts.compile(cfg, qopts)
	if err != nil {
		return nil, userErr("%v", err)
	}

	matched := make([]*item.Item, 0, len(items))
	for _, it := range items {
		if pred != nil && !pred(it, cfg) {
			continue
		}
		matched = append(matched, it)
	}
	sortMatched(cfg, matched, order)

	rows := make([]ListItem, len(matched))
	for i, it := range matched {
		rows[i] = listItemOf(cfg, it)
	}
	res := &ListResult{Items: rows, Count: len(rows), StderrNotes: notes}
	if opts.Tree {
		res.Tree = groupByEpic(rows, items)
	}
	return res, nil
}

func sortMatched(cfg *config.Config, matched []*item.Item, order *query.Order) {
	if order == nil {
		sortByPrecedence(cfg, matched)
		return
	}
	keyOf := order.Keyer(cfg)
	priorityIndex := query.PriorityIndex(cfg.Priorities)
	sortByKey(matched, func(it *item.Item) orderedKey {
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

func (opts ListOpts) compile(cfg *config.Config, qopts query.Options) (query.Predicate, *query.Order, []string, error) {
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
				return nil, nil, nil, userErr("only one ORDER BY clause is allowed across --filter and the query")
			}
			order = c.Order
		}
		preds = append(preds, c.Pred)
		notes = append(notes, c.Notes...)
	}

	if len(preds) == 0 {
		return nil, order, notes, nil
	}
	pred := func(it *item.Item, cfg *config.Config) bool {
		for _, p := range preds {
			if !p(it, cfg) {
				return false
			}
		}
		return true
	}
	return pred, order, notes, nil
}

func unknownFilterErr(cfg *config.Config, name string) error {
	if len(cfg.Filters) == 0 {
		return userErr("unknown filter %q (no filters configured)", name)
	}
	return userErr("unknown filter %q (available: %s)", name, strings.Join(filterNames(cfg), ", "))
}

func filterNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Filters))
	for name := range cfg.Filters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type FilterView struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

type FilterListResult struct {
	Filters []FilterView `json:"filters"`
}

func Filters(cfg *config.Config) *FilterListResult {
	views := make([]FilterView, 0, len(cfg.Filters))
	for _, name := range filterNames(cfg) {
		views = append(views, FilterView{Name: name, Query: cfg.Filters[name]})
	}
	return &FilterListResult{Filters: views}
}

// One flat level (epic -> member ULIDs, orphan bucket last) — the shape the
// query tree key specs; recursive hierarchy is kira tree's job. An epic heads
// its own group and is never a member; a group is created even for an epic
// filtered out of the results, so children never appear detached.
func groupByEpic(rows []ListItem, items []*item.Item) []TreeGroup {
	numOf := map[string]string{}
	for _, it := range items {
		if it.Type == item.TypeEpic {
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
		if r.Type == item.TypeEpic {
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
	sort.SliceStable(ds, func(i, j int) bool {
		if ds[i].orphan || ds[j].orphan {
			return !ds[i].orphan
		}
		return ds[i].k.Less(ds[j].k)
	})
	groups := make([]TreeGroup, 0, len(ds))
	for _, d := range ds {
		g := TreeGroup{Items: buckets[d.key]}
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
