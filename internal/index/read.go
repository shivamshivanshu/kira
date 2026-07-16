package index

import (
	"database/sql"
	"os"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

// ReadCached reads items straight from the on-disk index cache, read-only and
// without refreshing it; it returns nil, nil if no cache exists yet.
func ReadCached(cacheDir string) ([]*datamodel.Item, error) {
	if _, err := os.Stat(dbPath(cacheDir)); err != nil {
		return nil, nil
	}
	db, err := sql.Open("sqlite", "file:"+dbPath(cacheDir)+"?mode=ro&_pragma=busy_timeout(2000)")
	if err != nil {
		return nil, errx.Env("opening index read-only: %v", err)
	}
	defer func() { _ = db.Close() }()
	return (&Index{db: db, cacheDir: cacheDir}).Items()
}

// Items returns all cached items along with their aliases, labels, and links.
func (i *Index) Items() ([]*datamodel.Item, error) {
	items, byID, err := i.scanItems()
	if err != nil {
		return nil, err
	}
	if err := i.attachAliases(byID); err != nil {
		return nil, err
	}
	if err := i.attachLabels(byID); err != nil {
		return nil, err
	}
	if err := i.attachLinks(byID); err != nil {
		return nil, err
	}
	return items, nil
}

func (i *Index) scanItems() ([]*datamodel.Item, map[string]*datamodel.Item, error) {
	rows, err := i.db.Query(`SELECT id, number, type, subtype, title, state, resolution,
		priority, rank, owner, reporter, epic, sprint, due, estimate, created, updated, activity
		FROM items ORDER BY id`)
	if err != nil {
		return nil, nil, errx.Env("querying index items: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var items []*datamodel.Item
	byID := map[string]*datamodel.Item{}
	for rows.Next() {
		var it datamodel.Item
		var subtype, resolution, priority, rank, owner, reporter, epic, sprint, due sql.NullString
		var estimate sql.NullFloat64
		if err := rows.Scan(&it.ID, &it.Number, &it.Type, &subtype, &it.Title, &it.State,
			&resolution, &priority, &rank, &owner, &reporter, &epic, &sprint, &due,
			&estimate, &it.Created, &it.Updated, &it.Activity); err != nil {
			return nil, nil, errx.Env("scanning index item: %v", err)
		}
		it.Subtype = strPtr(subtype)
		it.Resolution = strPtr(resolution)
		it.Priority = strPtr(priority)
		it.Rank = strPtr(rank)
		it.Owner = strPtr(owner)
		it.Reporter = strPtr(reporter)
		it.Epic = strPtr(epic)
		it.Sprint = strPtr(sprint)
		it.Due = strPtr(due)
		if estimate.Valid {
			it.Estimate = &estimate.Float64
		}
		items = append(items, &it)
		byID[it.ID] = &it
	}
	return items, byID, rows.Err()
}

func (i *Index) attachAliases(byID map[string]*datamodel.Item) error {
	return i.eachChild("SELECT item_id, number FROM aliases ORDER BY item_id, ord", byID,
		func(it *datamodel.Item, v string) { it.Aliases = append(it.Aliases, v) })
}

func (i *Index) attachLabels(byID map[string]*datamodel.Item) error {
	return i.eachChild("SELECT item_id, label FROM labels ORDER BY item_id, ord", byID,
		func(it *datamodel.Item, v string) { it.Labels = append(it.Labels, v) })
}

func (i *Index) attachLinks(byID map[string]*datamodel.Item) error {
	rows, err := i.db.Query("SELECT item_id, kind, target_id FROM links ORDER BY item_id, ord")
	if err != nil {
		return errx.Env("querying index links: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var itemID, kind, target string
		if err := rows.Scan(&itemID, &kind, &target); err != nil {
			return errx.Env("scanning index link: %v", err)
		}
		it := byID[itemID]
		if it == nil {
			continue
		}
		if kind == datamodel.KeyBlockedBy {
			it.BlockedBy = append(it.BlockedBy, target)
			continue
		}
		if it.Links == nil {
			it.Links = map[string][]string{}
		}
		it.Links[kind] = append(it.Links[kind], target)
	}
	return rows.Err()
}

func (i *Index) eachChild(q string, byID map[string]*datamodel.Item, add func(*datamodel.Item, string)) error {
	rows, err := i.db.Query(q)
	if err != nil {
		return errx.Env("querying index children: %v", err)
	}
	return eachPair(rows, func(r *sql.Rows) error {
		var itemID, v string
		if err := r.Scan(&itemID, &v); err != nil {
			return errx.Env("scanning index child: %v", err)
		}
		if it := byID[itemID]; it != nil {
			add(it, v)
		}
		return nil
	})
}

func eachPair(rows *sql.Rows, fn func(*sql.Rows) error) error {
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func strPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}
