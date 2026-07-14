package core

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/hooks"
)

func assertDoctorAgreesWithHooksStatus(t *testing.T, s *Store) {
	t.Helper()
	env := s.doctorEnv()
	status, err := s.HooksStatus()
	if err != nil {
		t.Fatalf("hooks status: %v", err)
	}
	if !status.OK {
		t.Fatalf("hooks status not OK: %+v", status)
	}
	for _, name := range hooks.Default {
		if !slices.Contains(env.TrackedHooks, name) {
			t.Errorf("doctor does not track hook %s: %v", name, env.TrackedHooks)
		}
		if !slices.Contains(env.InstalledHooks, name) {
			t.Errorf("doctor reports hook %s missing while hooks status says installed", name)
		}
	}
	if !env.MergeDriverRegistered || !env.TicketAttrRegistered {
		t.Errorf("doctor reports merge driver=%v attr=%v while hooks status says registered",
			env.MergeDriverRegistered, env.TicketAttrRegistered)
	}
}

func TestDoctorEnvHonorsCoreHooksPath(t *testing.T) {
	s, cfg, repo := stagedFixture(t)
	if err := repo.SetConfig("core.hooksPath", ".githooks"); err != nil {
		t.Fatalf("set hooksPath: %v", err)
	}
	if _, err := s.InstallHooks(cfg, HooksInstallOpts{IntoHooksPath: true}); err != nil {
		t.Fatalf("install: %v", err)
	}
	for _, name := range hooks.Default {
		if _, err := os.Stat(filepath.Join(s.Root(), ".githooks", name)); err != nil {
			t.Fatalf("hook %s not installed under core.hooksPath: %v", name, err)
		}
	}
	assertDoctorAgreesWithHooksStatus(t, s)
}

func TestDoctorEnvHonorsLinkedWorktree(t *testing.T) {
	s, cfg, repo := stagedFixture(t)
	if _, err := s.InstallHooks(cfg, HooksInstallOpts{}); err != nil {
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
	assertDoctorAgreesWithHooksStatus(t, ws)
}

func TestDoctorEnvDetectsPlainInvocationHook(t *testing.T) {
	s, _, repo := stagedFixture(t)
	tracked := filepath.Join(s.fs().KiraDir(), "hooks")
	if err := os.MkdirAll(tracked, 0o755); err != nil {
		t.Fatalf("mkdir tracked hooks: %v", err)
	}
	body := "#!/bin/sh\nkira hooks post-merge \"$@\"\n"
	if err := os.WriteFile(filepath.Join(tracked, "post-merge"), []byte(body), 0o755); err != nil {
		t.Fatalf("write tracked hook: %v", err)
	}
	dst := installedHookPath(t, s, "post-merge")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	if err := os.WriteFile(dst, []byte(body), 0o755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	got := s.installedHooks(repo, []string{"post-merge"})
	if !slices.Contains(got, "post-merge") {
		t.Fatalf("a plainly-installed hook invoking 'kira hooks post-merge' must be detected, got %v", got)
	}
}

func TestDoctorReportRunsCleanOnFreshStore(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	if _, err := s.InstallHooks(cfg, HooksInstallOpts{}); err != nil {
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
