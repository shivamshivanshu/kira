package perf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
	fixMu    sync.Mutex
	fixtures = map[int]string{}

	tmpMu   sync.Mutex
	tmpDirs []string
)

func TestMain(m *testing.M) {
	testutil.ApplyHermeticEnvironment()
	code := m.Run()
	tmpMu.Lock()
	for _, d := range tmpDirs {
		_ = os.RemoveAll(d)
	}
	tmpMu.Unlock()
	os.Exit(code)
}

func registerTmp(directory string) {
	tmpMu.Lock()
	tmpDirs = append(tmpDirs, directory)
	tmpMu.Unlock()
}

func requirePerf(test testing.TB) {
	test.Helper()
	if os.Getenv("KIRA_PERF") == "" {
		test.Skip("perf harness: set KIRA_PERF=1 to run")
	}
	if !gitx.Installed() {
		test.Skip("git not installed")
	}
}

func kiraBinary(test testing.TB) string {
	return testutil.KiraBinary(test)
}

func fixture(test testing.TB, size int) string {
	test.Helper()
	fixMu.Lock()
	defer fixMu.Unlock()
	if p, ok := fixtures[size]; ok {
		return p
	}
	p := seedFixture(test, size)
	warm(test, kiraBinary(test), p)
	fixtures[size] = p
	return p
}

func freshFixture(test testing.TB, size int) string {
	test.Helper()
	src := fixture(test, size)
	dst, err := os.MkdirTemp("", "kira-mut-")
	if err != nil {
		test.Fatal(err)
	}
	registerTmp(dst)
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		test.Fatalf("copy fixture: %v", err)
	}
	return dst
}

func seedFixture(test testing.TB, size int) string {
	test.Helper()
	root, err := os.MkdirTemp("", fmt.Sprintf("kira-fix-%d-", size))
	if err != nil {
		test.Fatal(err)
	}
	registerTmp(root)
	if err := testutil.GitInit(root); err != nil {
		test.Fatalf("git init: %v", err)
	}
	if _, err := core.Init(root, "KIRA", false); err != nil {
		test.Fatalf("Init: %v", err)
	}
	s, err := core.Discover(root)
	if err != nil {
		test.Fatal(err)
	}
	cfg, err := s.Config()
	if err != nil {
		test.Fatal(err)
	}
	if _, err := seed.Run(root, cfg, seed.Opts{Size: size, Seed: fixtureSeed}); err != nil {
		test.Fatalf("Seed: %v", err)
	}
	return root
}

func runKira(binaryPath, repositoryPath string, extraEnv []string, args ...string) (time.Duration, []byte, error) {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = repositoryPath
	cmd.Env = append(append(os.Environ(), testutil.HermeticEnvironment()...), extraEnv...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	start := time.Now()
	err := cmd.Run()
	return time.Since(start), buf.Bytes(), err
}

func warm(test testing.TB, binaryPath, repositoryPath string) {
	test.Helper()
	for _, args := range [][]string{{"list", "--json"}, {"stats", "--json"}} {
		if _, out, err := runKira(binaryPath, repositoryPath, nil, args...); err != nil {
			test.Fatalf("warm-up %v failed: %v\n%s", args, err, out)
		}
	}
}

func gitShim(test testing.TB) (directory, counterPath string) {
	return testutil.GitCountingShim(test)
}

func spawnCount(test testing.TB, binaryPath, repositoryPath, shimDirectory, counterPath string, args ...string) int {
	test.Helper()
	_ = os.Remove(counterPath)
	env := []string{"PATH=" + shimDirectory + string(os.PathListSeparator) + os.Getenv("PATH")}
	if _, out, err := runKira(binaryPath, repositoryPath, env, args...); err != nil {
		test.Fatalf("%v: %v\n%s", args, err, out)
	}
	return testutil.CountSpawns(test, counterPath)
}
