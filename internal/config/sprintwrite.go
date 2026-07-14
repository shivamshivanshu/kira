package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func AppendSprint(data []byte, sp datamodel.Sprint) ([]byte, error) {
	entry, err := inlineSprintEntry(sp)
	if err != nil {
		return nil, err
	}
	out, err := appendToTopLevelList(data, "sprints", entry)
	if err != nil {
		return nil, err
	}
	cfg, err := Parse(out)
	if err != nil {
		return nil, err
	}
	if n := len(cfg.Sprints); n == 0 || cfg.Sprints[n-1] != sp {
		return nil, fmt.Errorf("config: sprints: appended entry did not round-trip")
	}
	return out, nil
}

// appendToTopLevelList splices entry onto data's top-level key list in place
// rather than re-encoding the document: a whole-file yaml re-encode would
// normalize comment alignment and flow styles across the hand-edited config.yaml
func appendToTopLevelList(data []byte, key, entry string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("config: %w", err)
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
		return nil, fmt.Errorf("config: %s: expected a list, found %s", key, val.Tag)
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
	i := key.Line - 1
	keyLineWithoutBrackets := lines[i]
	if val.Kind == yaml.SequenceNode {
		keyLineWithoutBrackets = strings.TrimRight(strings.Replace(keyLineWithoutBrackets, "[]", "", 1), " ")
	}
	return replaceLine(lines, i, keyLineWithoutBrackets, "  - "+entry)
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
		s, err := flowScalar("sprints", v)
		if err != nil {
			return "", err
		}
		fields[i] = s
	}
	return fmt.Sprintf("{ key: %s, name: %s, start: %s, end: %s }",
		fields[0], fields[1], fields[2], fields[3]), nil
}

func singleLineScalar(subsystem, v string) (string, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("config: %s: %w", subsystem, err)
	}
	s := strings.TrimSpace(string(b))
	if strings.ContainsAny(s, "\n\r") {
		return "", fmt.Errorf("config: %s: field value %q does not fit on one line", subsystem, v)
	}
	return s, nil
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
