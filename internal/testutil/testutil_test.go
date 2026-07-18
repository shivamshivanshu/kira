package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHermeticEnvironment(t *testing.T) {
	environmentValues := make(map[string]string)
	for _, environmentEntry := range HermeticEnvironment() {
		key, environmentValue, ok := strings.Cut(environmentEntry, "=")
		if ok {
			environmentValues[key] = environmentValue
		}
	}
	if environmentValues["GIT_CONFIG_GLOBAL"] != os.DevNull {
		t.Fatalf("GIT_CONFIG_GLOBAL = %q", environmentValues["GIT_CONFIG_GLOBAL"])
	}
	if environmentValues["GIT_CONFIG_SYSTEM"] != os.DevNull {
		t.Fatalf("GIT_CONFIG_SYSTEM = %q", environmentValues["GIT_CONFIG_SYSTEM"])
	}
	if environmentValues["EDITOR"] != "true" {
		t.Fatalf("EDITOR = %q", environmentValues["EDITOR"])
	}
	if environmentValues["XDG_CONFIG_HOME"] == "" {
		t.Fatal("XDG_CONFIG_HOME is empty")
	}
	if _, err := os.Stat(filepath.Join(environmentValues["XDG_CONFIG_HOME"], "kira")); !os.IsNotExist(err) {
		t.Fatalf("XDG_CONFIG_HOME/kira must not exist, got err=%v", err)
	}
}
