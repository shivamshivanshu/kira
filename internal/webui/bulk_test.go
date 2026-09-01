package webui

import (
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
)

func TestBulkPartialWorkflowResultsAndDeduplication(t *testing.T) {
	d := newDashboardTest(t, Options{})
	allowed := d.create(t, core.CreateOpts{Title: "Ready to close"})
	blocked := d.create(t, core.CreateOpts{Title: "Still TODO"})
	unselected := d.create(t, core.CreateOpts{Title: "Leave alone"})
	blocker := d.create(t, core.CreateOpts{Title: "Open blocker"})
	// states are a free enum, so an open blocker on a done-category move is
	// what makes one item of the batch fail.
	if res := d.post("link", url.Values{"id": {blocked.ID}, "kind": {"blocked-by"}, "target": {blocker.Number}}); res.Code >= 400 {
		t.Fatalf("link blocked-by: %d %s", res.Code, res.Body.String())
	}
	s := &server{store: d.store}
	form := url.Values{"ids": {allowed.Number, allowed.ID, blocked.ID}, "operation": {"move"}, "value": {"DONE"}}
	outcomes, err := s.bulk(d.cfg, form)
	if err != nil || len(outcomes) != 2 || outcomes[0].Error != "" || outcomes[1].Error == "" {
		t.Fatalf("expected one success and one failure: %+v %v", outcomes, err)
	}
	retry := bulkRetry(form, outcomes)
	if !slices.Equal(retry["ids"], []string{blocked.ID}) || retry.Get("operation") != "move" || retry.Get("value") != "DONE" {
		t.Fatalf("retry included successful items: %v", retry)
	}
	for ref, state := range map[string]string{allowed.ID: "DONE", blocked.ID: "TODO", unselected.ID: "TODO"} {
		it, err := d.store.Show(d.cfg, ref, "")
		if err != nil || it.State != state {
			t.Errorf("%s: state=%+v err=%v, want %s", ref, it, err, state)
		}
	}
	res := requireAction(t, d, "bulk", url.Values{"ids": {allowed.ID, blocked.ID}, "operation": {"move"}, "value": {"DONE"}, "resolution": {"done"}})
	page := followAction(d, res)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "1 succeeded, 1 failed") || !strings.Contains(page.Body.String(), "Still TODO") {
		t.Fatalf("partial outcome not presented: %d %s", page.Code, page.Body.String())
	}
	checked := regexp.MustCompile(`<input[^>]*name="ids"[^>]*value="([^"]+)"[^>]*checked`).FindAllStringSubmatch(page.Body.String(), -1)
	if len(checked) != 1 || checked[0][1] != blocked.ID {
		t.Fatalf("partial result must select only the failed immutable ID: %v", checked)
	}
}

