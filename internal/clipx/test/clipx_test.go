package clipx_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/clipx"
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
	const text = "KIRA-140"
	cases := []struct {
		name     string
		env      map[string]string
		goos     string
		tools    []string
		wantTmux bool
		wantExec string
		wantArgs []string
	}{
		{name: "linux no display", env: nil, goos: "linux", tools: []string{"xclip"}, wantExec: ""},
		{name: "tmux wraps osc52", env: map[string]string{"TMUX": "/tmp/tmux-1000/default"}, goos: "linux", wantTmux: true, wantExec: ""},
		{name: "x11 xclip", env: map[string]string{"DISPLAY": ":0"}, goos: "linux", tools: []string{"xclip"}, wantExec: "xclip", wantArgs: []string{"-selection", "clipboard"}},
		{name: "x11 falls back to xsel", env: map[string]string{"DISPLAY": ":0"}, goos: "linux", tools: []string{"xsel"}, wantExec: "xsel", wantArgs: []string{"--clipboard", "--input"}},
		{name: "x11 no tool", env: map[string]string{"DISPLAY": ":0"}, goos: "linux", tools: nil, wantExec: ""},
		{name: "wayland wl-copy", env: map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, goos: "linux", tools: []string{"wl-copy"}, wantExec: "wl-copy"},
		{name: "darwin pbcopy", env: nil, goos: "darwin", tools: []string{"pbcopy"}, wantExec: "pbcopy"},
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
			if err := cb.Copy(text); err != nil {
				t.Fatalf("Copy: %v", err)
			}

			if got := term.String(); got != clipx.OSC52(text, tc.wantTmux) {
				t.Errorf("OSC52 payload mismatch (tmux=%v)\n got %q", tc.wantTmux, got)
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
			if c.stdin != text {
				t.Errorf("stdin = %q, want %q", c.stdin, text)
			}
		})
	}
}

func TestOSC52Encoding(t *testing.T) {
	const text = "hello"
	b64 := base64.StdEncoding.EncodeToString([]byte(text))

	plain := clipx.OSC52(text, false)
	if want := "\x1b]52;c;" + b64 + "\x07"; plain != want {
		t.Fatalf("plain OSC52 = %q, want %q", plain, want)
	}

	wrapped := clipx.OSC52(text, true)
	if !strings.HasPrefix(wrapped, "\x1bPtmux;\x1b") || !strings.HasSuffix(wrapped, "\x1b\\") {
		t.Fatalf("tmux wrap missing DCS passthrough envelope: %q", wrapped)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(wrapped, "\x1bPtmux;\x1b"), "\x1b\\")
	if want := strings.ReplaceAll(plain, "\x1b", "\x1b\x1b"); inner != want {
		t.Errorf("inner ESC not doubled for tmux passthrough:\n got %q\nwant %q", inner, want)
	}
}

func TestCopyReportsTerminalWriteError(t *testing.T) {
	cb := clipx.Clipboard{
		Getenv:   envFunc(nil),
		GOOS:     "linux",
		LookPath: lookPath(),
		Term:     failWriter{},
	}
	if err := cb.Copy("x"); err == nil {
		t.Fatal("expected error when terminal write fails")
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }
