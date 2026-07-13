// Package hooks holds kira's tracked git-hook shim scripts and the pure logic
// for chaining kira's hooks onto pre-existing ones without clobbering them.
package hooks

import (
	"embed"
	"path"
	"strings"
)

//go:embed scripts/post-merge scripts/prepare-commit-msg scripts/pre-commit
var scriptFS embed.FS

const (
	marker    = "# kira:chain v1"
	markerEnd = "# /kira:chain"
)

// Default names the hooks installed unconditionally; PreCommit is opt-in.
var Default = []string{"post-merge", "prepare-commit-msg"}

const PreCommit = "pre-commit"

func Script(name string) (string, bool) {
	data, err := scriptFS.ReadFile("scripts/" + name)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func hasMarker(content string) bool {
	return strings.Contains(content, marker)
}

// Invokes reports whether an installed hook already delegates to kira. It is the
// single source of truth for the invocation the embedded scripts use, so
// re-install idempotency stays coupled to the script text (pinned by a test).
func Invokes(content, name string) bool {
	return strings.Contains(content, "kira hooks "+name)
}

// Classify derives an installed hook's state from its content: whether it runs
// kira at all, and whether kira is chained after a pre-existing hook.
func Classify(content, name string) (installed, chained bool) {
	chained = hasMarker(content)
	return chained || Invokes(content, name), chained
}

func IsShellScript(content string) bool {
	line, _, _ := strings.Cut(content, "\n")
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#!") {
		return false
	}
	fields := strings.Fields(line[2:])
	if len(fields) == 0 {
		return false
	}
	interp := fields[0]
	if path.Base(interp) == "env" && len(fields) > 1 {
		interp = fields[1]
	}
	switch path.Base(interp) {
	case "sh", "bash", "zsh", "dash", "ash", "ksh":
		return true
	}
	return false
}

func Chain(content, name string) string {
	if hasMarker(content) {
		return content
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + marker + "\n.kira/hooks/" + name + " \"$@\"\n" + markerEnd + "\n"
}
