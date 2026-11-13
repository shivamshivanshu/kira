package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func complete(t *testing.T, dir string, args ...string) []string {
	t.Helper()
	root, _ := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"__complete", "-C", dir}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("__complete %v: %v\n%s", args, err, out.String())
	}
	var suggestions []string
	for _, line := range strings.Split(out.String(), "\n") {
		if line == "" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "Completion ended") {
			continue
		}
		suggestions = append(suggestions, strings.SplitN(line, "\t", 2)[0])
	}
	return suggestions
}

func hasSuggestion(got []string, want string) bool {
	for _, g := range got {
		if g == want {
			return true
		}
	}
	return false
}

func TestPositionalCompleters(t *testing.T) {
	dir := initFixture(t)
	s, cfg := reopen(t, dir)
	epic, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeEpic, Title: "E", NoEdit: true})
	if err != nil {
		t.Fatalf("create epic: %v", err)
	}
	ticket, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "T", NoEdit: true})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := s.SprintCreate(cfg, datamodel.Sprint{Key: "2026-S14", Name: "S14", Start: "2026-07-13", End: "2026-07-26"}); err != nil {
		t.Fatalf("sprint create: %v", err)
	}
	if _, err := s.BoardCreate(cfg, "OPS", "", ""); err != nil {
		t.Fatalf("board create: %v", err)
	}
	if _, err := s.Index(cfg, true, false); err != nil {
		t.Fatalf("index: %v", err)
	}

	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"tree", ""}, ticket.Number},
		{[]string{"stats", ""}, epic.Number},
		{[]string{"board", ""}, epic.Number},
		{[]string{"sprint", "activate", ""}, "2026-S14"},
		{[]string{"sprint", "close", ""}, "2026-S14"},
		{[]string{"board", "rename", ""}, "OPS"},
		{[]string{"board", "archive", ""}, "OPS"},
		{[]string{"board", "unarchive", ""}, "OPS"},
		{[]string{"board", "move", ticket.Number, ""}, "OPS"},
	} {
		if got := complete(t, dir, c.args...); !hasSuggestion(got, c.want) {
			t.Errorf("__complete %v = %v, want it to include %q", c.args, got, c.want)
		}
	}
}
