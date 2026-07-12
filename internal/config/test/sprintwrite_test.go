package config_test

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const commentedConfig = `version: 1

project:
  key: KIRA
  name: kira

id:
  style: sequential          # sequential (default) | hash

workflows:
  ticket:
    states:
      - { key: TODO,        category: todo }
      - { key: IN_PROGRESS, category: doing, wip: 3 }   # advisory column cap
      - { key: DONE,        category: done }
    initial: TODO
    transitions:
      TODO:        [IN_PROGRESS]
      IN_PROGRESS: [DONE]
      DONE:        []

filters: {}                   # named saved queries

sprints: []                   # sprint entities (kira sprint create)

commit:
  mode: auto                  # auto | manual | prompt
`

var s15 = datamodel.Sprint{Key: "2026-S15", Name: "Sprint 15", Start: "2026-07-27", End: "2026-08-09"}
var s16 = datamodel.Sprint{Key: "2026-S16", Name: "Sprint 16", Start: "2026-08-10", End: "2026-08-23"}

func appendSprint(t *testing.T, data string, sp datamodel.Sprint) string {
	t.Helper()
	out, err := config.AppendSprint([]byte(data), sp)
	if err != nil {
		t.Fatalf("AppendSprint(%s): %v", sp.Key, err)
	}
	return string(out)
}

func TestAppendSprintEmptyFlowListPreservesComments(t *testing.T) {
	out := appendSprint(t, commentedConfig, s15)

	want := strings.Replace(commentedConfig,
		"sprints: []                   # sprint entities (kira sprint create)",
		"sprints:                    # sprint entities (kira sprint create)\n"+
			`  - { key: 2026-S15, name: Sprint 15, start: "2026-07-27", end: "2026-08-09" }`, 1)
	if out != want {
		t.Errorf("append into empty list:\n--- got ---\n%s\n--- want ---\n%s", out, want)
	}

	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if len(cfg.Sprints) != 1 || cfg.Sprints[0] != s15 {
		t.Errorf("parsed sprints = %+v, want [%+v]", cfg.Sprints, s15)
	}
}

func TestAppendSprintBlockListAppendsAfterLastEntry(t *testing.T) {
	first := appendSprint(t, commentedConfig, s15)
	out := appendSprint(t, first, s16)

	want := strings.Replace(first,
		`  - { key: 2026-S15, name: Sprint 15, start: "2026-07-27", end: "2026-08-09" }`,
		`  - { key: 2026-S15, name: Sprint 15, start: "2026-07-27", end: "2026-08-09" }`+"\n"+
			`  - { key: 2026-S16, name: Sprint 16, start: "2026-08-10", end: "2026-08-23" }`, 1)
	if out != want {
		t.Errorf("append to block list:\n--- got ---\n%s\n--- want ---\n%s", out, want)
	}

	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if len(cfg.Sprints) != 2 || cfg.Sprints[1] != s16 {
		t.Errorf("parsed sprints = %+v, want [.., %+v]", cfg.Sprints, s16)
	}
}

func TestAppendSprintBlockListMultiLineEntries(t *testing.T) {
	data := `sprints:
  - key: 2026-S14
    name: Sprint 14
    start: 2026-07-13
    end: 2026-07-26
`
	out := appendSprint(t, data, s15)
	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if len(cfg.Sprints) != 2 || cfg.Sprints[1] != s15 || cfg.Sprints[0].Key != "2026-S14" {
		t.Errorf("parsed sprints = %+v", cfg.Sprints)
	}
	if !strings.Contains(out, "  - key: 2026-S14") {
		t.Errorf("existing multi-line entry rewritten:\n%s", out)
	}
}

func TestAppendSprintInlineFlowList(t *testing.T) {
	data := "sprints: [{ key: 2026-S14, name: Sprint 14, start: 2026-07-13, end: 2026-07-26 }]\n"
	out := appendSprint(t, data, s15)
	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if len(cfg.Sprints) != 2 || cfg.Sprints[1] != s15 {
		t.Errorf("parsed sprints = %+v", cfg.Sprints)
	}
}

func TestAppendSprintMissingKeyAppendsBlock(t *testing.T) {
	data := "version: 1\nproject:\n  key: KIRA\n"
	out := appendSprint(t, data, s15)
	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if len(cfg.Sprints) != 1 || cfg.Sprints[0] != s15 {
		t.Errorf("parsed sprints = %+v", cfg.Sprints)
	}
	if !strings.HasPrefix(out, data[:len(data)-1]) {
		t.Errorf("existing content rewritten:\n%s", out)
	}
}

func TestAppendSprintNullValue(t *testing.T) {
	data := "sprints:\n"
	out := appendSprint(t, data, s15)
	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if len(cfg.Sprints) != 1 || cfg.Sprints[0] != s15 {
		t.Errorf("parsed sprints = %+v", cfg.Sprints)
	}
}

func TestAppendSprintDuplicateKeyRejected(t *testing.T) {
	first := appendSprint(t, commentedConfig, s15)
	if _, err := config.AppendSprint([]byte(first), s15); err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Errorf("duplicate append error = %v, want duplicate key", err)
	}
}

func TestAppendSprintEmptyNameRejected(t *testing.T) {
	sp := datamodel.Sprint{Key: "K", Start: "2026-01-01", End: "2026-01-02"}
	if _, err := config.AppendSprint([]byte(commentedConfig), sp); err == nil || !strings.Contains(err.Error(), "empty name") {
		t.Errorf("unnamed sprint error = %v, want empty name", err)
	}
}

func TestAppendSprintMultiLineValueRejected(t *testing.T) {
	sp := datamodel.Sprint{Key: "K", Name: "a\nb", Start: "2026-01-01", End: "2026-01-02"}
	if _, err := config.AppendSprint([]byte(commentedConfig), sp); err == nil {
		t.Error("multi-line field value accepted, want error")
	}
}