func TestBulkCoreOperationsAndSelectionBoundaries(t *testing.T) {
	d := newDashboardTest(t, Options{})
	selected := d.create(t, core.CreateOpts{Title: "Selected"})
	unselected := d.create(t, core.CreateOpts{Title: "Not selected"})
	requireAction(t, d, "board-create", url.Values{"key": {"ALT"}, "name": {"Alternate"}})
	requireAction(t, d, "sprint-create", url.Values{"key": {"s1"}, "name": {"Sprint"}, "start": {"2026-08-01"}, "end": {"2026-08-14"}})
	for _, step := range []struct{ operation, value string }{
		{"assign", "tester"}, {"label-add", "selected-label"}, {"sprint", "s1"}, {"board-move", "ALT"},
	} {
		requireAction(t, d, "bulk", url.Values{"ids": {selected.ID}, "operation": {step.operation}, "value": {step.value}})
	}
	cfg, err := d.store.Config()
	if err != nil {
		t.Fatal(err)
	}
	it, err := d.store.Show(cfg, selected.ID, "")
	if err != nil || it.Owner == nil || *it.Owner != "tester" || !slices.Contains(it.Labels, "selected-label") || it.Sprint == nil || *it.Sprint != "s1" || it.Board != "ALT" {
		t.Fatalf("bulk changes not persisted: %+v %v", it, err)
	}
	other, err := d.store.Show(cfg, unselected.ID, "")
	if err != nil || other.Owner != nil || len(other.Labels) != 0 || other.Sprint != nil || other.Board != "WEB" {
		t.Fatalf("bulk changed unselected item: %+v %v", other, err)
	}
	for _, step := range []struct{ operation, value string }{
		{"assign", ""}, {"label-rm", "selected-label"}, {"sprint", ""},
	} {
		requireAction(t, d, "bulk", url.Values{"ids": {selected.ID}, "operation": {step.operation}, "value": {step.value}})
	}
	it, err = d.store.Show(cfg, selected.ID, "")
	if err != nil || it.Owner != nil || len(it.Labels) != 0 || it.Sprint != nil {
		t.Fatalf("bulk clears did not persist: %+v %v", it, err)
	}
	for _, form := range []url.Values{
		{"operation": {"assign"}, "value": {"tester"}},
		{"ids": {selected.ID}, "operation": {"unsupported"}},
		{"ids": {selected.ID}, "operation": {"move"}},
		{"ids": strings.Fields(strings.Repeat(selected.ID+" ", 101)), "operation": {"assign"}, "value": {"tester"}},
	} {
		if res := d.submit("bulk", form); res.Code != http.StatusBadRequest {
			t.Errorf("invalid bulk form returned %d", res.Code)
		}
	}
}

func TestBulkResolvesAliasesAndUnknownRefsBeforeRenumbering(t *testing.T) {
	d := newDashboardTest(t, Options{})
	first := d.create(t, core.CreateOpts{Title: "First selected"})
	second := d.create(t, core.CreateOpts{Title: "Second selected"})
	unselected := d.create(t, core.CreateOpts{Title: "Unselected"})
	for _, key := range []string{"ALT", "OPS"} {
		requireAction(t, d, "board-create", url.Values{"key": {key}, "name": {key}})
	}
	cfg, err := d.store.Config()
	if err != nil {
		t.Fatal(err)
	}
	moved, err := d.store.BoardMove(cfg, first.ID, "ALT")
	if err != nil {
		t.Fatal(err)
	}
	s := &server{store: d.store}
	results, err := s.bulk(cfg, url.Values{
		"ids":       {first.Number, moved.To, first.ID, second.Number, "OPS-1"},
		"operation": {"board-move"},
		"value":     {"OPS"},
	})
	if err != nil || len(results) != 3 {
		t.Fatalf("alias deduplication: %+v %v", results, err)
	}
	if results[0].Ref != first.ID || results[0].Number != "OPS-1" || results[0].Error != "" {
		t.Fatalf("first selected item not moved exactly once: %+v", results[0])
	}
	if results[1].Ref != second.ID || results[1].Number != "OPS-2" || results[1].Error != "" {
		t.Fatalf("second selected item not moved: %+v", results[1])
	}
	if results[2].Ref != "OPS-1" || results[2].Error == "" {
		t.Fatalf("initially unknown reference was resolved after renumbering: %+v", results[2])
	}
	batch, err := d.store.BeginBatch(cfg)
	if err != nil {
		t.Fatalf("bulk retained its resolution lock: %v", err)
	}
	defer batch.Close()
	other, err := batch.Resolve(unselected.ID)
	if err != nil || other.Number != unselected.Number {
		t.Fatalf("bulk changed an unselected item: %+v %v", other, err)
	}
}

func TestBulkResolutionReleasesLockWithoutMatches(t *testing.T) {
	d := newDashboardTest(t, Options{})
	s := &server{store: d.store}
	for _, refs := range [][]string{{"WEB-99999", "WEB-99999"}, {" ", ""}} {
		_, _ = s.bulk(d.cfg, url.Values{"ids": refs, "operation": {"assign"}})
		batch, err := d.store.BeginBatch(d.cfg)
		if err != nil {
			t.Fatalf("resolution retained its lock for refs %q: %v", refs, err)
		}
		batch.Close()
	}
}
