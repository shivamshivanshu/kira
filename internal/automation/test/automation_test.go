package automation_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/automation"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

func TestMatchesWildcardWhenNoMatchBlock(t *testing.T) {
	h := datamodel.AutomationHook{On: datamodel.EventItemStateChanged}
	ev := automation.Event{Name: datamodel.EventItemStateChanged, To: "done", Item: &datamodel.ShowResult{Type: "ticket"}}
	if !automation.Matches(h, ev) {
		t.Fatal("hook with no match block should fire on its event")
	}
}

func TestMatchesEventNameGate(t *testing.T) {
	h := datamodel.AutomationHook{On: datamodel.EventItemCreated}
	ev := automation.Event{Name: datamodel.EventItemStateChanged}
	if automation.Matches(h, ev) {
		t.Fatal("hook must not fire on a different event")
	}
}

func TestMatchesToByStateKeyAndCategory(t *testing.T) {
	ev := automation.Event{Name: datamodel.EventItemStateChanged, To: "shipped", ToCategory: "done"}

	byKey := datamodel.AutomationHook{On: datamodel.EventItemStateChanged, Match: &datamodel.AutomationMatch{To: "shipped"}}
	byCat := datamodel.AutomationHook{On: datamodel.EventItemStateChanged, Match: &datamodel.AutomationMatch{To: "done"}}
	miss := datamodel.AutomationHook{On: datamodel.EventItemStateChanged, Match: &datamodel.AutomationMatch{To: "todo"}}

	if !automation.Matches(byKey, ev) || !automation.Matches(byCat, ev) {
		t.Fatal("to should match on state key or category")
	}
	if automation.Matches(miss, ev) {
		t.Fatal("to should not match an unrelated state")
	}
}

func TestMatchesTypeAndFrom(t *testing.T) {
	ev := automation.Event{Name: datamodel.EventItemStateChanged, From: "todo", FromCategory: "todo", Item: &datamodel.ShowResult{Type: "epic"}}
	wrongType := datamodel.AutomationHook{On: datamodel.EventItemStateChanged, Match: &datamodel.AutomationMatch{Type: "ticket"}}
	rightFrom := datamodel.AutomationHook{On: datamodel.EventItemStateChanged, Match: &datamodel.AutomationMatch{From: "todo", Type: "epic"}}
	if automation.Matches(wrongType, ev) {
		t.Fatal("type mismatch should block")
	}
	if !automation.Matches(rightFrom, ev) {
		t.Fatal("matching from+type should fire")
	}
}

func TestMatchesFromByCategoryDistinctFromStateKey(t *testing.T) {
	ev := automation.Event{Name: datamodel.EventItemStateChanged, From: "in_progress", FromCategory: "doing"}
	byKey := datamodel.AutomationHook{On: datamodel.EventItemStateChanged, Match: &datamodel.AutomationMatch{From: "in_progress"}}
	byCat := datamodel.AutomationHook{On: datamodel.EventItemStateChanged, Match: &datamodel.AutomationMatch{From: "doing"}}
	miss := datamodel.AutomationHook{On: datamodel.EventItemStateChanged, Match: &datamodel.AutomationMatch{From: "todo"}}
	if !automation.Matches(byKey, ev) {
		t.Fatal("from should match the state key")
	}
	if !automation.Matches(byCat, ev) {
		t.Fatal("from should match the category when it differs from the state key")
	}
	if automation.Matches(miss, ev) {
		t.Fatal("from should not match an unrelated value")
	}
}

func TestHashChangesWithConfigAndIsStable(t *testing.T) {
	a := &datamodel.Config{Automation: []datamodel.AutomationHook{{On: datamodel.EventItemCreated, Run: "true"}}}
	b := &datamodel.Config{Automation: []datamodel.AutomationHook{{On: datamodel.EventItemCreated, Run: "false"}}}
	if automation.Hash(a) != automation.Hash(a) {
		t.Fatal("hash must be deterministic for identical config")
	}
	if automation.Hash(a) == automation.Hash(b) {
		t.Fatal("editing a hook must change the hash")
	}
}

