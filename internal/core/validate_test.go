package core

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func strPtr(s string) *string { return &s }

func TestVocabStrictWarn(t *testing.T) {
	base := datamodel.Item{ID: "X", Number: "KIRA-1", Type: datamodel.TypeTicket, Title: "t", State: "TODO"}
	cases := []struct {
		name       string
		strict     bool
		owner      string
		force      bool
		wantErr    bool
		wantWarned bool
	}{
		{"known", true, "shivam", false, false, false},
		{"unknown-strict", true, "mallory", false, true, false},
		{"unknown-strict-force", true, "mallory", true, false, true},
		{"unknown-lenient", false, "mallory", false, false, true},
		{"known-lenient", false, "alice", false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.People.Strict = tc.strict
			it := base
			it.Owner = &tc.owner
			errs, warns := validateItem(cfg, &it, tc.force)
			if gotErr := len(errs) > 0; gotErr != tc.wantErr {
				t.Errorf("errs = %v, wantErr = %v", errs, tc.wantErr)
			}
			if gotWarn := len(warns) > 0; gotWarn != tc.wantWarned {
				t.Errorf("warns = %v, wantWarned = %v", warns, tc.wantWarned)
			}
		})
	}
}

func TestParityFieldValidation(t *testing.T) {
	base := datamodel.Item{ID: "X", Number: "KIRA-1", Type: datamodel.TypeTicket, Title: "t", State: "TODO"}
	cases := []struct {
		name     string
		tweak    func(*datamodel.Config)
		mutate   func(*datamodel.Item)
		wantErr  bool
		wantWarn bool
	}{
		{"subtype-known", nil, func(it *datamodel.Item) { it.Subtype = strPtr("bug") }, false, false},
		{"subtype-unknown-lenient", nil, func(it *datamodel.Item) { it.Subtype = strPtr("saga") }, false, true},
		{"subtype-unknown-strict", func(c *datamodel.Config) { c.Labels.Strict = true },
			func(it *datamodel.Item) { it.Subtype = strPtr("saga") }, true, false},
		{"subtype-freeform-when-empty", func(c *datamodel.Config) { c.Subtypes = nil },
			func(it *datamodel.Item) { it.Subtype = strPtr("saga") }, false, false},
		{"priority-unknown-lenient", nil, func(it *datamodel.Item) { it.Priority = strPtr("P9") }, false, true},
		{"priority-unknown-strict", func(c *datamodel.Config) { c.Labels.Strict = true },
			func(it *datamodel.Item) { it.Priority = strPtr("P9") }, true, false},
		{"resolution-known", nil, func(it *datamodel.Item) { it.Resolution = strPtr("dropped") }, false, false},
		{"resolution-unknown-lenient", nil, func(it *datamodel.Item) { it.Resolution = strPtr("meh") }, false, true},
		{"rank-empty", nil, func(it *datamodel.Item) { it.Rank = strPtr("") }, true, false},
		{"rank-freeform", nil, func(it *datamodel.Item) { it.Rank = strPtr("0|zzz:") }, false, false},
		{"sprint-known", func(c *datamodel.Config) {
			c.Sprints = []datamodel.Sprint{{Key: "2026-S14", Name: "Sprint 14", Start: "2026-07-13", End: "2026-07-26"}}
		}, func(it *datamodel.Item) { it.Sprint = strPtr("2026-S14") }, false, false},
		{"sprint-unknown", nil, func(it *datamodel.Item) { it.Sprint = strPtr("2099-S1") }, true, false},
		{"due-valid", nil, func(it *datamodel.Item) { it.Due = strPtr("2026-07-20") }, false, false},
		{"due-invalid", nil, func(it *datamodel.Item) { it.Due = strPtr("someday") }, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			if tc.tweak != nil {
				tc.tweak(cfg)
			}
			it := base
			tc.mutate(&it)
			errs, warns := validateItem(cfg, &it, false)
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
		Title: "t", Subtype: ptrOrNil("bug"), Resolution: ptrOrNil("done"),
		Priority: ptrOrNil("P1"), Rank: ptrOrNil("0|m:"), Owner: ptrOrNil("shivam"),
		Reporter: ptrOrNil("alice"), Labels: []string{"x"}, Epic: ptrOrNil("01X"),
		Sprint: ptrOrNil("2026-S14"), Due: ptrOrNil("2026-07-20"), Estimate: &estimate,
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
