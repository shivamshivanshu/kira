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

const (
	PostMerge        = "post-merge"
	PrepareCommitMsg = "prepare-commit-msg"
	PreCommit        = "pre-commit"
)

var defaultHooks = []string{PostMerge, PrepareCommitMsg}

func Defaults() []string {
	return append([]string(nil), defaultHooks...)
}

func Known() []string {
	return append(Defaults(), PreCommit)
}

func Script(name string) (string, bool) {
	data, err := scriptFS.ReadFile("scripts/" + name)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func HasMarker(content string) bool {
	return strings.Contains(content, marker)
}

// Invokes reports whether an installed hook already delegates to kira. It is the
// single source of truth for the invocation the embedded scripts use, so
// re-install idempotency stays coupled to the script text (pinned by a test).
func Invokes(content, name string) bool {
	return strings.Contains(content, "kira hooks run "+name) ||
		strings.Contains(content, "kira hooks "+name) ||
		strings.Contains(content, ".kira/hooks/"+name)
}

func IsPureShim(content, name string) bool {
	if HasMarker(content) {
		return false
	}
	delegation := delegationLineRe(name)
	delegates := false
	for line := range strings.Lines(content) {
		trimmed := strings.TrimSpace(line)
		switch {
		case isBlankOrComment(trimmed):
		case guardLineRe.MatchString(trimmed):
		case delegation.MatchString(trimmed):
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
	state := StateOf(content, name)
	return state.Installed(), state == StateChained
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
	if path.Base(interp) == "env" {
		rest := fields[1:]
		for len(rest) > 0 && strings.HasPrefix(rest[0], "-") {
			rest = rest[1:]
		}
		if len(rest) == 0 {
			return false
		}
		interp = rest[0]
	}
	switch path.Base(interp) {
	case "sh", "bash", "zsh", "dash", "ash", "ksh":
		return true
	}
	return false
}

func EndsInExecOrExit(content string) bool {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if isBlankOrComment(trimmed) {
			continue
		}
		return execOrExitLineRe.MatchString(trimmed)
	}
	return false
}

func isBlankOrComment(trimmedLine string) bool {
	return trimmedLine == "" || strings.HasPrefix(trimmedLine, "#")
}

var execOrExitLineRe = regexp.MustCompile(`^(exec|exit)\b`)

func Chain(content, name string) string {
	if HasMarker(content) {
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

// git treats a 127 exit from a hook script as failure.
func chainTail(name string) string {
	shim := ".kira/hooks/" + name
	return "rc=$?\n" +
		"if [ -x \"" + shim + "\" ]; then\n" +
		"  \"" + shim + "\" \"$@\" || exit\n" +
		"fi\n" +
		"exit $rc\n" +
		markerEnd + "\n"
}

func Unchain(content, name string) (string, bool) {
	if stripped := strings.Replace(content, "\n"+chainBlockAddedNewline(name), "", 1); stripped != content {
		return stripped, true
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

func (s State) Installed() bool {
	return s == StateInstalled || s == StateChained
}

func StateOf(content, name string) State {
	script, ok := Script(name)
	if ok && content == script {
		return StateInstalled
	}
	if HasMarker(content) {
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
