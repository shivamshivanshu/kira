package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/hooks"
)

func assertHooksStatusOK(t *testing.T, s *Store) {
	t.Helper()
	status, err := s.HooksStatus()
	if err != nil {
		t.Fatalf("hooks status: %v", err)
	}
	if !status.OK {
		t.Fatalf("hooks status not OK: %+v", status)
	}
}

func TestHooksStatusHonorsCoreHooksPath(t *testing.T) {
	s, _, repo := stagedFixture(t)
	if err := repo.SetConfig("core.hooksPath", ".githooks"); err != nil {
		t.Fatalf("set hooksPath: %v", err)
	}
	if _, err := s.InstallHooks(HooksInstallOpts{IntoHooksPath: true}); err != nil {
		t.Fatalf("install: %v", err)
	}
	for _, name := range hooks.Defaults() {
		if _, err := os.Stat(filepath.Join(s.Root(), ".githooks", name)); err != nil {
			t.Fatalf("hook %s not installed under core.hooksPath: %v", name, err)
		}
	}
	assertHooksStatusOK(t, s)
}

func TestHooksStatusHonorsLinkedWorktree(t *testing.T) {
	s, _, repo := stagedFixture(t)
	if _, err := s.InstallHooks(HooksInstallOpts{}); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := repo.Output("add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := repo.Output("commit", "-m", "snapshot", "--allow-empty"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	wt := filepath.Join(t.TempDir(), "wt")
	if _, err := repo.Output("worktree", "add", wt); err != nil {
		t.Fatalf("worktree add: %v", err)
	}
	ws, err := Discover(wt)
	if err != nil {
		t.Fatalf("discover in worktree: %v", err)
	}
	assertHooksStatusOK(t, ws)
}

func TestDoctorReportRunsCleanOnFreshStore(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	if _, err := s.InstallHooks(HooksInstallOpts{}); err != nil {
		t.Fatalf("install: %v", err)
	}
	report, err := s.DoctorReport(cfg)
	if err != nil {
		t.Fatalf("doctor report: %v", err)
	}
	if !report.OK {
		t.Fatalf("fresh store not OK: %+v", report.Findings)
	}
	for _, f := range report.Findings {
		if f.Severity != doctor.SeverityInfo {
			t.Errorf("unexpected finding on fresh store: %+v", f)
		}
	}
}
