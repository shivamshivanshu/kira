package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// TestFindDiscover exercises find/discover on both the accelerated (rg/fzf) and
// degraded (pure-Go / plain-list) paths. Setup builds a "clean" bin holding
// only git and a no-op editor, so a script can point PATH at it to simulate rg
// and fzf being absent regardless of what the host has installed.
func TestFindDiscover(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:   "testdata/finddiscover",
		Setup: setupCleanBin,
	})
}

func setupCleanBin(env *testscript.Env) error {
	bin := filepath.Join(env.WorkDir, "cleanbin")
	if err := os.MkdirAll(bin, 0o777); err != nil {
		return err
	}
	// git is a hard dependency; true stands in for $EDITOR on the edit path.
	for _, tool := range []string{"git", "true"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			return err
		}
		if err := os.Symlink(path, filepath.Join(bin, tool)); err != nil {
			return err
		}
	}
	// testscript runs `kira` by re-execing the test binary through a symlink on
	// PATH; a script that sets PATH=$CLEANBIN would otherwise lose it, so mirror
	// that symlink here. rg/fzf are deliberately omitted to simulate absence.
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.Symlink(self, filepath.Join(bin, "kira")); err != nil {
		return err
	}
	env.Setenv("CLEANBIN", bin)
	return nil
}
