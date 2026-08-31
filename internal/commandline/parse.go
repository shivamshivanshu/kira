package commandline

import (
	"strings"
	"unicode"

	"github.com/shivamshivanshu/kira/internal/errx"
)

// Parse splits arguments with quoting and escaping, without shell expansion.
func Parse(line string) ([]string, error) {
	var args []string
	var word strings.Builder
	var quote rune
	inWord, escaped := false, false
	flush := func() {
		if inWord {
			args = append(args, word.String())
			word.Reset()
			inWord = false
		}
	}
	for _, r := range line {
		switch {
		case escaped:
			word.WriteRune(r)
			escaped = false
		case r == '\\' && quote != '\'':
			escaped, inWord = true, true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case unicode.IsSpace(r):
			flush()
		default:
			word.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 || escaped {
		return nil, errx.User("unfinished quote or escape in command")
	}
	flush()
	return args, nil
}
