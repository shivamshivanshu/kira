// Package hooks holds kira's tracked git-hook shim scripts and the pure logic
// for chaining kira's hooks onto pre-existing ones without clobbering them.
package hooks

import (
	"embed"
	"path"
	"regexp"
	"strings"
)

//go:embed scripts/post-merge scripts/prepare-commit-msg scripts/pre-commit
var scriptFS embed.FS

const (
	marker    = "# kira:chain v1"
	markerEnd = "# /kira:chain"
)

var Default = []string{"post-merge", "prepare-commit-msg"}

const PreCommit = "pre-commit"

func Known() []string {
	return append(append([]string(nil), Default...), PreCommit)
}

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
	return strings.Contains(content, "kira hooks run "+name) ||
		strings.Contains(content, "kira hooks "+name)
}

func IsPureShim(content, name string) bool {
	if hasMarker(content) {
		return false
	}
	delegates := false
	for line := range strings.Lines(content) {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "" || strings.HasPrefix(trimmed, "#"):
		case guardLineRe.MatchString(trimmed):
		case delegationLineRe(name).MatchString(trimmed):
			delegates = true
		default:
			return false
		}
	}
	return delegates
}

var guardLineRe = regexp.MustCompile(`^command\s+-v\s+kira\s+>/dev/null\s+2>&1\s+\|\|\s+exit\s+0$`)

const shimArgs = `[\s"'$@*]*$`

func delegationLineRe(name string) *regexp.Regexp {
	q := regexp.QuoteMeta(name)
	return regexp.MustCompile(`^(exec\s+)?(kira\s+hooks\s+(run\s+)?|\.kira/hooks/)` + q + shimArgs)
}

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
		return content + "\n" + chainBlockAddedNewline(name)
	}
	return content + chainBlock(name)
}

func chainBlock(name string) string {
	return marker + "\n" + chainTail(name)
}

func chainBlockAddedNewline(name string) string {
	return marker + " nonl\n" + chainTail(name)
}

func chainTail(name string) string {
	return ".kira/hooks/" + name + " \"$@\"\n" + markerEnd + "\n"
}

func Unchain(content, name string) (string, bool) {
	if stripped := strings.Replace(content, chainBlockAddedNewline(name), "", 1); stripped != content {
		return strings.TrimSuffix(stripped, "\n"), true
	}
	stripped := strings.Replace(content, chainBlock(name), "", 1)
	return stripped, stripped != content
}

type State string

const (
	StateInstalled State = "installed"
	StateChained   State = "chained"
	StateMissing   State = "missing"
	StateDrifted   State = "drifted"
	StateForeign   State = "foreign"
)

func StateOf(content, name string) State {
	script, ok := Script(name)
	if ok && content == script {
		return StateInstalled
	}
	if hasMarker(content) {
		if strings.Contains(content, chainBlock(name)) || strings.Contains(content, chainBlockAddedNewline(name)) {
			return StateChained
		}
		return StateDrifted
	}
	if Invokes(content, name) {
		return StateDrifted
	}
	return StateForeign
}
