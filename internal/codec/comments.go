package codec

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const commentMarker = "<!-- kira:comment id="

var commentOpen = regexp.MustCompile(`^` + regexp.QuoteMeta(commentMarker) + `(\S+) author=(\S+) ts=(\S+) -->$`)

const commentClose = "<!-- /kira:comment -->"

func ParseComments(body string) []datamodel.Comment {
	lines := strings.Split(body, "\n")
	var out []datamodel.Comment
	for i := 0; i < len(lines); i++ {
		m := commentOpen.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		end := -1
		for j := i + 1; j < len(lines); j++ {
			if lines[j] == commentClose {
				end = j
				break
			}
		}
		if end < 0 {
			break
		}
		out = append(out, datamodel.Comment{
			ID:     m[1],
			Author: m[2],
			Ts:     m[3],
			Body:   strings.Join(lines[i+1:end], "\n"),
		})
		i = end
	}
	return out
}

func LintComments(body string) []string {
	var out []string
	open := false
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "<!-- kira:comment"):
			if open {
				out = append(out, "comment block opened before the previous one closed")
			}
			open = true
			m := commentOpen.FindStringSubmatch(trimmed)
			if m == nil {
				out = append(out, "comment marker does not match `id=<ulid> author=<name> ts=<rfc3339>`")
				continue
			}
			if _, err := time.Parse(time.RFC3339, m[3]); err != nil {
				out = append(out, "comment timestamp "+strconv.Quote(m[3])+" is not RFC3339")
			}
		case trimmed == commentClose:
			if !open {
				out = append(out, "comment close marker without a matching open")
			}
			open = false
		}
	}
	if open {
		out = append(out, "comment block is never closed")
	}
	return out
}

func Description(body string) string {
	lines := strings.Split(body, "\n")
	if i := commentBoundary(lines); i >= 0 {
		return strings.TrimRight(strings.Join(lines[:i], "\n"), "\n")
	}
	return strings.TrimRight(body, "\n")
}

func commentBoundary(lines []string) int {
	for i, line := range lines {
		if commentOpen.MatchString(line) {
			return i
		}
	}
	return -1
}

func AppendComment(content string, c datamodel.Comment) string {
	return content + "\n" + formatComment(c) + "\n"
}

func SplitComments(body string) (prose string, comments []datamodel.Comment) {
	comments = ParseComments(body)
	lines := strings.Split(body, "\n")
	if i := commentBoundary(lines); i >= 0 {
		return strings.Join(lines[:i], "\n"), comments
	}
	return body, comments
}

func JoinComments(prose string, comments []datamodel.Comment) string {
	out := prose
	for _, c := range comments {
		out = AppendComment(out, c)
	}
	return out
}

func formatComment(c datamodel.Comment) string {
	return commentMarker + c.ID + " author=" + c.Author + " ts=" + c.Ts + " -->\n" +
		c.Body + "\n" +
		commentClose
}
