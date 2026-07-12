package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// splices lines rather than re-encoding the document: a whole-file yaml
// re-encode would normalize comment alignment and flow styles across the
// hand-edited config.yaml
func AppendSprint(data []byte, sp datamodel.Sprint) ([]byte, error) {
	entry, err := inlineSprintEntry(sp)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	key, val := findTopLevel(&doc, "sprints")

	var out []string
	switch {
	case key == nil:
		out = appendNewSprintsBlock(lines, entry)
	case isEmptyList(val):
		out = openBlockListUnderKey(lines, key, val, entry)
	case val.Kind == yaml.SequenceNode && val.Style&yaml.FlowStyle != 0:
		out, err = appendToSingleLineFlowList(lines, val, entry)
		if err != nil {
			return nil, err
		}
	case val.Kind == yaml.SequenceNode:
		out = appendToBlockList(lines, val, entry)
	default:
		return nil, fmt.Errorf("config: sprints: expected a list, found %s", val.Tag)
	}

	res := strings.Join(out, "\n")
	cfg, err := Parse([]byte(res))
	if err != nil {
		return nil, err
	}
	if n := len(cfg.Sprints); n == 0 || cfg.Sprints[n-1] != sp {
		return nil, fmt.Errorf("config: sprints: appended entry did not round-trip")
	}
	return []byte(res), nil
}

func appendNewSprintsBlock(lines []string, entry string) []string {
	out := lines
	if len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return append(out, "sprints:", "  - "+entry, "")
}

func openBlockListUnderKey(lines []string, key, val *yaml.Node, entry string) []string {
	i := key.Line - 1
	keyLineWithoutBrackets := lines[i]
	if val.Kind == yaml.SequenceNode {
		keyLineWithoutBrackets = strings.TrimRight(strings.Replace(keyLineWithoutBrackets, "[]", "", 1), " ")
	}
	return replaceLine(lines, i, keyLineWithoutBrackets, "  - "+entry)
}

func appendToSingleLineFlowList(lines []string, val *yaml.Node, entry string) ([]string, error) {
	i := val.Line - 1
	closingBracket := strings.LastIndexByte(lines[i], ']')
	if closingBracket < 0 || maxLine(val) != val.Line {
		return nil, fmt.Errorf("config: sprints: cannot append to a multi-line flow list; reformat it as a block list")
	}
	return replaceLine(lines, i, lines[i][:closingBracket]+", "+entry+lines[i][closingBracket:]), nil
}

func appendToBlockList(lines []string, val *yaml.Node, entry string) []string {
	firstEntryLine := lines[val.Content[0].Line-1]
	indent := firstEntryLine[:strings.IndexByte(firstEntryLine, '-')+1] + " "
	lastLine := maxLine(val)
	return append(append(append([]string{}, lines[:lastLine]...), indent+entry), lines[lastLine:]...)
}

func inlineSprintEntry(sp datamodel.Sprint) (string, error) {
	fields := [4]string{}
	for i, v := range []string{sp.Key, sp.Name, sp.Start, sp.End} {
		s, err := singleLineScalar(v)
		if err != nil {
			return "", err
		}
		fields[i] = s
	}
	return fmt.Sprintf("{ key: %s, name: %s, start: %s, end: %s }",
		fields[0], fields[1], fields[2], fields[3]), nil
}

func singleLineScalar(v string) (string, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("config: sprints: %w", err)
	}
	s := strings.TrimSpace(string(b))
	if strings.ContainsAny(s, "\n\r") {
		return "", fmt.Errorf("config: sprints: field value %q does not fit on one line", v)
	}
	return s, nil
}

func findTopLevel(doc *yaml.Node, name string) (key, val *yaml.Node) {
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil
	}
	m := doc.Content[0]
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == name {
			return m.Content[i], m.Content[i+1]
		}
	}
	return nil, nil
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
