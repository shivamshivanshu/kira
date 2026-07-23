package e2e

import (
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/shivamshivanshu/kira/internal/cli"
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
			return nil
		},
	})
}
