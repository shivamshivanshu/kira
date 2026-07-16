package perf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/seed"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

const fixtureSeed = 20260713

type command struct {
	name string
	args []string
}

var commands = []command{
	{"version", []string{"version"}},
	{"list", []string{"list", "--json"}},
	{"tree", []string{"tree", "--json"}},
	{"show", []string{"show", "KIRA-1", "--json"}},
	{"query", []string{"list", "--query", "type = ticket", "--json"}},
	{"stats", []string{"stats", "--json"}},
}

var mutations = []command{
	{"create", []string{"create", "ticket", "--title", "perf", "--no-edit", "--json"}},
	{"edit", []string{"edit", "KIRA-1", "--rank", "zzz", "--json"}},
	{"move", []string{"move", "KIRA-1", "ACTIVE", "--force", "--json"}},
}

var (
	binOnce sync.Once
	binPath string
	binErr  error

	fixMu    sync.Mutex
	fixtures = map[int]string{}

	tmpMu   sync.Mutex
	tmpDirs []string
)

func TestMain(m *testing.M) {
	_ = os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	_ = os.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	_ = os.Setenv("EDITOR", "true")
	code := m.Run()
	tmpMu.Lock()
	for _, d := range tmpDirs {
		_ = os.RemoveAll(d)
	}
	tmpMu.Unlock()
	os.Exit(code)
}

func registerTmp(dir string) {
	tmpMu.Lock()
	tmpDirs = append(tmpDirs, dir)
	tmpMu.Unlock()
}

func requirePerf(tb testing.TB) {
	tb.Helper()
	if os.Getenv("KIRA_PERF") == "" {
		tb.Skip("perf harness: set KIRA_PERF=1 to run")
	}
	if !gitx.Installed() {
		tb.Skip("git not installed")
	}
}

func kiraBin(tb testing.TB) string {
	tb.Helper()
	if p := os.Getenv("KIRA_BIN"); p != "" {
		return p
	}
	binOnce.Do(func() {
		dir, err := os.MkdirTemp("", "kira-bin-")
		if err != nil {
			binErr = err
			return
		}
		registerTmp(dir)
		out := filepath.Join(dir, "kira")
		cmd := exec.Command("go", "build", "-o", out, "./cmd/kira")
		cmd.Dir = repoRoot()
		if b, err := cmd.CombinedOutput(); err != nil {
			binErr = fmt.Errorf("build kira: %v\n%s", err, b)
			return
		}
		binPath = out
	})
	if binErr != nil {
		tb.Fatal(binErr)
	}
	return binPath
}

func repoRoot() string {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "."
	}
	return filepath.Dir(strings.TrimSpace(string(out)))
}

func fixture(tb testing.TB, size int) string {
	tb.Helper()
	fixMu.Lock()
	defer fixMu.Unlock()
	if p, ok := fixtures[size]; ok {
		return p
	}
	p := seedFixture(tb, size)
	warm(tb, kiraBin(tb), p)
	fixtures[size] = p
	return p
}

func freshFixture(tb testing.TB, size int) string {
	tb.Helper()
	src := fixture(tb, size)
	dst, err := os.MkdirTemp("", "kira-mut-")
	if err != nil {
		tb.Fatal(err)
	}
	registerTmp(dst)
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		tb.Fatalf("copy fixture: %v", err)
	}
	return dst
}

func seedFixture(tb testing.TB, size int) string {
	tb.Helper()
	root, err := os.MkdirTemp("", fmt.Sprintf("kira-fix-%d-", size))
	if err != nil {
		tb.Fatal(err)
	}
	registerTmp(root)
	if err := testutil.GitInit(root); err != nil {
		tb.Fatalf("git init: %v", err)
	}
	if _, err := core.Init(root, "KIRA", false); err != nil {
		tb.Fatalf("Init: %v", err)
	}
	s, err := core.Discover(root)
	if err != nil {
		tb.Fatal(err)
	}
	cfg, err := s.Config()
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := seed.Run(root, cfg, seed.Opts{Size: size, Seed: fixtureSeed}); err != nil {
		tb.Fatalf("Seed: %v", err)
	}
	return root
}

func runKira(bin, dir string, extraEnv []string, args ...string) (time.Duration, []byte, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	if extraEnv != nil {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	start := time.Now()
	err := cmd.Run()
	return time.Since(start), buf.Bytes(), err
}

func warm(tb testing.TB, bin, dir string) {
	tb.Helper()
	for _, args := range [][]string{{"list", "--json"}, {"stats", "--json"}} {
		if _, out, err := runKira(bin, dir, nil, args...); err != nil {
			tb.Fatalf("warm-up %v failed: %v\n%s", args, err, out)
		}
	}
}

func gitShim(tb testing.TB) (dir, counter string) {
	tb.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		tb.Fatalf("locate git: %v", err)
	}
	dir, err = os.MkdirTemp("", "kira-shim-")
	if err != nil {
		tb.Fatal(err)
	}
	registerTmp(dir)
	counter = filepath.Join(dir, "count")
	script := "#!/bin/sh\nprintf 'x\\n' >> \"$KIRA_SPAWN_COUNTER\"\nexec " + gitPath + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		tb.Fatal(err)
	}
	return dir, counter
}

func spawnCount(tb testing.TB, bin, dir, shimDir, counter string, args ...string) int {
	tb.Helper()
	if err := os.Remove(counter); err != nil && !os.IsNotExist(err) {
		tb.Fatal(err)
	}
	env := []string{"PATH=" + shimDir + string(os.PathListSeparator) + os.Getenv("PATH"), "KIRA_SPAWN_COUNTER=" + counter}
	if _, out, err := runKira(bin, dir, env, args...); err != nil {
		tb.Fatalf("%v: %v\n%s", args, err, out)
	}
	data, err := os.ReadFile(counter)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		tb.Fatal(err)
	}
	return bytes.Count(data, []byte{'\n'})
}