func TestHashIsPinnedToTheJSONContract(t *testing.T) {
	cfg := &datamodel.Config{Automation: []datamodel.AutomationHook{
		{Name: "notify", On: datamodel.EventItemCreated, Run: "echo hi", Timeout: "5s"},
	}}
	const want = "8747084ea3e7ce1aef46e2dd64e19cfb63c8b515a8d948c54b081b70a6cb3d75"
	if got := automation.Hash(cfg); got != want {
		t.Fatalf("Hash = %s, want %s (json contract for AutomationHook changed — this breaks every granted trust file)", got, want)
	}
}

func TestTrustRoundTripAndRevokeOnEdit(t *testing.T) {
	dir := t.TempDir()
	cfg := &datamodel.Config{Automation: []datamodel.AutomationHook{{Name: "x", On: datamodel.EventItemCreated, Run: "true"}}}

	if automation.Trusted(dir, cfg) {
		t.Fatal("fresh cache dir must not be trusted")
	}
	granted, err := automation.Grant(dir, cfg)
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if granted != automation.Hash(cfg) {
		t.Fatal("Grant must return the hash it wrote")
	}
	if !automation.Trusted(dir, cfg) {
		t.Fatal("granted config must be trusted")
	}

	path := filepath.Join(dir, "automation.trust")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trust file: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("rewrite trust file: %v", err)
	}
	if !automation.Trusted(dir, cfg) {
		t.Fatal("an editor-added trailing newline in the trust file must not revoke trust")
	}

	cfg.Automation[0].Run = "false"
	if automation.Trusted(dir, cfg) {
		t.Fatal("editing config must auto-revoke trust")
	}
}

func TestPayloadShapeForStateChanged(t *testing.T) {
	ev := automation.Event{
		Name:         datamodel.EventItemStateChanged,
		Source:       datamodel.SourceCLI,
		Item:         &datamodel.ShowResult{ID: "01ABC", Number: "KIRA-1", State: "done"},
		Changes:      map[string]automation.Change{"state": {Old: "todo", New: "done"}},
		From:         "todo",
		To:           "done",
		FromCategory: "doing",
		ToCategory:   "done",
		Commit:       "deadbeef",
	}
	raw, err := automation.Payload(ev, "/repo", "2026-07-13T00:00:00Z", automation.Actor{Name: "t", Email: "t@e"})
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["payload_version"].(float64) != 1 {
		t.Fatalf("payload_version = %v, want 1", got["payload_version"])
	}
	for _, k := range []string{"event", "source", "ts", "repo", "actor", "item", "changes", "from", "to", "to_category", "from_category", "commit"} {
		if _, ok := got[k]; !ok {
			t.Fatalf("payload missing key %q: %s", k, raw)
		}
	}
	if got["event"] != string(datamodel.EventItemStateChanged) {
		t.Fatalf("event = %v", got["event"])
	}
	item := got["item"].(map[string]any)
	if item["number"] != "KIRA-1" {
		t.Fatalf("item.number = %v", item["number"])
	}
}

func TestPayloadOmitsItemForSync(t *testing.T) {
	ev := automation.Event{Name: datamodel.EventSyncCompleted, Source: datamodel.SourceSync}
	raw, err := automation.Payload(ev, "/repo", "2026-07-13T00:00:00Z", automation.Actor{})
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	if strings.Contains(string(raw), `"item"`) {
		t.Fatalf("sync.completed payload should omit item: %s", raw)
	}
}

func TestEnabledDefaultsTrue(t *testing.T) {
	if !(datamodel.AutomationHook{}).IsEnabled() {
		t.Fatal("hook enabled must default to true")
	}
	if (datamodel.AutomationHook{Enabled: ptr.To(false)}).IsEnabled() {
		t.Fatal("explicit enabled:false must disable")
	}
}
