package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

func TestValidateGraph(t *testing.T) {
	epic := &datamodel.Item{ID: "E", Number: "KIRA-1", Type: datamodel.TypeEpic}
	ticket := &datamodel.Item{ID: "T", Number: "KIRA-2", Type: datamodel.TypeTicket}
	dupLinks := func(target string) map[string][]string {
		return map[string][]string{string(datamodel.LinkDuplicateOf): {target}}
	}

	t.Run("epic parent allowed", func(t *testing.T) {
		child := &datamodel.Item{ID: "C", Number: "KIRA-3", Type: datamodel.TypeTicket, Epic: ptr.To("E")}
		if errs := validateGraph(child, []*datamodel.Item{epic, child}); len(errs) != 0 {
			t.Fatalf("epic parent must be allowed: %v", errs)
		}
	})
	t.Run("non-epic parent rejected", func(t *testing.T) {
		child := &datamodel.Item{ID: "C", Number: "KIRA-3", Type: datamodel.TypeTicket, Epic: ptr.To("T")}
		errs := validateGraph(child, []*datamodel.Item{ticket, child})
		if len(errs) != 1 || !strings.Contains(errs[0].Error(), "not an epic") {
			t.Fatalf("non-epic parent must be rejected, got %v", errs)
		}
	})
	t.Run("blocked_by cycle rejected", func(t *testing.T) {
		a := &datamodel.Item{ID: "A", Number: "KIRA-4", Type: datamodel.TypeTicket, BlockedBy: []string{"B"}}
		b := &datamodel.Item{ID: "B", Number: "KIRA-5", Type: datamodel.TypeTicket, BlockedBy: []string{"A"}}
		errs := validateGraph(a, []*datamodel.Item{a, b})
		if len(errs) != 1 || !strings.Contains(errs[0].Error(), "cycle") {
			t.Fatalf("blocked_by cycle must be rejected, got %v", errs)
		}
	})
	t.Run("blocked_by acyclic allowed", func(t *testing.T) {
		a := &datamodel.Item{ID: "A", Number: "KIRA-4", Type: datamodel.TypeTicket, BlockedBy: []string{"B"}}
		b := &datamodel.Item{ID: "B", Number: "KIRA-5", Type: datamodel.TypeTicket}
		if errs := validateGraph(a, []*datamodel.Item{a, b}); len(errs) != 0 {
			t.Fatalf("acyclic blocked_by must be allowed: %v", errs)
		}
	})
	t.Run("duplicate_of cycle rejected", func(t *testing.T) {
		a := &datamodel.Item{ID: "A", Number: "KIRA-4", Type: datamodel.TypeTicket, Links: dupLinks("B")}
		b := &datamodel.Item{ID: "B", Number: "KIRA-5", Type: datamodel.TypeTicket, Links: dupLinks("A")}
		errs := validateGraph(a, []*datamodel.Item{a, b})
		if len(errs) != 1 || !strings.Contains(errs[0].Error(), "cycle") {
			t.Fatalf("duplicate_of cycle must be rejected, got %v", errs)
		}
	})
	t.Run("symmetric relates not a cycle", func(t *testing.T) {
		rel := func(target string) map[string][]string {
			return map[string][]string{string(datamodel.LinkRelates): {target}}
		}
		a := &datamodel.Item{ID: "A", Number: "KIRA-4", Type: datamodel.TypeTicket, Links: rel("B")}
		b := &datamodel.Item{ID: "B", Number: "KIRA-5", Type: datamodel.TypeTicket, Links: rel("A")}
		if errs := validateGraph(a, []*datamodel.Item{a, b}); len(errs) != 0 {
			t.Fatalf("symmetric relates must not be treated as a cycle: %v", errs)
		}
	})
	t.Run("pre-existing cycle elsewhere ignored", func(t *testing.T) {
		a := &datamodel.Item{ID: "A", Number: "KIRA-4", Type: datamodel.TypeTicket, BlockedBy: []string{"B"}}
		b := &datamodel.Item{ID: "B", Number: "KIRA-5", Type: datamodel.TypeTicket, BlockedBy: []string{"A"}}
		other := &datamodel.Item{ID: "U", Number: "KIRA-6", Type: datamodel.TypeTicket}
		if errs := validateGraph(other, []*datamodel.Item{a, b, other}); len(errs) != 0 {
			t.Fatalf("a cycle not involving the written item must not block it: %v", errs)
		}
	})
}

