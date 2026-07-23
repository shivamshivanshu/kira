package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/tui"
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

// A crash in --json mode must still emit valid JSON: renderError checks
// jsonMode before short-circuiting on the crash case.
func TestRenderErrorCrashInJSONMode(t *testing.T) {
	crash := &tui.CrashError{LogPath: "/repo/.kira/.local/crash-20260723T000000Z.log"}

	var buf bytes.Buffer
	if got := renderError(&buf, crash, true); got != int(errx.ExitCrash) {
		t.Errorf("exit code = %d, want %d (ExitCrash)", got, errx.ExitCrash)
	}

	out := buf.String()
	errIdx, hintIdx, codeIdx := strings.Index(out, `"error"`), strings.Index(out, `"hint"`), strings.Index(out, `"code"`)
	if errIdx < 0 || hintIdx < 0 || codeIdx < 0 || errIdx >= hintIdx || hintIdx >= codeIdx {
		t.Fatalf("expected error, hint, code fields in that order, got %s", out)
	}

	var shape struct {
		Error string `json:"error"`
		Hint  string `json:"hint"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal(buf.Bytes(), &shape); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if shape.Code != int(errx.ExitCrash) {
		t.Errorf("json code = %d, want %d", shape.Code, errx.ExitCrash)
	}
	if shape.Hint != crash.LogPath {
		t.Errorf("json hint = %q, want the crash log path %q", shape.Hint, crash.LogPath)
	}
	if shape.Error == "" {
		t.Error("json error field is empty")
	}
}
