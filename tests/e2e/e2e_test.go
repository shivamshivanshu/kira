package e2e

import (
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/shivamshivanshu/kira/internal/cli"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"kira": cli.Main,
	}))
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
	})
}
