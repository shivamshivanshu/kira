package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func readExampleConfig(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestParseExample(t *testing.T) {
	t.Parallel()
	got, err := config.Parse(readExampleConfig(t))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := config.Default()
	want.Filters = map[string]string{
		"mine-active": "owner=shivam AND category=doing",
		"blocked":     "blocked_by IS NOT EMPTY",
		"overdue":     "due<2026-07-12 AND NOT category=done",
	}
	want.Sprints = []datamodel.Sprint{
		{Key: "2026-S13", Name: "Sprint 13", Start: "2026-06-29", End: "2026-07-12"},
		{Key: "2026-S14", Name: "Sprint 14", Start: "2026-07-13", End: "2026-07-26"},
	}
	want.Labels = datamodel.Vocab{Known: []string{"bug", "feature", "perf", "tech-debt", "orderbook", "infra", "p0", "p1", "p2"}}
	want.People = datamodel.People{Known: []datamodel.Person{{Name: "shivam"}, {Name: "alice"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parsed example does not match documented defaults\n got: %#v\nwant: %#v", got, want)
	}
}

func TestMinimalYieldsDefaults(t *testing.T) {
	t.Parallel()
	got, err := config.Parse([]byte("version: 1\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual(got, config.Default()) {
		t.Errorf("minimal config did not expand to the documented defaults\n got: %#v\nwant: %#v", got, config.Default())
	}
}

func TestParseAppliesOverrides(t *testing.T) {
	t.Parallel()
	got, err := config.Parse([]byte("version: 1\ncommit:\n  mode: manual\nestimate:\n  unit: hours\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Commit.Mode != datamodel.CommitManual {
		t.Errorf("commit.mode = %q, want manual (override did not flow from yaml)", got.Commit.Mode)
	}
	if got.Estimate.Unit != datamodel.EstimateHours {
		t.Errorf("estimate.unit = %q, want hours", got.Estimate.Unit)
	}
}

func TestLoadReadsFromKiraDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".kira"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".kira", "config.yaml"), readExampleConfig(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(root); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := config.Load(t.TempDir()); err == nil {
		t.Error("Load on a repo with no config.yaml should error")
	}
}

func TestFilterSprintOverrides(t *testing.T) {
	t.Parallel()
	got, err := config.Parse([]byte("version: 1\nfilters:\n  only: \"state=TODO\"\nsprints:\n  - {key: S1, name: a, start: 2026-01-01, end: 2026-01-14}\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if want := map[string]string{"only": "state=TODO"}; !reflect.DeepEqual(got.Filters, want) {
		t.Errorf("filters = %v, want %v", got.Filters, want)
	}
	if len(got.Sprints) != 1 || got.Sprints[0].Key != "S1" || !got.HasSprint("S1") || got.HasSprint("S2") {
		t.Errorf("sprints = %v, want the single configured S1", got.Sprints)
	}
}

func TestValidationRejections(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		yaml    string
		wantKey string
	}{
		{
			name:    "unsupported version",
			yaml:    "version: 99\n",
			wantKey: "version",
		},
		{
			name:    "board key bad charset",
			yaml:    "version: 2\nboards:\n  - {key: ab-c, name: Alpha}\n",
			wantKey: "boards[0].key",
		},
		{
			name:    "board key with dash",
			yaml:    "version: 2\nboards:\n  - {key: ABC-1, name: Alpha}\n",
			wantKey: "boards[0].key",
		},
		{
			name:    "duplicate board key",
			yaml:    "version: 2\nboards:\n  - {key: ABC, name: Alpha}\n  - {key: ABC, name: Beta}\n",
			wantKey: "boards: duplicate key",
		},
		{
			name:    "two default boards",
			yaml:    "version: 2\nboards:\n  - {key: ABC, name: Alpha, default: true}\n  - {key: XYZ, name: Beta, default: true}\n",
			wantKey: "boards: at most one",
		},
		{
			name:    "board without name",
			yaml:    "version: 2\nboards:\n  - {key: ABC, name: \"\"}\n",
			wantKey: "boards[0].name",
		},
		{
			name:    "invalid commit.mode",
			yaml:    "version: 1\ncommit:\n  mode: sometimes\n",
			wantKey: "commit.mode",
		},
		{
			name:    "invalid merge.policy",
			yaml:    "version: 1\nmerge:\n  policy: smart\n",
			wantKey: "merge.policy",
		},
		{
			name:    "invalid ui.icons",
			yaml:    "version: 1\nui:\n  icons: maybe\n",
			wantKey: "ui.icons",
		},
		{
			name:    "invalid estimate.unit",
			yaml:    "version: 1\nestimate:\n  unit: bananas\n",
			wantKey: "estimate.unit",
		},
		{
			name:    "empty filter query",
			yaml:    "version: 1\nfilters:\n  blank: \"  \"\n",
			wantKey: "filters.blank",
		},
		{
			name:    "duplicate sprint key",
			yaml:    "version: 1\nsprints:\n  - {key: S1, name: a, start: 2026-01-01, end: 2026-01-14}\n  - {key: S1, name: b, start: 2026-01-15, end: 2026-01-28}\n",
			wantKey: "sprints",
		},
		{
			name:    "invalid sprint date",
			yaml:    "version: 1\nsprints:\n  - {key: S1, name: a, start: someday, end: 2026-01-14}\n",
			wantKey: "sprints[S1].start",
		},
		{
			name:    "sprint start not before end",
			yaml:    "version: 1\nsprints:\n  - {key: S1, name: a, start: 2026-01-14, end: 2026-01-14}\n",
			wantKey: "sprints[S1]",
		},
		{
			name:    "invalid ui.background",
			yaml:    "version: 1\nui:\n  background: chartreuse\n",
			wantKey: "ui.background",
		},
		{
			name:    "branch_pattern missing number token",
			yaml:    "version: 1\nworkon:\n  branch_pattern: feature/foo\n",
			wantKey: "workon.branch_pattern",
		},
		{
			name:    "branch_pattern with only key and slug",
			yaml:    "version: 1\nworkon:\n  branch_pattern: \"{key}/{slug}\"\n",
			wantKey: "workon.branch_pattern",
		},
		{
			name:    "filter with broken query grammar",
			yaml:    "version: 1\nfilters:\n  broken: \"state = \"\n",
			wantKey: "filters.broken",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := config.Parse([]byte(tc.yaml))
			if err == nil {
				t.Fatalf("expected rejection, got nil error")
			}
			if !strings.Contains(err.Error(), tc.wantKey) {
				t.Errorf("error %q does not name key %q", err.Error(), tc.wantKey)
			}
		})
	}
}
