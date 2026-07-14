package core

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCommitKiraPackagesDirtyChanges(t *testing.T) {
	s, cfg, repo := stagedFixture(t)
	if _, err := s.fs().WriteItem(eventTicket()); err != nil {
		t.Fatalf("write item: %v", err)
	}

	res, err := s.CommitKira(cfg)
	if err != nil {
		t.Fatalf("CommitKira: %v", err)
	}
	if !res.Committed || res.Files != 1 {
		t.Errorf("result = committed=%v files=%d, want committed=true files=1", res.Committed, res.Files)
	}
	if len(res.Items) != 1 || res.Items[0] != "KIRA-1" {
		t.Errorf("items = %v, want [KIRA-1]", res.Items)
	}
	if res.Subject != "kira: update 1 items" {
		t.Errorf("subject = %q", res.Subject)
	}

	msg, err := repo.Output("log", "-1", "--format=%B")
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	for _, want := range []string{"kira: update 1 items", "KIRA-1 T", "Kira-Ticket: KIRA-1"} {
		if !strings.Contains(msg, want) {
			t.Errorf("commit message missing %q:\n%s", want, msg)
		}
	}

	if dirty, _ := repo.DirtyPaths(".kira"); len(dirty) != 0 {
		t.Errorf("tree still dirty after commit: %v", dirty)
	}
	if _, err := s.CommitKira(cfg); err == nil {
		t.Error("CommitKira on a clean tree must refuse")
	}
}

func TestCommitKiraLeavesForeignStagedContentAlone(t *testing.T) {
	s, cfg, repo := stagedFixture(t)
	foreign := filepath.Join(repo.Dir, "src.txt")
	if err := os.WriteFile(foreign, []byte("code\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := repo.Stage("src.txt"); err != nil {
		t.Fatalf("stage foreign: %v", err)
	}
	if _, err := s.fs().WriteItem(eventTicket()); err != nil {
		t.Fatalf("write item: %v", err)
	}

	if _, err := s.CommitKira(cfg); err != nil {
		t.Fatalf("CommitKira: %v", err)
	}
	committed, err := repo.Output("show", "--name-only", "--format=", "HEAD")
	if err != nil {
		t.Fatalf("git show: %v", err)
	}
	if strings.Contains(committed, "src.txt") {
		t.Errorf("foreign staged file swept into the kira commit:\n%s", committed)
	}
	if !strings.Contains(committed, ".kira/tickets/") {
		t.Errorf("kira item missing from the commit:\n%s", committed)
	}
	staged, err := repo.StagedPaths()
	if err != nil {
		t.Fatalf("staged paths: %v", err)
	}
	if !slices.Contains(staged, "src.txt") {
		t.Errorf("foreign file no longer staged after kira commit: %v", staged)
	}
}

func TestRenderCommitSubject(t *testing.T) {
	cases := []struct {
		template string
		count    int
		numbers  []string
		want     string
	}{
		{"kira: update {count} items", 2, []string{"KIRA-1", "KIRA-2"}, "kira: update 2 items"},
		{"tickets: {numbers}", 2, []string{"KIRA-1", "KIRA-2"}, "tickets: KIRA-1 KIRA-2"},
		{"{numbers}", 7, []string{"A-1", "A-2", "A-3", "A-4", "A-5", "A-6", "A-7"}, "A-1 A-2 A-3 A-4 A-5 +2 more"},
		{"no placeholders", 1, nil, "no placeholders"},
	}
	for _, c := range cases {
		if got := renderCommitSubject(c.template, c.count, c.numbers); got != c.want {
			t.Errorf("renderCommitSubject(%q) = %q, want %q", c.template, got, c.want)
		}
	}
}

func TestCommitTrailersCapped(t *testing.T) {
	numbers := make([]string, commitTrailerCap+5)
	for i := range numbers {
		numbers[i] = "KIRA-1"
	}
	got := commitTrailers("Kira-Ticket", numbers)
	if n := strings.Count(got, "Kira-Ticket:"); n != commitTrailerCap {
		t.Errorf("trailer count = %d, want %d", n, commitTrailerCap)
	}
	if commitTrailers("Kira-Ticket", nil) != "" {
		t.Error("no items must yield no trailer block")
	}
}
