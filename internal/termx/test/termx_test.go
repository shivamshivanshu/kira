package termx_test

import (
	"os"
	"testing"

	"github.com/shivamshivanshu/kira/internal/termx"
)

// /dev/null is a character device but not a terminal; a mode-bit check
// mistakes it for one and turns `kira </dev/null` interactive.
func TestIsTerminalDevNull(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if termx.IsTerminal(f) {
		t.Errorf("IsTerminal(%s) = true, want false", os.DevNull)
	}
}

func TestIsTerminalPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()
	if termx.IsTerminal(r) || termx.IsTerminal(w) {
		t.Error("IsTerminal(pipe) = true, want false")
	}
}

func TestReadLineDefaultConsecutivePrompts(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig; _ = r.Close() })
	if _, err := w.WriteString("y\nn\n"); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	if got := termx.ReadLineDefault("", "d"); got != "y" {
		t.Fatalf("first prompt = %q, want %q", got, "y")
	}
	if got := termx.ReadLineDefault("", "d"); got != "n" {
		t.Fatalf("second prompt = %q, want %q (buffered stdin dropped between prompts)", got, "n")
	}
}
