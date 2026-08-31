package automation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func trustCfg() *datamodel.Config {
	return &datamodel.Config{Automation: []datamodel.AutomationHook{{Name: "h", On: "item.created", Run: "true"}}}
}

func TestGrantAndTrustRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := trustCfg()

	if Trusted(dir, cfg) {
		t.Fatal("an ungranted dir must not be trusted")
	}
	if _, err := Grant(dir, cfg); err != nil {
		t.Fatalf("Grant: %v", err)
	}
	if !Trusted(dir, cfg) {
		t.Fatal("Grant must make the same config trusted")
	}

	cfg.Automation[0].Run = "rm -rf /"
	if Trusted(dir, cfg) {
		t.Fatal("editing a hook must revoke trust")
	}
}

// An empty dir means the git dir could not be resolved. Trust must fail closed
// rather than resolving the marker against the process cwd, where a hostile
// repo's checkout could supply it (KIRA-108).
func TestEmptyDirIsNeverTrusted(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	if err := os.WriteFile(filepath.Join(cwd, trustFile), []byte(Hash(trustCfg())), 0o600); err != nil {
		t.Fatal(err)
	}

	if Trusted("", trustCfg()) {
		t.Fatal("an unresolvable trust dir must never be trusted")
	}
	if _, err := Grant("", trustCfg()); err == nil {
		t.Fatal("Grant into an unresolvable dir must fail")
	}
}
