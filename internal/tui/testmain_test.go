package tui

import (
	"os"
	"testing"

	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestMain(main *testing.M) {
	testutil.ApplyHermeticEnvironment()
	os.Exit(main.Run())
}
