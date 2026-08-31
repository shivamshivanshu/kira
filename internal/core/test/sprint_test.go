package core_test

import (
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func TestSprintCloseReportsPointerRemovalAsEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix permission bits only")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permission bits")
	}
	t.Parallel()
	s, cfg := newStore(t)
	cfg.Sprints = []datamodel.Sprint{{Key: "2026-S14", Name: "Sprint 14", Start: "2026-07-13", End: "2026-07-26"}}
	if _, err := s.SprintActivate(cfg, "2026-S14"); err != nil {
		t.Fatalf("SprintActivate: %v", err)
	}
	cache := storage.New(s.Root()).CacheDir()
	if err := os.Chmod(cache, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(cache, 0o755) })

	_, err := s.SprintClose(cfg, "2026-S14", "")
	var e *errx.Error
	if !errors.As(err, &e) {
		t.Fatalf("want *errx.Error, got %v (%T)", err, err)
	}
	if e.Code != errx.ExitEnv {
		t.Fatalf("exit code = %d, want %d (ExitEnv): a cache write failure is not the user's input", e.Code, errx.ExitEnv)
	}
}
