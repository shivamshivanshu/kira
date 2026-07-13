package tui

import (
	"os/exec"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@e.c")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@e.c")
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return dir
}

func initRepo(t *testing.T) (*core.Store, *datamodel.Config, string) {
	t.Helper()
	dir := gitRepo(t)
	if _, err := core.Init(dir, "KIRA", false); err != nil {
		t.Fatalf("core.Init: %v", err)
	}
	s, err := core.Discover(dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	return s, cfg, dir
}

func createTicket(t *testing.T, s *core.Store, cfg *datamodel.Config, title string) string {
	t.Helper()
	res, err := s.Create(cfg, core.CreateOpts{Type: "ticket", Title: title, NoEdit: true})
	if err != nil {
		t.Fatalf("create %q: %v", title, err)
	}
	return res.Number
}
