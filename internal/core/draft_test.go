package core

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/editorx"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func TestDraftRoundTrip(t *testing.T) {
	est := 3.0
	d := draft{
		Title:        "Fix race",
		Type:         "ticket",
		Subtype:      new("bug"),
		Priority:     new("P1"),
		Rank:         new("0|m:"),
		Owner:        new("shivam"),
		Labels:       []string{"bug", "perf"},
		Sprint:       new("2026-S14"),
		Due:          new("2026-07-20"),
		Estimate:     &est,
		ExtraScalars: map[string]string{"severity": "high"},
		ExtraLists:   map[string][]string{"watchers": {"alice", "bob"}},
		Body:         "\n## Description\n\nbody text\n",
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
	if !reflect.DeepEqual(got.ExtraScalars, d.ExtraScalars) || !reflect.DeepEqual(got.ExtraLists, d.ExtraLists) {
		t.Fatalf("extras lost: scalars %+v lists %+v", got.ExtraScalars, got.ExtraLists)
	}
	if got.Body != d.Body {
		t.Fatalf("body = %q, want %q", got.Body, d.Body)
	}
}

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
	base, err := parseDraft(defaultTemplate("ticket", entityschema.Schema{}))
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
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "sh "+script)

	calls := 0
	got, err := runEditor("", editorx.Stdio{}, "initial\n", func(content string) []error {
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
	_, err := runEditor("", editorx.Stdio{}, "x", func(string) []error { return nil })
	var ce *errx.Error
	if !errors.As(err, &ce) || ce.Code != errx.ExitEnv {
		t.Fatalf("want errx.ExitEnv, got %v", err)
	}
}

func draftSchema() entityschema.Schema {
	return entityschema.Schema{Name: "ticket", Fields: []entityschema.FieldDef{
		{Name: "severity", Type: entityschema.FieldString, Placement: entityschema.PlacementFrontmatter},
		{Name: "watchers", Type: entityschema.FieldString, List: true, Placement: entityschema.PlacementFrontmatter},
		{Name: "notes", Type: entityschema.FieldMarkdown, Placement: entityschema.PlacementBody, Section: "Notes"},
	}}
}

func TestDefaultTemplateOffersDeclaredFrontmatterFields(t *testing.T) {
	tmpl := defaultTemplate("ticket", draftSchema())
	for _, want := range []string{"severity: \n", "watchers: []\n", "## Notes\n"} {
		if !strings.Contains(tmpl, want) {
			t.Fatalf("template missing %q:\n%s", want, tmpl)
		}
	}
	// the offered key must read as blank, not as the literal null an empty
	// YAML scalar parses back to
	if strings.Contains(tmpl, "severity: null") {
		t.Fatalf("offered field rendered as null:\n%s", tmpl)
	}
}

func TestDefaultTemplateSurvivesItsOwnRoundTrip(t *testing.T) {
	d, err := parseDraft(defaultTemplate("ticket", draftSchema()))
	if err != nil {
		t.Fatal(err)
	}
	if got := serializeDraft(d); strings.Contains(got, "severity: null") {
		t.Fatalf("re-serialized template rendered the offered field as null:\n%s", got)
	}
}

func TestDropUnsetDeclaredExtrasKeepsUndeclaredKeys(t *testing.T) {
	d := draft{
		ExtraScalars: map[string]string{
			"severity":    "null",
			"set":         "high",
			"from_newer":  "null",
			"from_newer2": "value",
		},
		ExtraLists: map[string][]string{"watchers": {}, "kept": {"a"}},
	}
	got := dropUnsetDeclaredExtras(d, draftSchema())
	if _, ok := got.ExtraScalars["severity"]; ok {
		t.Error("an unset declared scalar must be dropped")
	}
	if _, ok := got.ExtraLists["watchers"]; ok {
		t.Error("an unset declared list must be dropped")
	}
	if got.ExtraScalars["set"] != "high" || !reflect.DeepEqual(got.ExtraLists["kept"], []string{"a"}) {
		t.Errorf("a set value must survive: %+v", got)
	}
	// an undeclared key is a newer kira's field: never dropped, null included
	if got.ExtraScalars["from_newer"] != "null" || got.ExtraScalars["from_newer2"] != "value" {
		t.Errorf("undeclared keys must round-trip verbatim: %+v", got.ExtraScalars)
	}
}

func TestDropUnsetDoesNotMutateCaller(t *testing.T) {
	d := draft{ExtraScalars: map[string]string{"severity": "null"}}
	schema := entityschema.Schema{Name: "ticket", Fields: []entityschema.FieldDef{
		{Name: "severity", Type: entityschema.FieldString, Placement: entityschema.PlacementFrontmatter},
	}}
	_ = dropUnsetDeclaredExtras(d, schema)
	if _, ok := d.ExtraScalars["severity"]; !ok {
		t.Fatal("caller's draft was mutated through the copied map header")
	}
}
