package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestHookStoreUsesGitPrefixForNestedRoot(t *testing.T) {
	toplevel := testutil.InitGitRepo(t)
	nested := filepath.Join(toplevel, "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Init(nested, "KIRA", false); err != nil {
		t.Fatalf("core.Init: %v", err)
	}

	t.Chdir(toplevel)
	t.Setenv("GIT_PREFIX", "sub/")
	s, err := hookStore(&globalFlags{})
	if err != nil {
		t.Fatalf("hookStore: %v", err)
	}
	if s == nil {
		t.Fatal("hookStore returned nil; want it to find the nested .kira via GIT_PREFIX from a hook's toplevel cwd")
	}
	got, err := filepath.EvalSymlinks(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("store root = %q, want %q", got, want)
	}
}
