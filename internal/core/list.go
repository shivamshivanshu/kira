package core

import (
	"sort"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
	"github.com/shivamshivanshu/kira/internal/query"
)

// ListOpts are the list/query filters (docs/design/04-cli.md list, query). The
// flat filters (empty = inactive) and Query are ANDed together. Tree requests
// the epic grouping (list --tree / query's default render).
type ListOpts struct {
	Type     string
	State    string
	Category string
	Owner    string
	Label    string
	Epic     string // ULID or number, resolved via the query engine
	Query    string // query-grammar expression, ANDed with the flat filters
	Tree     bool   // group results by epic
}

// List scans every ticket file, applies the flat filters and any query
// expression (ANDed), and returns rows sorted by display number ascending, ties
// broken by ULID (docs/design/04-cli.md §7). When opts.Tree is set the result
// also carries the epic grouping.
func (s *Store) List(cfg *config.Config, opts ListOpts) (*ListResult, error) {
	items, _, resolver, err := s.load(cfg)
	if err != nil {
		return nil, err
	}

	pred, err := opts.predicate(resolver)
	if err != nil {
		return nil, userErr("%v", err)
	}

	rows := make([]ListItem, 0, len(items))
	for _, it := range items {
		if pred != nil && !pred(it, cfg) {
			continue
		}
		rows = append(rows, listItemOf(cfg, it))
	}
	sortRows(rows)

	res := &ListResult{Items: rows, Count: len(rows)}
	if opts.Tree {
		res.Tree = groupByEpic(rows, items)
	}
	return res, nil
}

// predicate lowers every active filter — the flat flags and the query
// expression alike — into one conjoined query.Predicate, so list and query
// share a single comparison and epic-resolution engine rather than a hand-
// written second one. It returns nil when no filter is active (match all).
func (opts ListOpts) predicate(resolver *id.Resolver) (query.Predicate, error) {
	var preds []query.Predicate
	flat := []struct{ field, value string }{
		{"type", opts.Type}, {"state", opts.State}, {"category", opts.Category},
		{"owner", opts.Owner}, {"label", opts.Label}, {"epic", opts.Epic},
	}
	for _, f := range flat {
		if f.value == "" {
			continue
		}
		p, err := query.Match(f.field, f.value, resolver)
		if err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	if opts.Query != "" {
		p, err := query.Compile(opts.Query, resolver)
		if err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	if len(preds) == 0 {
		return nil, nil
	}
	return func(it *item.Item, cfg *config.Config) bool {
		for _, p := range preds {
			if !p(it, cfg) {
				return false
			}
		}
		return true
	}, nil
}

// groupByEpic buckets the result rows by parent epic for tree rendering. The
// grouping is deliberately flat (one level: epic -> member ULIDs, plus an
// orphan bucket), which is the shape docs/design/04-cli.md query specs for the
// tree key; the recursive multi-level hierarchy is kira tree's job, not this.
// A non-epic row joins its parent epic's group (or the orphan bucket when it
// has no epic); an epic row heads its own group and is never itself a member,
// so it is not double-listed as an orphan. A group is created for any epic referenced
// by a child even if that epic is filtered out of the results, so children
// never appear detached. Groups are ordered by the epic's display number
// ascending, orphan bucket last; epic_number is filled from the loaded item set
// (null for a dangling pointer).
func groupByEpic(rows []ListItem, items []*item.Item) []TreeGroup {
	// Only epics are ever looked up here (every bucket key is an epic ULID), so
	// index just those.
	numOf := map[string]string{}
	for _, it := range items {
		if it.Type == item.TypeEpic {
			numOf[it.ID] = it.Number
		}
	}
	order := make([]string, 0)       // epic ULIDs in first-seen order; "" = orphan
	buckets := map[string][]string{} // epic ULID -> member ULIDs
	ensure := func(key string) {
		if _, seen := buckets[key]; !seen {
			buckets[key] = []string{}
			order = append(order, key)
		}
	}
	for _, r := range rows {
		if r.Type == item.TypeEpic {
			ensure(r.ID) // an epic heads its own group; not a member
			continue
		}
		key := ""
		if r.Epic != nil {
			key = *r.Epic
		}
		ensure(key)
		buckets[key] = append(buckets[key], r.ID)
	}
	// Decorate each bucket key with its sort key; the orphan bucket ("") sorts
	// last, every epic bucket by its display number.
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
			return !ds[i].orphan // orphan bucket last
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

// sortRows orders rows by the shared display-number key (docs/design/04-cli.md §7).
func sortRows(rows []ListItem) {
	sortByKey(rows, func(r ListItem) id.SortKey { return id.NewSortKey(r.Number, r.ID) })
}
