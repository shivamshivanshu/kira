package config

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/errx"
)

// byteColumn converts n.Column (yaml.v3's 1-based rune count) into a 0-based
// byte offset into line, since splicing slices raw config lines by byte index.
func byteColumn(line string, n *yaml.Node) int {
	col := n.Column - 1
	if col <= 0 {
		return col
	}
	runes := []rune(line)
	if col > len(runes) {
		col = len(runes)
	}
	return len(string(runes[:col]))
}

// appendToTopLevelList splices entry onto data's top-level key list in place
// rather than re-encoding the document: a whole-file yaml re-encode would
// normalize comment alignment and flow styles across the hand-edited config.yaml
func appendToTopLevelList(data []byte, key, entry string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, errx.User("config: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	keyNode, val := findTopLevel(&doc, key)

	var out []string
	var err error
	switch {
	case keyNode == nil:
		out = appendNewTopLevelBlock(lines, key, entry)
	case isEmptyList(val):
		out = openBlockListUnderKey(lines, keyNode, val, entry)
	case val.Kind == yaml.SequenceNode && val.Style&yaml.FlowStyle != 0:
		if out, err = appendToFlowList(lines, key, val, entry); err != nil {
			return nil, err
		}
	case val.Kind == yaml.SequenceNode:
		out = appendToBlockList(lines, val, entry)
	default:
		return nil, errx.User("config: %s: expected a list, found %s", key, val.Tag)
	}
	return []byte(strings.Join(out, "\n")), nil
}

func appendNewTopLevelBlock(lines []string, key, entry string) []string {
	out := lines
	if len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return append(out, key+":", "  - "+entry, "")
}

func openBlockListUnderKey(lines []string, key, val *yaml.Node, entry string) []string {
	if val.Kind == yaml.SequenceNode && val.Line != key.Line {
		i := val.Line - 1
		line := lines[i]
		open := byteColumn(line, val)
		if closing := flowCloseIndex(line, open); closing >= 0 {
			return replaceLine(lines, i, "  - "+entry+line[closing+1:])
		}
	}
	i := key.Line - 1
	line := lines[i]
	if val.Kind == yaml.SequenceNode && val.Line == key.Line {
		open := byteColumn(line, val)
		if closing := flowCloseIndex(line, open); closing >= 0 {
			line = line[:open] + line[closing+1:]
		}
	}
	return replaceLine(lines, i, strings.TrimRight(line, " "), "  - "+entry)
}

func appendToBlockList(lines []string, val *yaml.Node, entry string) []string {
	firstEntryLine := lines[val.Content[0].Line-1]
	indent := firstEntryLine[:strings.IndexByte(firstEntryLine, '-')+1] + " "
	lastLine := maxLine(val)
	return append(append(append([]string{}, lines[:lastLine]...), indent+entry), lines[lastLine:]...)
}

func appendToFlowList(lines []string, subsystem string, val *yaml.Node, entry string) ([]string, error) {
	i := val.Line - 1
	open := byteColumn(lines[i], val)
	closing := -1
	if maxLine(val) == val.Line && open >= 0 && open < len(lines[i]) && lines[i][open] == '[' {
		closing = flowCloseIndex(lines[i], open)
	}
	if closing < 0 {
		return nil, errx.User("config: %s: cannot append to a multi-line flow list; reformat it as a block list", subsystem)
	}
	sep := ", "
	if strings.TrimSpace(lines[i][open+1:closing]) == "" {
		sep = ""
	}
	return replaceLine(lines, i, lines[i][:closing]+sep+entry+lines[i][closing:]), nil
}

func flowCloseIndex(line string, open int) int {
	depth, quote := 0, byte(0)
	for j := open; j < len(line); j++ {
		c := line[j]
		switch {
		case quote == '\'':
			if c != '\'' {
				continue
			}
			if j+1 < len(line) && line[j+1] == '\'' {
				j++
			} else {
				quote = 0
			}
		case quote == '"':
			switch c {
			case '\\':
				j++
			case '"':
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '[' || c == '{':
			depth++
		case c == ']' || c == '}':
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

func oneLine(subsystem, v string) error {
	if strings.ContainsAny(v, "\n\r") {
		return errx.User("config: %s: value %q does not fit on one line", subsystem, v)
	}
	return nil
}

func flowScalar(field, v string) (string, error) {
	if err := oneLine(field, v); err != nil {
		return "", err
	}
	if v == "" || v != strings.TrimSpace(v) || strings.ContainsAny(v, ",{}[]:#&*!|>'\"%@`") {
		return "'" + strings.ReplaceAll(v, "'", "''") + "'", nil
	}
	b, err := yaml.Marshal(v)
	if err != nil {
		return "", errx.User("config: %s: %w", field, err)
	}
	return strings.TrimSpace(string(b)), nil
}

func singleLineScalar(subsystem, v string) (string, error) {
	if err := oneLine(subsystem, v); err != nil {
		return "", err
	}
	b, err := yaml.Marshal(v)
	if err != nil {
		return "", errx.User("config: %s: %w", subsystem, err)
	}
	s := strings.TrimSpace(string(b))
	if err := oneLine(subsystem, s); err != nil {
		return "", err
	}
	return s, nil
}

func replaceScalarLine(lines []string, leaf *yaml.Node, token string) ([]string, error) {
	i := leaf.Line - 1
	if i < 0 || i >= len(lines) {
		return nil, errx.User("config: value node points outside the file")
	}
	line := lines[i]
	col := byteColumn(line, leaf)
	old := marshalScalar(leaf)
	if col < 0 || col+len(old) > len(line) || line[col:col+len(old)] != old {
		return nil, errx.User("config: cannot locate value on line %d", leaf.Line)
	}
	return replaceLine(lines, i, line[:col]+token+line[col+len(old):]), nil
}

func marshalScalar(n *yaml.Node) string {
	bare := yaml.Node{Kind: yaml.ScalarNode, Tag: n.Tag, Value: n.Value, Style: n.Style}
	b, err := yaml.Marshal(&bare)
	if err != nil {
		return n.Value
	}
	return strings.TrimRight(string(b), "\n")
}

func findTopLevel(doc *yaml.Node, name string) (key, val *yaml.Node) {
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil
	}
	return mapEntry(doc.Content[0], name)
}

func mapEntry(m *yaml.Node, name string) (key, val *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == name {
			return m.Content[i], m.Content[i+1]
		}
	}
	return nil, nil
}

func childValue(m *yaml.Node, name string) *yaml.Node {
	_, v := mapEntry(m, name)
	return v
}

func isEmptyList(n *yaml.Node) bool {
	return (n.Kind == yaml.SequenceNode && len(n.Content) == 0) ||
		(n.Kind == yaml.ScalarNode && n.Tag == "!!null")
}

func maxLine(n *yaml.Node) int {
	m := n.Line
	for _, c := range n.Content {
		if l := maxLine(c); l > m {
			m = l
		}
	}
	return m
}

func replaceLine(lines []string, i int, repl ...string) []string {
	return append(append(append([]string{}, lines[:i]...), repl...), lines[i+1:]...)
}
