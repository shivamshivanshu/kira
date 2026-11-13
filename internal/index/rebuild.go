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
	"github.com/shivamshivanshu/kira/internal/setx"
	"github.com/shivamshivanshu/kira/internal/storage"
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
		return nil, errx.Env("beginning index tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, table := range []string{"aliases", "labels", "links", "items"} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return nil, errx.Env("clearing index %s: %v", table, err)
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
	stmts, err := prepareItemStmts(tx)
	if err != nil {
		return nil, err
	}
	defer stmts.close()
	for _, f := range files {
		if winners[f.it.ID].name != f.name {
			continue
		}
		if err := stmts.insert(f.it); err != nil {
			return nil, err
		}
	}
	if err := deleteOrphanedGitRows(tx); err != nil {
		return nil, err
	}
	return skipped, commit(tx)
}

func (i *Index) refresh(store *storage.FS, absPaths []string, prevSkips map[string]skipEntry) (map[string]skipEntry, []string, error) {
	tx, err := i.db.Begin()
	if err != nil {
		return nil, nil, errx.Env("beginning index tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	deleteByID := func(id string) error {
		if _, err := tx.Exec("DELETE FROM items WHERE id = ?", id); err != nil {
			return errx.Env("deleting index item: %v", err)
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
	queued := setx.NewDeduper[string]()
	for _, abs := range slices.Sorted(slices.Values(absPaths)) {
		if ulid := storage.ULIDFromPath(abs); ulid == "" || !queued.Add(filepath.Base(abs)) {
			continue
		}
		queue = append(queue, abs)
	}
	for _, abs := range queue {
		if err := deleteByID(storage.ULIDFromPath(abs)); err != nil {
			return nil, nil, err
		}
	}
	enqueuePartners := func(id string) {
		for _, name := range conflicts[id] {
			if queued.Add(name) {
				queue = append(queue, filepath.Join(store.ItemsDir(), name))
			}
		}
	}

	stmts, err := prepareItemStmts(tx)
	if err != nil {
		return nil, nil, err
	}
	defer stmts.close()

	skipped := map[string]skipEntry{}
	claimed := map[string]string{}
	for qi := 0; qi < len(queue); qi++ { //nolint:intrange // queue grows via enqueuePartners during iteration
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
		if err := stmts.insert(it); err != nil {
			return nil, nil, err
		}
	}
	if err := deleteOrphanedGitRows(tx); err != nil {
		return nil, nil, err
	}
	processed := make([]string, 0, len(queue))
	for _, abs := range queue {
		processed = append(processed, filepath.Base(abs))
	}
	return skipped, processed, commit(tx)
}

// deleteOrphanedGitRows drops git-derived rows for items no longer in the
// index: they have no FK to items, so nothing else cleans them up when a
// ticket is deleted. Events re-derive from git if the item ever returns;
// commit links come back on the next full trailer rescan.
func deleteOrphanedGitRows(tx *sql.Tx) error {
	for _, table := range []string{"events", "event_heads", "commit_links"} {
		if _, err := tx.Exec("DELETE FROM " + table + " WHERE item_id NOT IN (SELECT id FROM items)"); err != nil {
			return errx.Env("clearing orphaned index %s rows: %v", table, err)
		}
	}
	return nil
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

// itemStmts holds the per-item insert statements prepared once per
// transaction; modernc's sqlite driver re-prepares on every tx.Exec call,
// so re-running the same inserts per item without caching them turns an
// N-item rebuild into a multiple of N prepares.
type itemStmts struct {
	item, alias, label, link *sql.Stmt
}

func prepareItemStmts(tx *sql.Tx) (itemStmts, error) {
	var prepared []*sql.Stmt
	prepare := func(query, what string) (*sql.Stmt, error) {
		stmt, err := tx.Prepare(query)
		if err != nil {
			for _, p := range prepared {
				_ = p.Close()
			}
			return nil, errx.Env("preparing %s: %v", what, err)
		}
		prepared = append(prepared, stmt)
		return stmt, nil
	}

	item, err := prepare(`INSERT INTO items
		(id, number, type, subtype, title, state, resolution, priority, rank,
		 owner, reporter, parent, sprint, due, estimate, created, updated)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, "item insert")
	if err != nil {
		return itemStmts{}, err
	}
	alias, err := prepare("INSERT INTO aliases (item_id, ord, number) VALUES (?,?,?)", "alias insert")
	if err != nil {
		return itemStmts{}, err
	}
	label, err := prepare("INSERT INTO labels (item_id, ord, label) VALUES (?,?,?)", "label insert")
	if err != nil {
		return itemStmts{}, err
	}
	link, err := prepare("INSERT INTO links (item_id, ord, kind, target_id) VALUES (?,?,?,?)", "link insert")
	if err != nil {
		return itemStmts{}, err
	}
	return itemStmts{item: item, alias: alias, label: label, link: link}, nil
}

func (s itemStmts) close() {
	_ = s.item.Close()
	_ = s.alias.Close()
	_ = s.label.Close()
	_ = s.link.Close()
}

func (s itemStmts) insert(it *datamodel.Item) error {
	if _, err := s.item.Exec(
		it.ID, it.Number, it.Type, nullable(it.Subtype), it.Title, it.State,
		nullable(it.Resolution), nullable(it.Priority), nullable(it.Rank), nullable(it.Owner),
		nullable(it.Reporter), nullable(it.Parent), nullable(it.Sprint), nullable(it.Due),
		nullable(it.Estimate), it.Created, it.Updated); err != nil {
		return errx.Env("inserting index item %s: %v", it.ID, err)
	}
	for ord, alias := range it.Aliases {
		if _, err := s.alias.Exec(it.ID, ord, alias); err != nil {
			return errx.Env("inserting index alias: %v", err)
		}
	}
	for ord, label := range it.Labels {
		if _, err := s.label.Exec(it.ID, ord, label); err != nil {
			return errx.Env("inserting index label: %v", err)
		}
	}
	ord := 0
	for _, target := range it.BlockedBy {
		if _, err := s.link.Exec(it.ID, ord, datamodel.KeyBlockedBy, target); err != nil {
			return errx.Env("inserting index link: %v", err)
		}
		ord++
	}
	for _, kind := range datamodel.LinkTypes {
		for _, target := range it.Links[string(kind)] {
			if _, err := s.link.Exec(it.ID, ord, string(kind), target); err != nil {
				return errx.Env("inserting index link: %v", err)
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
		return errx.Env("committing index tx: %v", err)
	}
	return nil
}

func nullable[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
