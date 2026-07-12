package codec

import (
	"regexp"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

var commentOpen = regexp.MustCompile(`^<!-- kira:comment id=(\S+) author=(\S+) ts=(\S+) -->$`)

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

func AppendComment(content string, c datamodel.Comment) string {
	return content + "\n" + formatComment(c) + "\n"
}

func formatComment(c datamodel.Comment) string {
	return "<!-- kira:comment id=" + c.ID + " author=" + c.Author + " ts=" + c.Ts + " -->\n" +
		c.Body + "\n" +
		commentClose
}
