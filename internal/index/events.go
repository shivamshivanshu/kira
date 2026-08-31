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
func EventsByItem(store *storage.FS, ids []string, heads map[string]string, derive func(string) ([]datamodel.Event, error)) (map[string][]datamodel.Event, error) {
	idx, err := Open(store.CacheDir())
	if err != nil {
		return nil, err
	}
	defer func() { _ = idx.Close() }()
	out := make(map[string][]datamodel.Event, len(ids))
	for _, id := range ids {
		events, err := idx.cachedEvents(id, heads[id], func() ([]datamodel.Event, error) { return derive(id) })
		if err != nil {
			return nil, err
		}
		out[id] = events
	}
	return out, nil
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
	rows, err := i.db.Query(`SELECT ts, field, old_value, new_value, commit_sha
		FROM events WHERE item_id = ? ORDER BY seq`, itemID)
	if err != nil {
		return nil, errx.Env("querying events: %v", err)
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
