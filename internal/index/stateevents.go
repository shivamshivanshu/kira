package index

import (
	"maps"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

// StateEvents loads state transitions for a request with one cache connection.
// derive returns all fields for stale files, so refreshing stats also warms Log.
func (i *Index) StateEvents(heads map[string]string, derive func([]string) (map[string][]datamodel.Event, error)) (map[string][]datamodel.Event, error) {
	out := make(map[string][]datamodel.Event, len(heads))
	stale := maps.Clone(heads)
	ids := slices.Sorted(maps.Keys(heads))
	for chunk := range slices.Chunk(ids, 256) {
		args := make([]any, len(chunk))
		for n, id := range chunk {
			args[n] = id
		}
		in := strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")
		rows, err := i.db.Query("SELECT item_id, head_sha FROM event_heads WHERE item_id IN ("+in+")", args...)
		if err != nil {
			return nil, errx.Env("querying event heads: %v", err)
		}
		for rows.Next() {
			var id, head string
			if err := rows.Scan(&id, &head); err != nil {
				_ = rows.Close()
				return nil, err
			}
			if head != "" && head == heads[id] {
				delete(stale, id)
			}
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return nil, err
		}
		rows, err = i.db.Query(`SELECT item_id, ts, field, old_value, new_value, commit_sha FROM events
			WHERE field = 'state' AND item_id IN (`+in+`) ORDER BY item_id, seq`, args...)
		if err != nil {
			return nil, errx.Env("querying state events: %v", err)
		}
		for rows.Next() {
			var id string
			var e datamodel.Event
			if err := rows.Scan(&id, &e.Ts, &e.Field, &e.Old, &e.New, &e.CommitSHA); err != nil {
				_ = rows.Close()
				return nil, err
			}
			if _, missing := stale[id]; !missing {
				out[id] = append(out[id], e)
			}
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return nil, err
		}
	}
	if len(stale) > 0 {
		events, err := derive(slices.Sorted(maps.Keys(stale)))
		if err != nil {
			return nil, err
		}
		if err := i.replaceEventBatch(stale, events); err != nil {
			return nil, err
		}
		for id := range stale {
			for _, e := range events[id] {
				if e.Field == datamodel.KeyState {
					out[id] = append(out[id], e)
				}
			}
		}
	}
	return out, nil
}
