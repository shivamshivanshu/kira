package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func (i *Index) full(store *storage.Store) error {
	items, err := store.LoadAll()
	if err != nil {
		return err
	}
	tx, err := i.db.Begin()
	if err != nil {
		return errx.User("beginning index tx: %v", err)
	}
	defer tx.Rollback()
	for _, table := range []string{"aliases", "labels", "links", "items"} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return errx.User("clearing index %s: %v", table, err)
		}
	}
	for _, it := range items {
		if err := insertItem(tx, it); err != nil {
			return err
		}
	}
	return commit(tx)
}

func (i *Index) refresh(absPaths []string) error {
	tx, err := i.db.Begin()
	if err != nil {
		return errx.User("beginning index tx: %v", err)
	}
	defer tx.Rollback()
	for _, abs := range absPaths {
		ulid := ulidFromPath(abs)
		if ulid == "" {
			continue
		}
		if _, err := tx.Exec("DELETE FROM items WHERE id = ?", ulid); err != nil {
			return errx.User("deleting index item: %v", err)
		}
		it, err := readItem(abs)
		if err != nil {
			return err
		}
		if it == nil {
			continue
		}
		if err := insertItem(tx, it); err != nil {
			return err
		}
	}
	return commit(tx)
}

func insertItem(tx *sql.Tx, it *datamodel.Item) error {
	if _, err := tx.Exec(`INSERT INTO items
		(id, number, type, subtype, title, state, resolution, priority, rank,
		 owner, reporter, epic, sprint, due, estimate, created, updated)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		it.ID, it.Number, it.Type, nullStr(it.Subtype), it.Title, it.State,
		nullStr(it.Resolution), nullStr(it.Priority), nullStr(it.Rank), nullStr(it.Owner),
		nullStr(it.Reporter), nullStr(it.Epic), nullStr(it.Sprint), nullStr(it.Due),
		nullFloat(it.Estimate), it.Created, it.Updated); err != nil {
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
		for _, target := range it.Links[kind] {
			if _, err := tx.Exec("INSERT INTO links (item_id, ord, kind, target_id) VALUES (?,?,?,?)",
				it.ID, ord, kind, target); err != nil {
				return errx.User("inserting index link: %v", err)
			}
			ord++
		}
	}
	return nil
}

func readItem(abs string) (*datamodel.Item, error) {
	it, err := storage.ReadItem(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, errx.User("indexing %s: %v", filepath.Base(abs), err)
	}
	return it, nil
}

func dirtyState(absPaths []string) (string, []string) {
	sorted := append([]string(nil), absPaths...)
	sort.Strings(sorted)
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

func ulidFromPath(path string) string {
	ulid, _ := storage.ULIDFromFilename(filepath.Base(path))
	return ulid
}

func commit(tx *sql.Tx) error {
	if err := tx.Commit(); err != nil {
		return errx.User("committing index tx: %v", err)
	}
	return nil
}

func nullStr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
