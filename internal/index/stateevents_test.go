package index

import (
	"reflect"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestStateEventsRefreshAndReuse(t *testing.T) {
	i, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = i.Close() }()
	state := datamodel.Event{Field: "state", Old: "TODO", New: "DONE", Ts: "2026-08-31T10:00:00Z", CommitSHA: "a"}
	title := datamodel.Event{Field: "title", Old: "old", New: "new", Ts: state.Ts, CommitSHA: "a"}
	heads := map[string]string{"one": "a", "two": "b"}
	calls := 0
	derive := func(ids []string) (map[string][]datamodel.Event, error) {
		calls++
		if !reflect.DeepEqual(ids, []string{"one", "two"}) {
			t.Fatalf("stale IDs: %v", ids)
		}
		return map[string][]datamodel.Event{"one": {state, title}}, nil
	}
	for range 2 {
		got, err := i.StateEvents(heads, derive)
		if err != nil || !reflect.DeepEqual(got["one"], []datamodel.Event{state}) || len(got["two"]) != 0 {
			t.Fatalf("state events = %v, %v", got, err)
		}
	}
	if calls != 1 {
		t.Fatalf("warm cache derived events %d times", calls)
	}
	all, err := i.events("one")
	if err != nil || !reflect.DeepEqual(all, []datamodel.Event{state, title}) {
		t.Fatalf("stats dropped non-state history: %v, %v", all, err)
	}
	heads["one"] = "rewritten"
	got, err := i.StateEvents(heads, func(ids []string) (map[string][]datamodel.Event, error) {
		if !reflect.DeepEqual(ids, []string{"one"}) {
			t.Fatalf("refreshed unaffected item: %v", ids)
		}
		return nil, nil
	})
	if err != nil || len(got["one"]) != 0 {
		t.Fatalf("rewritten history survived: %v, %v", got, err)
	}
	all, err = i.events("one")
	if err != nil || len(all) != 0 {
		t.Fatalf("old events persisted: %v, %v", all, err)
	}
}

func TestStateEventsDoesNotCacheUncommittedHistory(t *testing.T) {
	i, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = i.Close() }()
	calls := 0
	for range 2 {
		_, err := i.StateEvents(map[string]string{"new": ""}, func([]string) (map[string][]datamodel.Event, error) { calls++; return nil, nil })
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatal("uncommitted item incorrectly reused cached history")
	}
}
