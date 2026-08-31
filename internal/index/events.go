package index

import (
	"database/sql"
	"errors"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

// LogEntries returns an item's cached events and commit links, recomputing the
// events from derive when the cache is missing or stale for fileHead.
func LogEntries(store *storage.FS, itemID, fileHead string, derive func() ([]datamodel.Event, error)) ([]datamodel.Event, []CommitLink, error) {
	idx, err := Open(store.CacheDir())
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = idx.Close() }()
	events, err := idx.cachedEvents(itemID, fileHead, derive)
	if err != nil {
		return nil, nil, err
	}
	links, err := idx.CommitLinks(itemID)
	if err != nil {
		return nil, nil, err
	}
	return events, links, nil
}

// EventsByItem returns the cached events for every id in one pass, opening the
// index once for the whole batch rather than once per item; derive recomputes
// an item whose cache is missing or stale for its heads entry.
//
// The cached rows come back in two queries for the whole batch rather than
// two per item; only the stale remainder, usually empty, goes through derive.
func EventsByItem(store *storage.FS, ids []string, heads map[string]string, derive func(string) ([]datamodel.Event, error)) (map[string][]datamodel.Event, error) {
	idx, err := Open(store.CacheDir())
	if err != nil {
		return nil, err
	}
	defer func() { _ = idx.Close() }()

	out := make(map[string][]datamodel.Event, len(ids))
	stale, err := idx.readCachedEvents(ids, heads, out)
	if err != nil {
		return nil, err
	}
	for _, id := range stale {
		events, err := derive(id)
		if err != nil {
			return nil, err
		}
		if head := heads[id]; head != "" {
			if err := idx.replaceEvents(id, head, events); err != nil {
				return nil, err
			}
		}
		out[id] = events
	}
	return out, nil
}

// readCachedEvents fills out with every id whose cached head still matches
// heads, and returns the ids it could not serve from the cache.
func (i *Index) readCachedEvents(ids []string, heads map[string]string, out map[string][]datamodel.Event) ([]string, error) {
	cachedHeads, err := i.eventHeads()
	if err != nil {
		return nil, err
	}
	var stale []string
	var wanted []string
	for _, id := range ids {
		if h := heads[id]; h == "" || cachedHeads[id] != h {
			stale = append(stale, id)
			continue
		}
		wanted = append(wanted, id)
	}
	if len(wanted) == 0 {
		return stale, nil
	}
	all, err := i.allEvents()
	if err != nil {
		return nil, err
	}
	for _, id := range wanted {
		out[id] = all[id]
	}
	return stale, nil
}

func (i *Index) eventHeads() (map[string]string, error) {
	rows, err := i.db.Query("SELECT item_id, head_sha FROM event_heads")
	if err != nil {
		return nil, errx.Env("querying event heads: %v", err)
	}
	heads := map[string]string{}
	return heads, eachPair(rows, func(r *sql.Rows) error {
		var id, head string
		if err := r.Scan(&id, &head); err != nil {
			return errx.Env("scanning event head: %v", err)
		}
		heads[id] = head
		return nil
	})
}

func (i *Index) allEvents() (map[string][]datamodel.Event, error) {
	rows, err := i.db.Query(`SELECT item_id, ts, field, old_value, new_value, commit_sha
		FROM events ORDER BY item_id, seq`)
	if err != nil {
		return nil, errx.Env("querying events: %v", err)
	}
	byItem := map[string][]datamodel.Event{}
	return byItem, eachPair(rows, func(r *sql.Rows) error {
		var id string
		var e datamodel.Event
		if err := r.Scan(&id, &e.Ts, &e.Field, &e.Old, &e.New, &e.CommitSHA); err != nil {
			return errx.Env("scanning event: %v", err)
		}
		byItem[id] = append(byItem[id], e)
		return nil
	})
}

func (i *Index) cachedEvents(itemID, fileHead string, derive func() ([]datamodel.Event, error)) ([]datamodel.Event, error) {
	if fileHead == "" {
		return derive()
	}
	head, err := i.eventHead(itemID)
	if err != nil {
		return nil, err
	}
	if head == fileHead {
		return i.events(itemID)
	}
	events, err := derive()
	if err != nil {
		return nil, err
	}
	if err := i.replaceEvents(itemID, fileHead, events); err != nil {
		return nil, err
	}
	return events, nil
}

func (i *Index) eventHead(itemID string) (string, error) {
	var head string
	err := i.db.QueryRow("SELECT head_sha FROM event_heads WHERE item_id = ?", itemID).Scan(&head)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", errx.Env("querying event head: %v", err)
	}
	return head, nil
}

func (i *Index) replaceEvents(itemID, head string, events []datamodel.Event) error {
	tx, err := i.db.Begin()
	if err != nil {
		return errx.Env("beginning events tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("DELETE FROM events WHERE item_id = ?", itemID); err != nil {
		return errx.Env("clearing events: %v", err)
	}
	for seq, e := range events {
		if _, err := tx.Exec(`INSERT INTO events
			(item_id, seq, ts, field, old_value, new_value, commit_sha) VALUES (?,?,?,?,?,?,?)`,
			itemID, seq, e.Ts, e.Field, e.Old, e.New, e.CommitSHA); err != nil {
			return errx.Env("inserting event: %v", err)
		}
	}
	if _, err := tx.Exec(`INSERT INTO event_heads (item_id, head_sha) VALUES (?,?)
		ON CONFLICT(item_id) DO UPDATE SET head_sha = excluded.head_sha`, itemID, head); err != nil {
		return errx.Env("updating event head: %v", err)
	}
	return commit(tx)
}

func (i *Index) events(itemID string) ([]datamodel.Event, error) {
	return scanEvents(i.db.Query(`SELECT ts, field, old_value, new_value, commit_sha
		FROM events WHERE item_id = ? ORDER BY seq`, itemID))
}

func scanEvents(rows *sql.Rows, queryErr error) ([]datamodel.Event, error) {
	if queryErr != nil {
		return nil, errx.Env("querying events: %v", queryErr)
	}
	defer func() { _ = rows.Close() }()
	var out []datamodel.Event
	for rows.Next() {
		var e datamodel.Event
		if err := rows.Scan(&e.Ts, &e.Field, &e.Old, &e.New, &e.CommitSHA); err != nil {
			return nil, errx.Env("scanning event: %v", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
