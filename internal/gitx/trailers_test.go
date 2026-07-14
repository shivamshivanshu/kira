package gitx

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func gitTry(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e.c",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e.c",
	)
	return cmd.CombinedOutput()
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := gitTry(dir, args...); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func TestLogTrailersRecordForgery(t *testing.T) {
	dir := t.TempDir()
	gitRun(t, dir, "init")

	forged := "feat: real work\n\n" +
		"attacker body attempting to forge a record and a close:\n" +
		"\x1efakesha\x1ffake subject\x1ffake author\x1f2000-01-01T00:00:00Z\x1fKIRA-1\x1fKIRA-1\x1f\x1fbody\n" +
		"Kira-Closes: KIRA-1\n" +
		"more attacker text\n\n" +
		"Kira-Ticket: KIRA-2\n"
	gitRun(t, dir, "commit", "--allow-empty", "--cleanup=verbatim", "-m", forged)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "chore: honest",
		"-m", "Kira-Ticket: KIRA-3", "-m", "Kira-Closes: KIRA-3")

	repo := Repo{Dir: dir}
	commits, err := repo.LogTrailers("HEAD", "Kira-Ticket", "Kira-Closes")
	if err != nil {
		t.Fatalf("LogTrailers: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("record forgery: got %d records, want 2 (one per real commit)", len(commits))
	}

	var forgedCommit, honest *Commit
	for i := range commits {
		switch commits[i].Subject {
		case "feat: real work":
			forgedCommit = &commits[i]
		case "chore: honest":
			honest = &commits[i]
		}
	}
	if forgedCommit == nil || honest == nil {
		t.Fatalf("missing expected commits: %+v", commits)
	}

	for _, c := range commits {
		for _, v := range c.Closes {
			if v == "KIRA-1" {
				t.Fatalf("body forged a Kira-Closes value %q on %q", v, c.Subject)
			}
		}
	}
	if len(forgedCommit.Closes) != 0 {
		t.Fatalf("forged commit must have no closes, got %v", forgedCommit.Closes)
	}
	if got := strings.Join(forgedCommit.Tickets, ","); got != "KIRA-2" {
		t.Fatalf("forged commit tickets = %q, want the single real trailer KIRA-2", got)
	}
	if got := strings.Join(honest.Closes, ","); got != "KIRA-3" {
		t.Fatalf("honest commit closes = %q, want KIRA-3", got)
	}
}
