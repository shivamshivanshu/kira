package core

import (
	"os"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/hooks"
)

func installedHookPath(t *testing.T, s *Store, name string) string {
	t.Helper()
	dst, err := s.gitHookPath(s.repo(), name)
	if err != nil {
		t.Fatalf("hook path: %v", err)
	}
	return dst
}

func TestInstallHooksRepairsMangledShim(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	if _, err := s.InstallHooks(cfg, HooksInstallOpts{}); err != nil {
		t.Fatalf("install: %v", err)
	}
	dst := installedHookPath(t, s, "post-merge")
	script, _ := hooks.Script("post-merge")

	for name, mangled := range map[string]string{
		"missing guard": "#!/bin/sh\nexec kira hooks run post-merge \"$@\"\n",
		"crlf":          strings.ReplaceAll(script, "\n", "\r\n"),
	} {
		if err := os.WriteFile(dst, []byte(mangled), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := s.InstallHooks(cfg, HooksInstallOpts{}); err != nil {
			t.Fatalf("reinstall over %s shim: %v", name, err)
		}
		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != script {
			t.Errorf("%s shim not repaired:\n%q", name, got)
		}
	}
}

func TestInstallHooksRefusesHandRolledKiraHook(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	dst := installedHookPath(t, s, "post-merge")
	handRolled := "#!/bin/sh\nmy-linter --staged\nexec kira hooks run post-merge \"$@\"\n"
	if err := os.WriteFile(dst, []byte(handRolled), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := s.InstallHooks(cfg, HooksInstallOpts{})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	var status *datamodel.HookStatus
	for i := range res.Hooks {
		if res.Hooks[i].Name == "post-merge" {
			status = &res.Hooks[i]
		}
	}
	if status == nil {
		t.Fatal("no post-merge status in install result")
	}
	if status.Installed || status.Chained {
		t.Errorf("hand-rolled kira hook must be refused, got installed=%v chained=%v", status.Installed, status.Chained)
	}
	if status.Note == "" {
		t.Error("refusal must carry a note explaining the fix")
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != handRolled {
		t.Errorf("hand-rolled hook modified:\n%q", got)
	}
}

func TestUninstallHooksLeavesHandRolledKiraHook(t *testing.T) {
	s, _, _ := stagedFixture(t)
	dst := installedHookPath(t, s, "post-merge")
	handRolled := "#!/bin/sh\nmy-linter --staged\nexec kira hooks run post-merge \"$@\"\n"
	if err := os.WriteFile(dst, []byte(handRolled), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := s.UninstallHooks()
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	for _, h := range res.Hooks {
		if h.Name == "post-merge" {
			if h.State != "left" || h.Note == "" {
				t.Errorf("hand-rolled hook: state=%q note=%q, want left with warning", h.State, h.Note)
			}
		}
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != handRolled {
		t.Errorf("hand-rolled hook modified on uninstall:\n%q", got)
	}
}

func TestInstallHooksNeverRewritesChainedUserHook(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	dst := installedHookPath(t, s, "post-merge")
	userHook := "#!/bin/sh\necho user-hook-ran\n"
	if err := os.WriteFile(dst, []byte(userHook), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := s.InstallHooks(cfg, HooksInstallOpts{}); err != nil {
		t.Fatalf("install: %v", err)
	}
	chained, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(chained), "echo user-hook-ran") {
		t.Fatalf("user content lost on chain:\n%q", chained)
	}

	if _, err := s.InstallHooks(cfg, HooksInstallOpts{}); err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	after, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(chained) {
		t.Errorf("reinstall modified a chained hook:\nbefore %q\nafter  %q", chained, after)
	}

	mangledChain := strings.Replace(string(chained), ".kira/hooks/post-merge", "edited-line", 1)
	if err := os.WriteFile(dst, []byte(mangledChain), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := s.InstallHooks(cfg, HooksInstallOpts{}); err != nil {
		t.Fatalf("reinstall over mangled chain: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != mangledChain {
		t.Errorf("install rewrote a hook containing user content:\n%q", got)
	}
}
