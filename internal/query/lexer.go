package query

import (
	"strings"
	"unicode/utf8"
)

type tokKind int

const (
	tokEOF tokKind = iota
	tokWord
	tokString
	tokLParen
	tokRParen
	tokComma
	tokEq
	tokNe
	tokLt
	tokLe
	tokGt
	tokGe
)

type token struct {
	kind tokKind
	text string
	pos  int
}

func (t token) isCmp() bool { return t.kind >= tokEq && t.kind <= tokGe }

func (t token) cmpText() string {
	switch t.kind {
	case tokEq:
		return "="
	case tokNe:
		return "!="
	case tokLt:
		return "<"
	case tokLe:
		return "<="
	case tokGt:
		return ">"
	case tokGe:
		return ">="
	default:
		return ""
	}
}

// Dashes, colons, dots, and pipes stay inside a bare word so dates, ULIDs,
// KEY-n numbers, RFC3339 timestamps, rank strings, and state keys lex as one
// token.
func wordStop(r rune) bool {
	switch r {
	case '(', ')', ',', '"', '=', '!', '<', '>':
		return true
	}
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func lex(input string) ([]token, error) {
	var toks []token
	i := 0
	for i < len(input) {
		r, w := utf8.DecodeRuneInString(input[i:])
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			i += w
		case r == '(':
			toks = append(toks, token{tokLParen, "(", i})
			i += w
		case r == ')':
			toks = append(toks, token{tokRParen, ")", i})
			i += w
		case r == ',':
			toks = append(toks, token{tokComma, ",", i})
			i += w
		case r == '=':
			toks = append(toks, token{tokEq, "=", i})
			i += w
		case r == '!':
			if i+w < len(input) && input[i+w] == '=' {
				toks = append(toks, token{tokNe, "!=", i})
				i += w + 1
			} else {
				return nil, &Error{Pos: i, Msg: "expected '=' after '!'"}
			}
		case r == '<':
			if i+w < len(input) && input[i+w] == '=' {
				toks = append(toks, token{tokLe, "<=", i})
				i += w + 1
			} else {
				toks = append(toks, token{tokLt, "<", i})
				i += w
			}
		case r == '>':
			if i+w < len(input) && input[i+w] == '=' {
				toks = append(toks, token{tokGe, ">=", i})
				i += w + 1
			} else {
				toks = append(toks, token{tokGt, ">", i})
				i += w
			}
		case r == '"':
			s, next, err := lexString(input, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, token{tokString, s, i})
			i = next
		default:
			start := i
			for i < len(input) {
				r, w := utf8.DecodeRuneInString(input[i:])
				if wordStop(r) {
					break
				}
				i += w
			}
			toks = append(toks, token{tokWord, input[start:i], start})
		}
	}
	toks = append(toks, token{tokEOF, "", len(input)})
	return toks, nil
}

func lexString(input string, i int) (string, int, error) {
	var b strings.Builder
	j := i + 1
	for j < len(input) {
		c := input[j]
		switch c {
		case '"':
			return b.String(), j + 1, nil
		case '\\':
			if j+1 >= len(input) {
				return "", 0, &Error{Pos: j, Msg: "dangling escape in string"}
			}
			n := input[j+1]
			if n != '"' && n != '\\' {
				return "", 0, &Error{Pos: j, Msg: "invalid escape '\\" + string(n) + "' in string"}
			}
			b.WriteByte(n)
			j += 2
		default:
			b.WriteByte(c)
			j++
		}
	}
	return "", 0, &Error{Pos: i, Msg: "unterminated string"}
}
