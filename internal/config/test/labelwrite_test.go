package config_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
)

const labelConfig = `version: 1

project:
  key: KIRA
  name: kira

labels:
  known: []                   # add project labels
  strict: false
`

func appendLabels(t *testing.T, data string, names ...string) string {
	t.Helper()
	out, err := config.AppendKnownLabels([]byte(data), names)
	if err != nil {
		t.Fatalf("AppendKnownLabels(%v): %v", names, err)
	}
	return string(out)
}

func TestAppendKnownLabelsEmptyFlowListPreservesComments(t *testing.T) {
	out := appendLabels(t, labelConfig, "backend", "frontend")

	want := `version: 1

project:
  key: KIRA
  name: kira

labels:
  known: [backend, frontend]                   # add project labels
  strict: false
`
	if out != want {
		t.Errorf("append into empty list:\n--- got ---\n%s\n--- want ---\n%s", out, want)
	}
	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if !slices.Equal(cfg.Labels.Known, []string{"backend", "frontend"}) {
		t.Errorf("parsed known = %v", cfg.Labels.Known)
	}
}

func TestAppendKnownLabelsBlockList(t *testing.T) {
	src := `labels:
  known:
    - backend
  strict: false
`
	out := appendLabels(t, src, "frontend")
	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if !slices.Equal(cfg.Labels.Known, []string{"backend", "frontend"}) {
		t.Errorf("parsed known = %v", cfg.Labels.Known)
	}
}

func TestAppendKnownLabelsNamesOwnSubsystemOnError(t *testing.T) {
	_, err := config.AppendKnownLabels([]byte(labelConfig), []string{"a\nb"})
	if err == nil {
		t.Fatal("expected an error for a multi-line label name")
	}
	if !strings.Contains(err.Error(), "labels.known") {
		t.Fatalf("error should name the labels.known subsystem: %v", err)
	}
	if strings.Contains(err.Error(), "sprints") {
		t.Fatalf("error must not mention sprints: %v", err)
	}
}

func TestAppendKnownLabelsQuotesWhenNeeded(t *testing.T) {
	out := appendLabels(t, labelConfig, "needs review")
	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if !slices.Contains(cfg.Labels.Known, "needs review") {
		t.Errorf("parsed known = %v, want it to contain %q", cfg.Labels.Known, "needs review")
	}
}
