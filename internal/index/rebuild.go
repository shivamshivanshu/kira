package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
	"github.com/shivamshivanshu/kira/internal/timex"
)

func (i *Index) full(store *storage.FS) ([]string, error) {
	items, warnings, err := store.LoadAll()
	if err != nil {
		return nil, err
	}
	tx, err := i.db.Begin()
	if err != nil {
		return nil, errx.User("beginning index tx: %v", err)
	}
	defer tx.Rollback()
	for _, table := range []string{"aliases", "labels", "links", "items"} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return nil, errx.User("clearing index %s: %v", table, err)
		}
	}
	for _, it := range items {
		if err := insertItem(tx, it); err != nil {
			return nil, err
		}
	}
	return warnings, commit(tx)
}

func (i *Index) refresh(absPaths []string) ([]string, error) {
	tx, err := i.db.Begin()
	if err != nil {
		return nil, errx.User("beginning index tx: %v", err)
	}
	defer tx.Rollback()
	var warnings []string
	for _, abs := range absPaths {
		ulid := storage.ULIDFromPath(abs)
		if ulid == "" {
			continue
		}
		if _, err := tx.Exec("DELETE FROM items WHERE id = ?", ulid); err != nil {
			return nil, errx.User("deleting index item: %v", err)
		}
		it, warning, err := readItem(abs)
		if err != nil {
			return nil, err
		}
		if warning != "" {
			warnings = append(warnings, warning)
		}
		if it == nil {
			continue
		}
		if err := insertItem(tx, it); err != nil {
			return nil, err
		}
	}
	return warnings, commit(tx)
}

func insertItem(tx *sql.Tx, it *datamodel.Item) error {
	if _, err := tx.Exec(`INSERT INTO items
		(id, number, type, subtype, title, state, resolution, priority, rank,
		 owner, reporter, epic, sprint, due, estimate, created, updated, activity)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		it.ID, it.Number, it.Type, nullable(it.Subtype), it.Title, it.State,
		nullable(it.Resolution), nullable(it.Priority), nullable(it.Rank), nullable(it.Owner),
		nullable(it.Reporter), nullable(it.Epic), nullable(it.Sprint), nullable(it.Due),
		nullable(it.Estimate), it.Created, it.Updated, it.Updated); err != nil {
		return errx.User("inserting index item %s: %v", it.ID, err)
	}
	for ord, alias := range it.Aliases {
		if _, err := tx.Exec("INSERT INTO aliases (item_id, ord, number) VALUES (?,?,?)", it.ID, ord, alias); err != nil {
			return errx.User("inserting index alias: %v", err)
		}
	}
	for ord, label := range it.Labels {
		if _, err := tx.Exec("INSERT INTO labels (item_id, ord, label) VALUES (?,?,?)", it.ID, ord, label); err != nil {
			return errx.User("inserting index label: %v", err)
		}
	}
	ord := 0
	for _, target := range it.BlockedBy {
		if _, err := tx.Exec("INSERT INTO links (item_id, ord, kind, target_id) VALUES (?,?,?,?)",
			it.ID, ord, datamodel.KeyBlockedBy, target); err != nil {
			return errx.User("inserting index link: %v", err)
		}
		ord++
	}
	for _, kind := range datamodel.LinkTypes {
		for _, target := range it.Links[string(kind)] {
			if _, err := tx.Exec("INSERT INTO links (item_id, ord, kind, target_id) VALUES (?,?,?,?)",
				it.ID, ord, string(kind), target); err != nil {
				return errx.User("inserting index link: %v", err)
			}
			ord++
		}
	}
	return nil
}

func readItem(abs string) (*datamodel.Item, string, error) {
	it, err := storage.ReadItem(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", nil
		}
		return nil, storage.SkipNote(filepath.Base(abs), err), nil
	}
	return it, "", nil
}

func dirtyState(absPaths []string) (string, []string) {
	sorted := append([]string(nil), absPaths...)
	slices.Sort(sorted)
	h := sha256.New()
	for _, abs := range sorted {
		h.Write([]byte(abs))
		h.Write([]byte{0})
		if data, err := os.ReadFile(abs); err == nil {
			h.Write(data)
		}
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), sorted
}

func commit(tx *sql.Tx) error {
	if err := tx.Commit(); err != nil {
		return errx.User("committing index tx: %v", err)
	}
	return nil
}

func nullable[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

func (i *Index) fillActivity() error {
	latest, err := i.latestCommitTs()
	if err != nil {
		return err
	}
	rows, err := i.db.Query("SELECT id, updated FROM items")
	if err != nil {
		return errx.User("querying item activity: %v", err)
	}
	activity := map[string]string{}
	if err := eachPair(rows, func(r *sql.Rows) error {
		var id, updated string
		if err := r.Scan(&id, &updated); err != nil {
			return errx.User("scanning item activity: %v", err)
		}
		activity[id] = updated
		if ts, ok := latest[id]; ok && laterRFC3339(ts, updated) {
			activity[id] = ts
		}
		return nil
	}); err != nil {
		return err
	}

	tx, err := i.db.Begin()
	if err != nil {
		return errx.User("beginning activity tx: %v", err)
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare("UPDATE items SET activity = ? WHERE id = ?")
	if err != nil {
		return errx.User("preparing activity update: %v", err)
	}
	defer stmt.Close()
	for id, ts := range activity {
		if _, err := stmt.Exec(ts, id); err != nil {
			return errx.User("updating item activity: %v", err)
		}
	}
	return commit(tx)
}

func (i *Index) latestCommitTs() (map[string]string, error) {
	rows, err := i.db.Query("SELECT item_id, ts FROM commit_links")
	if err != nil {
		return nil, errx.User("querying commit-link timestamps: %v", err)
	}
	latest := map[string]string{}
	if err := eachPair(rows, func(r *sql.Rows) error {
		var id, ts string
		if err := r.Scan(&id, &ts); err != nil {
			return errx.User("scanning commit-link timestamp: %v", err)
		}
		if cur, ok := latest[id]; !ok || laterRFC3339(ts, cur) {
			latest[id] = ts
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return latest, nil
}

func laterRFC3339(a, b string) bool {
	cmp, aOK, bOK := timex.CompareRFC3339(a, b)
	return aOK && bOK && cmp > 0
}
