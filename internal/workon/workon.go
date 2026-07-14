// Package workon holds the pure branch-naming, slug, and active-pointer logic
// behind `kira workon` and the prepare-commit-msg trailer hook.
package workon

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func separator(c datamodel.Casing) string {
	if c == datamodel.CasingSnake {
		return "_"
	}
	return "-"
}

func normalize(s string, c datamodel.Casing) string {
	sep := separator(c)
	var b strings.Builder
	prevSep := true
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevSep = false
			continue
		}
		if !prevSep {
			b.WriteString(sep)
			prevSep = true
		}
	}
	return strings.TrimRight(b.String(), sep)
}

func Slug(title string, c datamodel.Casing) string { return normalize(title, c) }

func RenderBranch(pattern, key, number, title string, c datamodel.Casing) string {
	return strings.NewReplacer(
		"{key}", normalize(key, c),
		"{number}", normalize(number, c),
		"{slug}", Slug(title, c),
	).Replace(pattern)
}

func RenderWorktreeDir(pattern, repo, branch, key, number string) string {
	return strings.NewReplacer(
		"{repo}", repo,
		"{branch}", strings.ReplaceAll(branch, "/", "-"),
		"{key}", key,
		"{number}", number,
	).Replace(pattern)
}

func branchPrefix(pattern, key, number string, c datamodel.Casing) string {
	p := pattern
	if i := strings.Index(p, "{slug}"); i >= 0 {
		p = p[:i]
	}
	return strings.NewReplacer(
		"{key}", normalize(key, c),
		"{number}", normalize(number, c),
	).Replace(p)
}

func MatchBranch(branches []string, pattern, key, number string, c datamodel.Casing) (string, bool) {
	prefix := branchPrefix(pattern, key, number, c)
	bare := strings.TrimRight(prefix, "-_/")
	for _, b := range branches {
		if b == bare || strings.HasPrefix(b, prefix) {
			return b, true
		}
	}
	return "", false
}

func InferNumber(branch string, keys []string) (string, bool) {
	sorted := append([]string(nil), keys...)
	slices.SortFunc(sorted, func(a, b string) int {
		if len(a) != len(b) {
			return len(b) - len(a)
		}
		return strings.Compare(a, b)
	})
	for _, key := range sorted {
		if key == "" {
			continue
		}
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(key) + `-(\d+)\b`)
		if m := re.FindStringSubmatch(branch); m != nil {
			return key + "-" + m[1], true
		}
	}
	return "", false
}

type ActivePointer struct {
	Ticket string `json:"ticket"`
	Branch string `json:"branch"`
}

func (p ActivePointer) Marshal() []byte {
	data, _ := json.Marshal(p)
	return append(data, '\n')
}

func ParseActive(data []byte) (ActivePointer, bool) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return ActivePointer{}, false
	}
	var p ActivePointer
	if err := json.Unmarshal([]byte(trimmed), &p); err == nil && p.Ticket != "" {
		return p, true
	}
	// Legacy pre-WP-3.1.5 pointer: a bare ULID line, no recorded branch.
	return ActivePointer{Ticket: trimmed}, true
}
