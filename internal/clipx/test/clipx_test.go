package clipx_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/clipx"
)

const (
	sampleText  = "hello"
	samplePlain = "\x1b]52;c;aGVsbG8=\x07"
	sampleTmux  = "\x1bPtmux;\x1b\x1b]52;c;aGVsbG8=\x07\x1b\\"
)

func envFunc(pairs map[string]string) func(string) string {
	return func(k string) string { return pairs[k] }
}

func lookPath(available ...string) func(string) (string, error) {
	set := map[string]bool{}
	for _, a := range available {
		set[a] = true
	}
	return func(name string) (string, error) {
		if set[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
}

type execCall struct {
	name  string
	args  []string
	stdin string
}

func TestCopyChainMatrix(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		goos     string
		tools    []string
		wantTerm string
		wantExec string
		wantArgs []string
	}{
		{name: "linux no display", env: nil, goos: "linux", tools: []string{"xclip"}, wantTerm: samplePlain, wantExec: ""},
		{name: "tmux wraps osc52", env: map[string]string{"TMUX": "/tmp/tmux-1000/default"}, goos: "linux", wantTerm: sampleTmux, wantExec: ""},
		{name: "x11 xclip", env: map[string]string{"DISPLAY": ":0"}, goos: "linux", tools: []string{"xclip"}, wantTerm: samplePlain, wantExec: "xclip", wantArgs: []string{"-selection", "clipboard"}},
		{name: "x11 falls back to xsel", env: map[string]string{"DISPLAY": ":0"}, goos: "linux", tools: []string{"xsel"}, wantTerm: samplePlain, wantExec: "xsel", wantArgs: []string{"--clipboard", "--input"}},
		{name: "x11 no tool", env: map[string]string{"DISPLAY": ":0"}, goos: "linux", tools: nil, wantTerm: samplePlain, wantExec: ""},
		{name: "wayland wl-copy", env: map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, goos: "linux", tools: []string{"wl-copy"}, wantTerm: samplePlain, wantExec: "wl-copy"},
		{name: "xwayland falls back to xclip", env: map[string]string{"WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"}, goos: "linux", tools: []string{"xclip"}, wantTerm: samplePlain, wantExec: "xclip", wantArgs: []string{"-selection", "clipboard"}},
		{name: "darwin pbcopy", env: nil, goos: "darwin", tools: []string{"pbcopy"}, wantTerm: samplePlain, wantExec: "pbcopy"},
		{name: "darwin ignores display tools", env: map[string]string{"DISPLAY": ":0"}, goos: "darwin", tools: []string{"xclip"}, wantTerm: samplePlain, wantExec: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var term bytes.Buffer
			var calls []execCall
			cb := clipx.Clipboard{
				Getenv:   envFunc(tc.env),
				GOOS:     tc.goos,
				LookPath: lookPath(tc.tools...),
				Exec: func(name string, args []string, stdin []byte) error {
					calls = append(calls, execCall{name, args, string(stdin)})
					return nil
				},
				Term: &term,
			}
			if err := cb.Copy(sampleText); err != nil {
				t.Fatalf("Copy: %v", err)
			}

			if got := term.String(); got != tc.wantTerm {
				t.Errorf("terminal bytes = %q, want %q", got, tc.wantTerm)
			}

			if tc.wantExec == "" {
				if len(calls) != 0 {
					t.Fatalf("expected no external tool, got %+v", calls)
				}
				return
			}
			if len(calls) != 1 {
				t.Fatalf("expected exactly one external call, got %+v", calls)
			}
			c := calls[0]
			if c.name != tc.wantExec {
				t.Errorf("tool = %q, want %q", c.name, tc.wantExec)
			}
			if tc.wantArgs != nil && strings.Join(c.args, " ") != strings.Join(tc.wantArgs, " ") {
				t.Errorf("args = %v, want %v", c.args, tc.wantArgs)
			}
			if c.stdin != sampleText {
				t.Errorf("stdin = %q, want %q", c.stdin, sampleText)
			}
		})
	}
}

func TestOSC52LiteralBytes(t *testing.T) {
	if got := clipx.OSC52(sampleText, false); got != samplePlain {
		t.Errorf("plain OSC52 = %q, want %q", got, samplePlain)
	}
	if got := clipx.OSC52(sampleText, true); got != sampleTmux {
		t.Errorf("tmux OSC52 = %q, want %q", got, sampleTmux)
	}
}

func TestCopyWithoutTermSkipsSequence(t *testing.T) {
	var calls []execCall
	cb := clipx.Clipboard{
		Getenv:   envFunc(map[string]string{"DISPLAY": ":0"}),
		GOOS:     "linux",
		LookPath: lookPath("xclip"),
		Exec: func(name string, args []string, stdin []byte) error {
			calls = append(calls, execCall{name, args, string(stdin)})
			return nil
		},
	}
	if err := cb.Copy(sampleText); err != nil {
		t.Fatalf("Copy with nil Term: %v", err)
	}
	if len(calls) != 1 || calls[0].name != "xclip" {
		t.Fatalf("external tool must still run without a terminal, got %+v", calls)
	}
}

func TestCopyWrapsToolFailureWithToolName(t *testing.T) {
	var term bytes.Buffer
	toolErr := errors.New("exit status 1")
	cb := clipx.Clipboard{
		Getenv:   envFunc(map[string]string{"DISPLAY": ":0"}),
		GOOS:     "linux",
		LookPath: lookPath("xclip"),
		Exec:     func(string, []string, []byte) error { return toolErr },
		Term:     &term,
	}
	err := cb.Copy(sampleText)
	if !errors.Is(err, toolErr) {
		t.Fatalf("Copy must surface the tool failure, got %v", err)
	}
	if !strings.Contains(err.Error(), "xclip: ") {
		t.Errorf("tool failure must name the tool, got %q", err.Error())
	}
	if term.String() != samplePlain {
		t.Errorf("OSC 52 must still be emitted when the tool fails, got %q", term.String())
	}
}

func TestCopyReportsTerminalWriteError(t *testing.T) {
	writeErr := errors.New("closed")
	cb := clipx.Clipboard{
		Getenv:   envFunc(nil),
		GOOS:     "linux",
		LookPath: lookPath(),
		Term:     failWriter{writeErr},
	}
	err := cb.Copy("x")
	if !errors.Is(err, writeErr) {
		t.Fatalf("Copy must surface the terminal write failure, got %v", err)
	}
	if !strings.Contains(err.Error(), "terminal: ") {
		t.Errorf("terminal failure must name the sink, got %q", err.Error())
	}
}

type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }
