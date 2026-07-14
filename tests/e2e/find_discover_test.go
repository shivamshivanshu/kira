package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

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
	for _, tool := range []string{"git", "sh", "true"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			return err
		}
		if err := os.Symlink(path, filepath.Join(bin, tool)); err != nil {
			return err
		}
	}
	// testscript runs `kira` by re-execing the test binary via a PATH symlink;
	// a script that sets PATH=$CLEANBIN would lose it, so mirror that symlink
	// here. rg/fzf are deliberately omitted to simulate their absence.
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
