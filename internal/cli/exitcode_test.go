package cli

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/index"
)

func TestIndexOpenOnReadOnlyFSExitsEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix permission bits only")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permission bits")
	}
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(parent, 0o700) }()

	_, err := index.Open(filepath.Join(parent, "cache"))
	if err == nil {
		t.Fatal("expected an error opening the index under a read-only parent directory")
	}
	if got := renderError(io.Discard, err, false); got != int(errx.ExitEnv) {
		t.Errorf("exit code = %d, want %d (ExitEnv)", got, errx.ExitEnv)
	}
}
