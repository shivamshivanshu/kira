package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
	"github.com/shivamshivanshu/kira/internal/timex"
)

type parsedFile struct {
	name string
	it   *datamodel.Item
}

func (i *Index) full(store *storage.FS) (map[string]skipEntry, error) {
	names, err := store.ItemFilenames()
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
	skipped := map[string]skipEntry{}
	var files []parsedFile
	for _, name := range names {
		it, warning := readItem(filepath.Join(store.ItemsDir(), name))
		if warning != "" {
			skipped[name] = skipEntry{Note: warning}
			continue
		}
		if it == nil {
			continue
		}
		files = append(files, parsedFile{name: name, it: it})
	}
	winners := map[string]parsedFile{}
	for _, f := range files {
		cur, contested := winners[f.it.ID]
		switch {
		case !contested:
			winners[f.it.ID] = f
		case f.name == storage.ItemFilename(f.it.ID):
			skipped[cur.name] = duplicateSkip(cur.name, f.it.ID, f.name)
			winners[f.it.ID] = f
		default:
			skipped[f.name] = duplicateSkip(f.name, f.it.ID, cur.name)
		}
	}
	for _, f := range files {
		if winners[f.it.ID].name != f.name {
			continue
		}
		if err := insertItem(tx, f.it); err != nil {
			return nil, err
		}
	}
	return skipped, commit(tx)
}

func (i *Index) refresh(store *storage.FS, absPaths []string, prevSkips map[string]skipEntry) (map[string]skipEntry, []string, error) {
	tx, err := i.db.Begin()
	if err != nil {
		return nil, nil, errx.User("beginning index tx: %v", err)
	}
	defer tx.Rollback()

	deleteByID := func(id string) error {
		if _, err := tx.Exec("DELETE FROM items WHERE id = ?", id); err != nil {
			return errx.User("deleting index item: %v", err)
		}
		return nil
	}

	conflicts := map[string][]string{}
	for name, entry := range prevSkips {
		if entry.ConflictID != "" {
			conflicts[entry.ConflictID] = append(conflicts[entry.ConflictID], name)
		}
	}
	for _, names := range conflicts {
		slices.Sort(names)
	}

	var queue []string
	queued := map[string]bool{}
	for _, abs := range slices.Sorted(slices.Values(absPaths)) {
		if ulid := storage.ULIDFromPath(abs); ulid == "" || queued[filepath.Base(abs)] {
			continue
		}
		queued[filepath.Base(abs)] = true
		queue = append(queue, abs)
	}
	for _, abs := range queue {
		if err := deleteByID(storage.ULIDFromPath(abs)); err != nil {
			return nil, nil, err
		}
	}
	enqueuePartners := func(id string) {
		for _, name := range conflicts[id] {
			if queued[name] {
				continue
			}
			queued[name] = true
			queue = append(queue, filepath.Join(store.ItemsDir(), name))
		}
	}

	skipped := map[string]skipEntry{}
	claimed := map[string]string{}
	for qi := 0; qi < len(queue); qi++ {
		abs := queue[qi]
		ulid := storage.ULIDFromPath(abs)
		name := filepath.Base(abs)
		enqueuePartners(ulid)
		it, warning := readItem(abs)
		if warning != "" {
			skipped[name] = skipEntry{Note: warning}
			continue
		}
		if it == nil {
			continue
		}
		enqueuePartners(it.ID)
		if it.ID != ulid {
			canonical := storage.ItemFilename(it.ID)
			if fileExists(filepath.Join(store.ItemsDir(), canonical)) {
				skipped[name] = duplicateSkip(name, it.ID, canonical)
				continue
			}
			if first, contested := claimed[it.ID]; contested {
				skipped[name] = duplicateSkip(name, it.ID, first)
				continue
			}
			if err := deleteByID(it.ID); err != nil {
				return nil, nil, err
			}
		}
		claimed[it.ID] = name
		if err := insertItem(tx, it); err != nil {
			return nil, nil, err
		}
	}
	processed := make([]string, 0, len(queue))
	for _, abs := range queue {
		processed = append(processed, filepath.Base(abs))
	}
	return skipped, processed, commit(tx)
}

func duplicateSkip(name, ulid, other string) skipEntry {
	return skipEntry{
		Note:       storage.SkipNote(name, fmt.Errorf("duplicate id %s also in %s", ulid, other)),
		ConflictID: ulid,
	}
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
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

func readItem(abs string) (*datamodel.Item, string) {
	it, err := storage.ReadItem(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ""
		}
		return nil, storage.SkipNote(filepath.Base(abs), err)
	}
	return it, ""
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
