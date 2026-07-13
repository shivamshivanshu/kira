package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const terminalRestore = "\x1b[?1049l\x1b[?25h\x1b[?1000l\x1b[?1006l\x1b[0m"

type crashInfo struct {
	value any
	stack []byte
}

type CrashError struct {
	LogPath string
}

func (e *CrashError) Error() string { return "kira tui crashed" }

func handleCrash(root string, info crashInfo, stderr io.Writer) *CrashError {
	defer func() { _ = recover() }()
	guard(func() { fmt.Fprint(stderr, terminalRestore) })
	path := crashLogPath(root)
	guard(func() { path = writeCrashLog(path, info) })
	guard(func() { fmt.Fprintln(stderr, "kira crashed: "+firstLine(info.value)) })
	guard(func() { fmt.Fprintln(stderr, "crash log: "+path) })
	guard(func() {
		fmt.Fprintln(stderr, "your .kira data is intact — the TUI only reads it; every change commits atomically")
	})
	return &CrashError{LogPath: path}
}

func guard(f func()) {
	defer func() { _ = recover() }()
	f()
}

func crashLogPath(root string) string {
	name := "crash-" + time.Now().UTC().Format("20060102T150405Z") + ".log"
	return filepath.Join(root, ".kira", ".local", name)
}

func writeCrashLog(path string, info crashInfo) string {
	content := fmt.Sprintf("%s\n\n%s\n", firstLine(info.value), info.stack)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
		if err := os.WriteFile(path, []byte(content), 0o644); err == nil {
			return path
		}
	}
	fallback := filepath.Join(os.TempDir(), filepath.Base(path))
	if err := os.WriteFile(fallback, []byte(content), 0o644); err == nil {
		return fallback
	}
	return "(unwritable)"
}

func firstLine(v any) string {
	s := fmt.Sprint(v)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