func TestValidateItemVocabAndFields(t *testing.T) {
	base := datamodel.Item{ID: "X", Number: "KIRA-1", Type: datamodel.TypeTicket, Title: "t", State: "TODO"}
	cases := []struct {
		name     string
		tweak    func(*datamodel.Config)
		mutate   func(*datamodel.Item)
		force    bool
		wantErr  bool
		wantWarn bool
	}{
		{"owner-known-strict", func(c *datamodel.Config) {
			c.People = datamodel.People{Known: []datamodel.Person{{Name: "shivam"}}, Strict: true}
		}, func(it *datamodel.Item) { it.Owner = ptr.To("shivam") }, false, false, false},
		{"owner-unknown-strict", func(c *datamodel.Config) { c.People.Strict = true }, func(it *datamodel.Item) { it.Owner = ptr.To("mallory") }, false, true, false},
		{"owner-unknown-strict-force", func(c *datamodel.Config) { c.People.Strict = true }, func(it *datamodel.Item) { it.Owner = ptr.To("mallory") }, true, false, true},
		{"owner-unknown-lenient", nil, func(it *datamodel.Item) { it.Owner = ptr.To("mallory") }, false, false, true},
		{"owner-known-lenient", func(c *datamodel.Config) {
			c.People.Known = []datamodel.Person{{Name: "alice"}}
		}, func(it *datamodel.Item) { it.Owner = ptr.To("alice") }, false, false, false},
		{"subtype-known", nil, func(it *datamodel.Item) { it.Subtype = ptr.To("bug") }, false, false, false},
		{"subtype-unknown-lenient", nil, func(it *datamodel.Item) { it.Subtype = ptr.To("saga") }, false, false, true},
		{"subtype-unknown-strict", func(c *datamodel.Config) { c.Labels.Strict = true },
			func(it *datamodel.Item) { it.Subtype = ptr.To("saga") }, false, true, false},
		{"subtype-freeform-when-empty", func(c *datamodel.Config) { c.Subtypes = datamodel.EnumVocab{} },
			func(it *datamodel.Item) { it.Subtype = ptr.To("saga") }, false, false, false},
		{"priority-unknown-lenient", nil, func(it *datamodel.Item) { it.Priority = ptr.To("P9") }, false, false, true},
		{"priority-unknown-strict", func(c *datamodel.Config) { c.Labels.Strict = true },
			func(it *datamodel.Item) { it.Priority = ptr.To("P9") }, false, true, false},
		{"subtype-per-vocab-strict-without-labels-strict", func(c *datamodel.Config) { c.Subtypes.Strict = ptr.To(true) },
			func(it *datamodel.Item) { it.Subtype = ptr.To("saga") }, false, true, false},
		{"priority-per-vocab-lenient-overrides-labels-strict", func(c *datamodel.Config) { c.Labels.Strict = true; c.Priorities.Strict = ptr.To(false) },
			func(it *datamodel.Item) { it.Priority = ptr.To("P9") }, false, false, true},
		{"resolution-known", nil, func(it *datamodel.Item) { it.Resolution = ptr.To("dropped") }, false, false, false},
		{"resolution-unknown-lenient", nil, func(it *datamodel.Item) { it.Resolution = ptr.To("meh") }, false, false, true},
		{"rank-empty", nil, func(it *datamodel.Item) { it.Rank = ptr.To("") }, false, true, false},
		{"rank-freeform", nil, func(it *datamodel.Item) { it.Rank = ptr.To("0|zzz:") }, false, false, false},
		{"sprint-known", func(c *datamodel.Config) {
			c.Sprints = []datamodel.Sprint{{Key: "2026-S14", Name: "Sprint 14", Start: "2026-07-13", End: "2026-07-26"}}
		}, func(it *datamodel.Item) { it.Sprint = ptr.To("2026-S14") }, false, false, false},
		{"sprint-unknown", nil, func(it *datamodel.Item) { it.Sprint = ptr.To("2099-S1") }, false, true, false},
		{"due-valid", nil, func(it *datamodel.Item) { it.Due = ptr.To("2026-07-20") }, false, false, false},
		{"due-invalid", nil, func(it *datamodel.Item) { it.Due = ptr.To("someday") }, false, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			if tc.tweak != nil {
				tc.tweak(cfg)
			}
			it := base
			tc.mutate(&it)
			errs, warns := validateItem(cfg, &it, tc.force)
			if gotErr := len(errs) > 0; gotErr != tc.wantErr {
				t.Errorf("errs = %v, wantErr = %v", errs, tc.wantErr)
			}
			if gotWarn := len(warns) > 0; gotWarn != tc.wantWarn {
				t.Errorf("warns = %v, wantWarn = %v", warns, tc.wantWarn)
			}
		})
	}
}

