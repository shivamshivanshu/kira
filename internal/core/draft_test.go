package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDraftRoundTrip(t *testing.T) {
	est := 3.0
	d := draft{
		Title:    "Fix race",
		Type:     "ticket",
		Subtype:  strPtr("bug"),
		Priority: strPtr("P1"),
		Rank:     strPtr("0|m:"),
		Owner:    strPtr("shivam"),
		Labels:   []string{"bug", "perf"},
		Sprint:   strPtr("2026-S14"),
		Due:      strPtr("2026-07-20"),
		Estimate: &est,
		Body:     "\n## Description\n\nbody text\n",
	}
	got, err := parseDraft(serializeDraft(d))
	if err != nil {
		t.Fatalf("parseDraft: %v", err)
	}
	if got.Title != d.Title || got.Type != d.Type {
		t.Fatalf("scalars lost: %+v", got)
	}
	for _, f := range []struct {
		name      string
		got, want *string
	}{
		{"subtype", got.Subtype, d.Subtype},
		{"priority", got.Priority, d.Priority},
		{"rank", got.Rank, d.Rank},
		{"owner", got.Owner, d.Owner},
		{"sprint", got.Sprint, d.Sprint},
		{"due", got.Due, d.Due},
	} {
		if f.got == nil || *f.got != *f.want {
			t.Fatalf("%s lost: got %v, want %q", f.name, f.got, *f.want)
		}
	}
	if len(got.Labels) != 2 || got.Estimate == nil || *got.Estimate != est {
		t.Fatalf("labels/estimate lost: %+v", got)
	}
	if got.Body != d.Body {
		t.Fatalf("body = %q, want %q", got.Body, d.Body)
	}
}

// TestItemFromDraftEmptyScalarsAreAbsent pins that an editor-entered empty
// value (`owner: ""`) assembles as an absent field, so the written file keeps
// the canonical no-line form for every unset optional.
func TestItemFromDraftEmptyScalarsAreAbsent(t *testing.T) {
	empty := ""
	d := draft{
		Title: "t", Type: "ticket",
		Subtype: &empty, Priority: &empty, Rank: &empty, Owner: &empty,
		Reporter: &empty, Epic: &empty, Sprint: &empty, Due: &empty,
	}
	it := itemFromDraft(d, systemFields{ulid: "X", number: "KIRA-1", typ: "ticket", state: "TODO", created: "2026-07-12T00:00:00Z"})
	for name, got := range map[string]*string{
		"subtype": it.Subtype, "priority": it.Priority, "rank": it.Rank,
		"owner": it.Owner, "reporter": it.Reporter, "epic": it.Epic,
		"sprint": it.Sprint, "due": it.Due,
	} {
		if got != nil {
			t.Errorf("%s = %q, want absent (nil) for an empty draft value", name, *got)
		}
	}
}

func TestApplyFlagsOverridesTemplate(t *testing.T) {
	base, err := parseDraft(defaultTemplate("ticket"))
	if err != nil {
		t.Fatalf("parse default template: %v", err)
	}
	got := applyFlags(base, CreateOpts{
		Type:   "ticket",
		Title:  "T",
		Owner:  "alice",
		Labels: []string{"bug"},
		Parent: "KIRA-9",
	})
	if got.Title != "T" || got.Owner == nil || *got.Owner != "alice" {
		t.Fatalf("flags not applied: %+v", got)
	}
	if got.Epic == nil || *got.Epic != "KIRA-9" {
		t.Fatalf("parent not applied: %v", got.Epic)
	}
}

func TestStripErrorBanner(t *testing.T) {
	body := "---\ntitle: x\n---\n\n## Description\n"
	banner := errorBanner([]error{errors.New("boom")})
	if got := stripErrorBanner(banner + body); got != body {
		t.Fatalf("banner not stripped:\n%q", got)
	}
	if got := stripErrorBanner(body); got != body {
		t.Fatalf("no-banner input changed: %q", got)
	}
}

// TestRunEditorRetryLoop drives runEditor with a fake editor that writes an
// invalid buffer on the first call and a valid one on the second, proving the
// parse-validate-retry loop reopens on failure and accepts the fix.
func TestRunEditorRetryLoop(t *testing.T) {
	dir := t.TempDir()
	counter := filepath.Join(dir, "counter")
	invalid := filepath.Join(dir, "invalid.txt")
	valid := filepath.Join(dir, "valid.txt")
	script := filepath.Join(dir, "editor.sh")
	if err := os.WriteFile(invalid, []byte("still bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(valid, []byte("GOOD content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "#!/bin/sh\n" +
		"n=$(cat \"$KIRA_COUNTER\" 2>/dev/null || echo 0)\n" +
		"n=$((n+1)); echo \"$n\" > \"$KIRA_COUNTER\"\n" +
		"if [ \"$n\" -eq 1 ]; then cp \"$KIRA_INVALID\" \"$1\"; else cp \"$KIRA_VALID\" \"$1\"; fi\n"
	if err := os.WriteFile(script, []byte(src), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KIRA_COUNTER", counter)
	t.Setenv("KIRA_INVALID", invalid)
	t.Setenv("KIRA_VALID", valid)
	t.Setenv("EDITOR", "sh "+script)

	calls := 0
	got, err := runEditor("initial\n", func(content string) []error {
		calls++
		if !strings.Contains(content, "GOOD") {
			return []error{errors.New("need GOOD")}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("runEditor: %v", err)
	}
	if !strings.Contains(got, "GOOD") {
		t.Fatalf("final content = %q, want it to contain GOOD", got)
	}
	if calls != 2 {
		t.Fatalf("validate called %d times, want 2 (invalid then valid)", calls)
	}
}

func TestRunEditorUnsetIsEnvError(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	_, err := runEditor("x", func(string) []error { return nil })
	var ce *Error
	if !errors.As(err, &ce) || ce.Code != ExitEnv {
		t.Fatalf("want ExitEnv, got %v", err)
	}
}
