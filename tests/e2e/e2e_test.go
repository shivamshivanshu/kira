package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/shivamshivanshu/kira/internal/cli"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"kira": func() { os.Exit(cli.Main()) },
	})
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Setup: func(e *testscript.Env) error {
			e.Setenv("KIRA_NOW", "2026-07-15T09:00:00Z")
			disableGitMaintenance(e)
			return nil
		},
	})
}

func disableGitMaintenance(env *testscript.Env) {
	for _, entry := range testutil.GitMaintenanceDisabled() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			env.Setenv(key, value)
		}
	}
}
