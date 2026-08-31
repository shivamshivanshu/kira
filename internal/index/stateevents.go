package index

import (
	"database/sql"
	"maps"
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

// StateEventsByItem returns the state events for every id in heads, opening the
// cache once for the whole batch rather than once per item.
func StateEventsByItem(store *storage.FS, heads map[string]string, derive func([]string) (map[string][]datamodel.Event, error)) (map[string][]datamodel.Event, error) {
	idx, err := Open(store.CacheDir())
	if err != nil {
		return nil, err
	}
	defer func() { _ = idx.Close() }()
	return idx.StateEvents(heads, derive)
}

// StateEvents loads state transitions for a request with one cache connection,
// reading the whole batch in two queries rather than two per item. derive
// returns all fields for stale files, so refreshing stats also warms Log.
func (i *Index) StateEvents(heads map[string]string, derive func([]string) (map[string][]datamodel.Event, error)) (map[string][]datamodel.Event, error) {
	cached, err := i.eventHeads()
	if err != nil {
		return nil, err
	}
	fresh := make(map[string]bool, len(heads))
	stale := make(map[string]string)
	for itemID, head := range heads {
		if head != "" && cached[itemID] == head {
			fresh[itemID] = true
			continue
		}
		stale[itemID] = head
	}

	out := make(map[string][]datamodel.Event, len(heads))
	if len(fresh) > 0 {
		if err := i.eachStateEvent(func(itemID string, e datamodel.Event) {
			if fresh[itemID] {
				out[itemID] = append(out[itemID], e)
			}
		}); err != nil {
			return nil, err
		}
	}
	if len(stale) == 0 {
		return out, nil
	}

	events, err := derive(slices.Sorted(maps.Keys(stale)))
	if err != nil {
		return nil, err
	}
	if err := i.replaceEventBatch(stale, events); err != nil {
		return nil, err
	}
	for itemID := range stale {
		for _, e := range events[itemID] {
			if e.Field == datamodel.KeyState {
				out[itemID] = append(out[itemID], e)
			}
		}
	}
	return out, nil
}

func (i *Index) eventHeads() (map[string]string, error) {
	rows, err := i.db.Query("SELECT item_id, head_sha FROM event_heads")
	if err != nil {
		return nil, errx.Env("querying event heads: %v", err)
	}
	heads := map[string]string{}
	return heads, eachPair(rows, func(r *sql.Rows) error {
		var itemID, head string
		if err := r.Scan(&itemID, &head); err != nil {
			return errx.Env("scanning event head: %v", err)
		}
		heads[itemID] = head
		return nil
	})
}

func (i *Index) eachStateEvent(fn func(itemID string, e datamodel.Event)) error {
	rows, err := i.db.Query(`SELECT item_id, ts, field, old_value, new_value, commit_sha
		FROM events WHERE field = ? ORDER BY item_id, seq`, datamodel.KeyState)
	if err != nil {
		return errx.Env("querying state events: %v", err)
	}
	return eachPair(rows, func(r *sql.Rows) error {
		var itemID string
		var e datamodel.Event
		if err := r.Scan(&itemID, &e.Ts, &e.Field, &e.Old, &e.New, &e.CommitSHA); err != nil {
			return errx.Env("scanning state event: %v", err)
		}
		fn(itemID, e)
		return nil
	})
}
