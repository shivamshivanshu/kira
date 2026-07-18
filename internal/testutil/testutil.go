package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/shivamshivanshu/kira/internal/gitx"
)

func GitInit(directory string) error {
	applyHermeticEnvironment()
	repo := gitx.Repo{Dir: directory}
	for _, arguments := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "tester"},
	} {
		if _, err := repo.Output(arguments...); err != nil {
			return err
		}
	}
	return nil
}

var (
	binaryOnce  sync.Once
	binaryPath  string
	binaryError error
)

func HermeticEnvironment() []string {
	return []string{
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"VISUAL=",
		"EDITOR=true",
		"XDG_CONFIG_HOME=" + filepath.Join(os.TempDir(), fmt.Sprintf("kira-testutil-xdg-%d", os.Getpid())),
	}
}

func ApplyHermeticEnvironment() {
	setHermeticEnvironment(false)
}

func KiraBinary(test testing.TB) string {
	if configuredPath := os.Getenv("KIRA_BIN"); configuredPath != "" {
		return configuredPath
	}
	binaryOnce.Do(func() {
		binaryDirectory, err := os.MkdirTemp("", "kira-bin-")
		if err != nil {
			binaryError = err
			return
		}
		binaryPath = filepath.Join(binaryDirectory, "kira")
		moduleDirectory := moduleRoot()
		cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/kira")
		cmd.Dir = moduleDirectory
		cmd.Env = append(os.Environ(), HermeticEnvironment()...)
		if output, err := cmd.CombinedOutput(); err != nil {
			binaryError = fmt.Errorf("build kira: %v\n%s", err, output)
		}
	})
	if binaryError != nil {
		if test != nil {
			test.Fatal(binaryError)
		}
		panic(binaryError)
	}
	return binaryPath
}

func moduleRoot() string {
	output, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "."
	}
	return filepath.Dir(strings.TrimSpace(string(output)))
}

func GitCountingShim(test testing.TB) (directory, counterPath string) {
	test.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		test.Fatalf("locate git: %v", err)
	}
	directory = test.TempDir()
	counterPath = filepath.Join(directory, "count")
	script := "#!/bin/sh\nprintf 'x\\n' >> '" + counterPath + "'\nexec '" + gitPath + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(directory, "git"), []byte(script), 0o755); err != nil {
		test.Fatal(err)
	}
	return directory, counterPath
}

func CountSpawns(test testing.TB, counterPath string) int {
	test.Helper()
	data, err := os.ReadFile(counterPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		test.Fatal(err)
	}
	return bytes.Count(data, []byte{'\n'})
}

func InitializeRepository(test testing.TB, directory string) {
	test.Helper()
	if err := GitInit(directory); err != nil {
		test.Fatalf("git init: %v", err)
	}
}

func TemporaryGitRepository(test testing.TB) string {
	test.Helper()
	directory := test.TempDir()
	InitializeRepository(test, directory)
	return directory
}

func applyHermeticEnvironment() {
	setHermeticEnvironment(true)
}

func setHermeticEnvironment(preserveExistingValues bool) {
	for _, environmentEntry := range HermeticEnvironment() {
		key, environmentValue, ok := strings.Cut(environmentEntry, "=")
		if ok {
			if preserveExistingValues {
				if _, exists := os.LookupEnv(key); exists {
					continue
				}
			}
			_ = os.Setenv(key, environmentValue)
		}
	}
}

func InitGitRepo(t *testing.T) string {
	return TemporaryGitRepository(t)
}
