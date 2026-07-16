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

	stmts, err := prepareItemStmts(tx)
	if err != nil {
		return nil, nil, err
	}
	defer stmts.close()

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
		if err := stmts.insert(it); err != nil {
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

// itemStmts holds the item/alias/label/link insert statements prepared once
// per transaction; modernc's sqlite driver re-prepares on every tx.Exec call,
// so re-running the same four inserts per item without caching them turns an
// N-item rebuild into 4N prepares.
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
		 owner, reporter, epic, sprint, due, estimate, created, updated, activity)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, "item insert")
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
		nullable(it.Reporter), nullable(it.Epic), nullable(it.Sprint), nullable(it.Due),
		nullable(it.Estimate), it.Created, it.Updated, it.Updated); err != nil {
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

// fillActivity recomputes the activity column (the max of an item's updated
// timestamp and its latest linked-commit timestamp). full sweeps every item,
// the only correct option when commit_links may have lost rows out from under
// an item that wasn't itself reindexed this run (a full rebuild or a scan
// config change, both of which wipe and rebuild commit_links from scratch).
// Otherwise, insertItem already primed activity = updated for every row this
// run touched, so a plain incremental pass only needs to touch items whose
// commit_links can push activity past that baseline.
func (i *Index) fillActivity(full bool) error {
	if full {
		return i.fillActivityFull()
	}
	return i.fillActivityTargeted()
}

func (i *Index) fillActivityFull() error {
	latest, err := i.latestCommitTs()
	if err != nil {
		return err
	}
	rows, err := i.db.Query("SELECT id, updated FROM items")
	if err != nil {
		return errx.Env("querying item activity: %v", err)
	}
	activity := map[string]string{}
	if err := eachPair(rows, func(r *sql.Rows) error {
		var id, updated string
		if err := r.Scan(&id, &updated); err != nil {
			return errx.Env("scanning item activity: %v", err)
		}
		activity[id] = updated
		if ts, ok := latest[id]; ok && closesLater(ts, updated) {
			activity[id] = ts
		}
		return nil
	}); err != nil {
		return err
	}
	return applyActivity(i.db, activity)
}

func (i *Index) fillActivityTargeted() error {
	rows, err := i.db.Query(`SELECT c.item_id, c.ts, i.updated
		FROM commit_links c JOIN items i ON i.id = c.item_id`)
	if err != nil {
		return errx.Env("querying commit-link activity: %v", err)
	}
	boosted := map[string]string{}
	if err := eachPair(rows, func(r *sql.Rows) error {
		var id, ts, updated string
		if err := r.Scan(&id, &ts, &updated); err != nil {
			return errx.Env("scanning commit-link activity: %v", err)
		}
		if !closesLater(ts, updated) {
			return nil
		}
		if cur, ok := boosted[id]; !ok || closesLater(ts, cur) {
			boosted[id] = ts
		}
		return nil
	}); err != nil {
		return err
	}
	return applyActivity(i.db, boosted)
}

func applyActivity(db *sql.DB, activity map[string]string) error {
	if len(activity) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return errx.Env("beginning activity tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare("UPDATE items SET activity = ? WHERE id = ?")
	if err != nil {
		return errx.Env("preparing activity update: %v", err)
	}
	defer func() { _ = stmt.Close() }()
	for id, ts := range activity {
		if _, err := stmt.Exec(ts, id); err != nil {
			return errx.Env("updating item activity: %v", err)
		}
	}
	return commit(tx)
}

func (i *Index) latestCommitTs() (map[string]string, error) {
	rows, err := i.db.Query("SELECT item_id, ts FROM commit_links")
	if err != nil {
		return nil, errx.Env("querying commit-link timestamps: %v", err)
	}
	latest := map[string]string{}
	if err := eachPair(rows, func(r *sql.Rows) error {
		var id, ts string
		if err := r.Scan(&id, &ts); err != nil {
			return errx.Env("scanning commit-link timestamp: %v", err)
		}
		if cur, ok := latest[id]; !ok || closesLater(ts, cur) {
			latest[id] = ts
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return latest, nil
}
