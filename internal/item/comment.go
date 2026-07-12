package item

import (
	"regexp"
	"strings"
)

// Comment is one anchored comment block from the body's `## Comments` section.
// Grammar (docs/design/02-data-model.md §4):
//
//	<!-- kira:comment id=<ULID> author=<string> ts=<RFC3339> -->
//	<body, markdown, free-form>
//	<!-- /kira:comment -->
type Comment struct {
	ID     string
	Author string
	Ts     string
	Body   string
}

// commentOpen matches the opening marker. Fields are whitespace-delimited
// tokens, so author values are single tokens (people identifiers, no spaces).
var commentOpen = regexp.MustCompile(`^<!-- kira:comment id=(\S+) author=(\S+) ts=(\S+) -->$`)

const commentClose = "<!-- /kira:comment -->"

// ParseComments extracts every well-formed comment block from a body, in file
// order. Malformed or unterminated blocks are skipped (validation of block
// well-formedness is a warn-level concern handled by `kira validate`).
func ParseComments(body string) []Comment {
	lines := strings.Split(body, "\n")
	var out []Comment
	for i := 0; i < len(lines); i++ {
		m := commentOpen.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		close := -1
		for j := i + 1; j < len(lines); j++ {
			if lines[j] == commentClose {
				close = j
				break
			}
		}
		if close < 0 {
			break // unterminated: no valid block can follow
		}
		out = append(out, Comment{
			ID:     m[1],
			Author: m[2],
			Ts:     m[3],
			Body:   strings.Join(lines[i+1:close], "\n"),
		})
		i = close
	}
	return out
}

// AppendComment appends a comment block to raw file content. It is a pure
// byte-suffix operation: the original content is an exact prefix of the result,
// so concurrent appends on different branches stay disjoint and merge cleanly
// (docs/design/02-data-model.md §4). The `## Comments` section is last in the
// body, so appending to end-of-file appends within that section.
func AppendComment(content string, c Comment) string {
	return content + "\n" + formatComment(c) + "\n"
}

func formatComment(c Comment) string {
	return "<!-- kira:comment id=" + c.ID + " author=" + c.Author + " ts=" + c.Ts + " -->\n" +
		c.Body + "\n" +
		commentClose
}
