package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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
	got, err := Parse(readExampleConfig(t))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if want := Default(); !reflect.DeepEqual(got, want) {
		t.Errorf("parsed example does not match documented defaults\n got: %#v\nwant: %#v", got, want)
	}
}

func TestMinimalYieldsDefaults(t *testing.T) {
	got, err := Parse([]byte("version: 1\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Errorf("minimal config did not expand to the documented defaults\n got: %#v\nwant: %#v", got, Default())
	}
}

func TestParseAppliesOverrides(t *testing.T) {
	got, err := Parse([]byte("version: 1\ncommit:\n  mode: manual\nestimate:\n  unit: hours\n  hours_per_day: 6\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Commit.Mode != CommitManual {
		t.Errorf("commit.mode = %q, want manual (override did not flow from yaml)", got.Commit.Mode)
	}
	if got.Commit.Trailer != "Kira-Ticket" {
		t.Errorf("commit.trailer = %q, want default preserved alongside the override", got.Commit.Trailer)
	}
	if got.Estimate.Unit != EstimateHours || got.Estimate.HoursPerDay != 6 {
		t.Errorf("estimate = %+v, want {hours 6}", got.Estimate)
	}
}

func TestLoadReadsFromKiraDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".kira"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".kira", "config.yaml"), readExampleConfig(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := Load(t.TempDir()); err == nil {
		t.Error("Load on a repo with no config.yaml should error")
	}
}

func TestValidationRejections(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantKey string
	}{
		{
			name:    "unsupported version",
			yaml:    "version: 2\n",
			wantKey: "version",
		},
		{
			name:    "invalid id.style",
			yaml:    "version: 1\nid:\n  style: uuid\n",
			wantKey: "id.style",
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
			name:    "non-positive hours_per_day",
			yaml:    "version: 1\nestimate:\n  hours_per_day: 0\n",
			wantKey: "estimate.hours_per_day",
		},
		{
			name:    "negative hours_per_day",
			yaml:    "version: 1\nestimate:\n  hours_per_day: -3\n",
			wantKey: "estimate.hours_per_day",
		},
		{
			name:    "empty workflow",
			yaml:    "version: 1\nworkflows:\n  bad:\n    states: []\n    initial: X\n",
			wantKey: "workflows.bad.states",
		},
		{
			name:    "duplicate state",
			yaml:    "version: 1\nworkflows:\n  bad:\n    states:\n      - {key: A, category: todo}\n      - {key: A, category: done}\n    initial: A\n    transitions:\n      A: []\n",
			wantKey: "workflows.bad.states",
		},
		{
			name:    "invalid category",
			yaml:    "version: 1\nworkflows:\n  bad:\n    states:\n      - {key: A, category: sideways}\n    initial: A\n    transitions:\n      A: []\n",
			wantKey: "workflows.bad.states[A].category",
		},
		{
			name:    "initial not a defined state",
			yaml:    "version: 1\nworkflows:\n  bad:\n    states:\n      - {key: A, category: todo}\n    initial: Z\n    transitions:\n      A: []\n",
			wantKey: "workflows.bad.initial",
		},
		{
			name:    "transition from unknown state",
			yaml:    "version: 1\nworkflows:\n  bad:\n    states:\n      - {key: A, category: todo}\n    initial: A\n    transitions:\n      Z: [A]\n",
			wantKey: "workflows.bad.transitions",
		},
		{
			name:    "transition to unknown target",
			yaml:    "version: 1\nworkflows:\n  bad:\n    states:\n      - {key: A, category: todo}\n    initial: A\n    transitions:\n      A: [B]\n",
			wantKey: "workflows.bad.transitions.A",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.yaml))
			if err == nil {
				t.Fatalf("expected rejection, got nil error")
			}
			if !strings.Contains(err.Error(), tc.wantKey) {
				t.Errorf("error %q does not name key %q", err.Error(), tc.wantKey)
			}
		})
	}
}