func TestFieldPresentCoversMutableFields(t *testing.T) {
	estimate := 1.0
	full := &datamodel.Item{
		Title: "t", Subtype: ptr.NilIfEmpty("bug"), Resolution: ptr.NilIfEmpty("done"),
		Priority: ptr.NilIfEmpty("P1"), Rank: ptr.NilIfEmpty("0|m:"), Owner: ptr.NilIfEmpty("shivam"),
		Reporter: ptr.NilIfEmpty("alice"), Labels: []string{"x"}, Epic: ptr.NilIfEmpty("01X"),
		Sprint: ptr.NilIfEmpty("2026-S14"), Due: ptr.NilIfEmpty("2026-07-20"), Estimate: &estimate,
	}
	empty := &datamodel.Item{}
	for _, f := range datamodel.MutableFields {
		if !fieldPresent(full, f) {
			t.Errorf("fieldPresent(populated, %q) = false", f)
		}
		if fieldPresent(empty, f) {
			t.Errorf("fieldPresent(zero, %q) = true", f)
		}
	}
}

func TestValidateResolutionState(t *testing.T) {
	cfg := config.Default()
	stale := &datamodel.Item{Type: datamodel.TypeTicket, State: "TODO", Resolution: ptr.To("done")}
	done := &datamodel.Item{Type: datamodel.TypeTicket, State: "WONT_DO", Resolution: ptr.To("dropped")}

	if errs := validateResolutionState(cfg, stale, stale); len(errs) != 0 {
		t.Errorf("untouched stale item must be grandfathered, got %v", errs)
	}
	titleEdit := *stale
	titleEdit.Title = "renamed"
	if errs := validateResolutionState(cfg, stale, &titleEdit); len(errs) != 0 {
		t.Errorf("edit not touching state/resolution must be grandfathered, got %v", errs)
	}
	if errs := validateResolutionState(cfg, nil, stale); len(errs) != 1 {
		t.Errorf("newly created bad shape must be rejected, got %v", errs)
	}
	reEdit := *stale
	reEdit.Resolution = ptr.To("duplicate")
	errs := validateResolutionState(cfg, stale, &reEdit)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "done-category") {
		t.Fatalf("re-writing resolution on a non-done state must be rejected, got %v", errs)
	}
	var e *errx.Error
	if !errors.As(errs[0], &e) || !strings.Contains(e.Hint, "resolution=") {
		t.Fatalf("rejection must hint the repair, got %v", errs[0])
	}
	if errs := validateResolutionState(cfg, stale, done); len(errs) != 0 {
		t.Errorf("resolution on a done state is valid, got %v", errs)
	}
}
