package core

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func hasWarning(notes []datamodel.Warning, code datamodel.WarnCode) bool {
	for _, n := range notes {
		if n.Code == code {
			return true
		}
	}
	return false
}

func findWarning(notes []datamodel.Warning, code datamodel.WarnCode) (datamodel.Warning, bool) {
	for _, n := range notes {
		if n.Code == code {
			return n, true
		}
	}
	return datamodel.Warning{}, false
}

func TestItemsIndexFallbackAndLinear(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".kira", "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := newStore(dir)
	cfg := config.Default()
	cfg.Project.Key = "KIRA"
	it := eventTicket()
	if _, err := s.fs().WriteItem(it); err != nil {
		t.Fatalf("writeItem: %v", err)
	}

	fallback, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		t.Fatalf("items useIndex on a non-git store: %v", err)
	}
	if len(fallback.items) != 1 || fallback.items[0].ID != it.ID {
		t.Fatalf("fallback items = %v, want the one written item", fallback.items)
	}
	warn, ok := findWarning(fallback.notes, datamodel.WarnIndexFallback)
	if !ok {
		t.Fatalf("index unavailable must surface a fallback warning, got %v", fallback.notes)
	}
	if len(warn.Args) == 0 || warn.Args[0] == "" {
		t.Fatalf("fallback warning must carry the swallowed index error, got %+v", warn)
	}

	linear, err := s.read(cfg, loadOpts{useIndex: false})
	if err != nil {
		t.Fatalf("items linear: %v", err)
	}
	if len(linear.items) != 1 || linear.items[0].ID != it.ID {
		t.Fatalf("linear items = %v, want the one written item", linear.items)
	}
	if len(linear.notes) != 0 {
		t.Fatalf("linear load must not warn, got %v", linear.notes)
	}
}

func TestReadPathsAgreeOnCleanTree(t *testing.T) {
	s := eventRepo(t)
	if err := os.WriteFile(filepath.Join(s.Root(), ".kira", "config.yaml"), []byte(seedConfigYAML(t)), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Project.Key = "KIRA"
	owner, priority, due, estimate := "ada", "P1", "2026-02-01", 3.0
	detailed := &datamodel.Item{
		ID: "01HZZ0TEST0000000000000001", Number: "KIRA-2", Type: "ticket", Title: "Detailed",
		State: "IN_PROGRESS", Owner: &owner, Priority: &priority, Due: &due, Estimate: &estimate,
		Labels: []string{"b", "a"}, BlockedBy: []string{"01HZZ0TEST0000000000000000"},
		Created: "2026-01-04T10:00:00+05:30", Updated: "2026-01-06T10:00:00+05:30", Body: "body\n",
	}
	if _, err := s.fs().WriteItem(detailed); err != nil {
		t.Fatal(err)
	}
	commitState(t, s, eventTicket(), "TODO", "2026-01-05")

	paths := map[string]loadOpts{"index": {useIndex: true}, "fallback": {}, "treeish": {at: "HEAD"}}
	byPath := map[string][]datamodel.Item{}
	for name, opts := range paths {
		ld, err := s.read(cfg, opts)
		if err != nil {
			t.Fatalf("%s read: %v", name, err)
		}
		if hasWarning(ld.notes, datamodel.WarnIndexFallback) {
			t.Fatalf("%s read fell back off the index: %v", name, ld.notes)
		}
		byPath[name] = comparableItems(ld.items)
	}
	for _, name := range []string{"fallback", "treeish"} {
		if diff := firstItemDifference(byPath["index"], byPath[name]); diff != "" {
			t.Errorf("index and %s reads disagree: %s", name, diff)
		}
	}
}

func comparableItems(items []*datamodel.Item) []datamodel.Item {
	out := make([]datamodel.Item, 0, len(items))
	for _, it := range items {
		c := *it
		c.Body = ""
		c.Aliases = nilIfEmpty(c.Aliases)
		c.Labels = nilIfEmpty(c.Labels)
		c.BlockedBy = nilIfEmpty(c.BlockedBy)
		if len(c.Links) == 0 {
			c.Links = nil
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b datamodel.Item) int { return strings.Compare(a.ID, b.ID) })
	return out
}

func nilIfEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return values
}

func firstItemDifference(want, got []datamodel.Item) string {
	if len(want) != len(got) {
		return fmt.Sprintf("%d items vs %d", len(want), len(got))
	}
	for i := range want {
		wv, gv := reflect.ValueOf(want[i]), reflect.ValueOf(got[i])
		for f := range wv.NumField() {
			if !reflect.DeepEqual(wv.Field(f).Interface(), gv.Field(f).Interface()) {
				return fmt.Sprintf("%s field %s: %#v vs %#v", want[i].ID, wv.Type().Field(f).Name,
					wv.Field(f).Interface(), gv.Field(f).Interface())
			}
		}
	}
	return ""
}

func TestItemsIndexHitNoFallback(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")

	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		t.Fatalf("items useIndex on a healthy repo: %v", err)
	}
	if len(ld.items) != 1 || ld.items[0].ID != it.ID {
		t.Fatalf("index items = %v, want the committed item", ld.items)
	}
	if hasWarning(ld.notes, datamodel.WarnIndexFallback) {
		t.Fatalf("a healthy index must not fall back, got %v", ld.notes)
	}
}
