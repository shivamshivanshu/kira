package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitNeutralizesUserConfigTier(t *testing.T) {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		t.Fatal("XDG_CONFIG_HOME must be set by init() to neutralize the user config tier")
	}
	if _, err := os.Stat(filepath.Join(xdg, "kira")); !os.IsNotExist(err) {
		t.Fatalf("XDG_CONFIG_HOME/kira must not exist, got err=%v", err)
	}
}
