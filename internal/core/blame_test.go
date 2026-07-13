package core

import (
	"os"
	"os/exec"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func gitRun(t *testing.T, s *Store, date string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = s.root
	env := append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e.c",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e.c", "TZ=UTC")
	if date != "" {
		env = append(env, "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	}
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func blameField(res *datamodel.BlameResult, field string) *datamodel.BlameField {
	for i := range res.Fields {
		if res.Fields[i].Field == field {
			return &res.Fields[i]
		}
	}
	return nil
}

func TestBlameCreatedThenCommitSource(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	commitState(t, s, it, "IN_PROGRESS", "2026-01-06")

	res, err := s.Blame(cfg, it.ID)
	if err != nil {
		t.Fatal(err)
	}
	title := blameField(res, "title")
	if title == nil || title.SourceKind != sourceCreated || title.Degraded {
		t.Errorf("title = %+v, want created, not degraded", title)
	}
	state := blameField(res, "state")
	if state == nil || state.Value != "IN_PROGRESS" || state.SourceKind != sourceCommit || state.Degraded {
		t.Errorf("state = %+v, want IN_PROGRESS via commit", state)
	}
}

func TestBlameNullFieldOmittedUnlessEvent(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()

	commit := func(date string) {
		t.Helper()
		it.Updated = date
		if _, err := s.writeItem(it); err != nil {
			t.Fatal(err)
		}
		gitRun(t, s, date, "add", "-A")
		gitRun(t, s, date, "commit", "-m", "change")
	}

	commit("2026-01-05T10:00:00Z")
	res, err := s.Blame(cfg, it.ID)
	if err != nil {
		t.Fatal(err)
	}
	if f := blameField(res, "epic"); f != nil {
		t.Errorf("never-set epic emitted a row: %+v", f)
	}

	epic := "01HZZ0EPIC0000000000000000"
	it.Epic = &epic
	commit("2026-01-06T10:00:00Z")
	it.Epic = nil
	commit("2026-01-07T10:00:00Z")

	res, err = s.Blame(cfg, it.ID)
	if err != nil {
		t.Fatal(err)
	}
	f := blameField(res, "epic")
	if f == nil || f.Value != "null" || f.SourceKind != sourceCommit {
		t.Errorf("set-then-cleared epic = %+v, want null via commit", f)
	}
}

func TestBlameMergeLossIsSyntheticDegraded(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	commitState(t, s, it, "IN_PROGRESS", "2026-01-06")

	gitRun(t, s, "2026-01-06T10:00:00Z", "checkout", "-b", "side", "HEAD~1")
	gitRun(t, s, "2026-01-07T10:00:00Z", "commit", "--allow-empty", "-m", "side work")
	gitRun(t, s, "2026-01-08T10:00:00Z", "checkout", "-")
	gitRun(t, s, "2026-01-08T10:00:00Z", "merge", "--no-ff", "--no-edit", "side")

	it.State = "DONE"
	if _, err := s.writeItem(it); err != nil {
		t.Fatal(err)
	}
	gitRun(t, s, "2026-01-08T10:00:00Z", "add", "-A")
	gitRun(t, s, "2026-01-08T10:00:00Z", "commit", "--amend", "--no-edit")

	res, err := s.Blame(cfg, it.ID)
	if err != nil {
		t.Fatal(err)
	}
	state := blameField(res, "state")
	if state == nil || state.Value != "DONE" || state.SourceKind != sourceSynthetic || !state.Degraded {
		t.Errorf("state = %+v, want DONE flagged synthetic+degraded (merge-loss)", state)
	}
}
