// Package workon holds the pure branch-naming, slug, and active-pointer logic
// behind `kira workon` and the prepare-commit-msg trailer hook.
package workon

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
)

const (
	sepChars = "-_"
	sepClass = `[-_]`
	boundary = `[^a-z0-9]`
)

var (
	placeholderRe = regexp.MustCompile(`\{(key|number|slug)\}`)
	bareTokenRe   = regexp.MustCompile(`^[0-9A-Za-z]+$`)
)

func normalize(s, sep string) string {
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

func Slug(title, sep string) string { return normalize(title, sep) }

func RenderBranch(pattern, key, number, title, sep string) string {
	return strings.NewReplacer(
		"{key}", normalize(key, sep),
		"{number}", normalize(number, sep),
		"{slug}", Slug(title, sep),
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

func quoteSeparatorInsensitive(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(sepChars, r) {
			b.WriteString(sepClass)
		} else {
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	return b.String()
}

func branchRegexp(pattern, key, number, sep string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString(`^`)
	rest := pattern
	for rest != "" {
		loc := placeholderRe.FindStringIndex(rest)
		if loc == nil {
			b.WriteString(regexp.QuoteMeta(rest))
			break
		}
		lit, ph := rest[:loc[0]], rest[loc[0]:loc[1]]
		rest = rest[loc[1]:]
		if ph == "{slug}" && rest == "" && lit != "" && strings.Trim(lit, sepChars+"/") == "" {
			b.WriteString(`(?:[` + sepChars + `/].*)?`)
			continue
		}
		b.WriteString(regexp.QuoteMeta(lit))
		switch ph {
		case "{key}":
			b.WriteString(quoteSeparatorInsensitive(normalize(key, sep)))
		case "{number}":
			b.WriteString(quoteSeparatorInsensitive(normalize(number, sep)))
		case "{slug}":
			b.WriteString(`.*`)
		}
	}
	b.WriteString(`$`)
	return regexp.MustCompile(b.String())
}

func MatchBranch(branches []string, pattern, key, number, sep string) (string, bool) {
	re := branchRegexp(pattern, key, number, sep)
	for _, b := range branches {
		if re.MatchString(b) {
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
		re := regexp.MustCompile(`(?i)(?:^|` + boundary + `)` + regexp.QuoteMeta(key) + sepClass + `(\d+)(?:` + boundary + `|$)`)
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
	if err := json.Unmarshal([]byte(trimmed), &p); err == nil {
		return p, p.Ticket != ""
	}
	if bareTokenRe.MatchString(trimmed) {
		return ActivePointer{Ticket: trimmed}, true
	}
	return ActivePointer{}, false
}
