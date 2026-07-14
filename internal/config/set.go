package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type setKind int

const (
	kindLiteral setKind = iota
	kindStr
)

var setKeys = map[string]setKind{
	"project.name":          kindStr,
	"commit.mode":           kindLiteral,
	"commit.trailer":        kindStr,
	"commit.close_trailer":  kindStr,
	"merge.policy":          kindLiteral,
	"sync.push":             kindLiteral,
	"ui.icons":              kindLiteral,
	"ui.background":         kindLiteral,
	"workon.branch_pattern": kindStr,
	"workon.casing":         kindLiteral,
	"git.landed_ref":        kindStr,
	"estimate.unit":         kindLiteral,
}

func SetKeys() []string {
	return slices.Sorted(maps.Keys(setKeys))
}

// SetScalar edits one scalar by splicing its single line rather than
// re-encoding the document, so every comment and untouched line stays
// byte-identical (a whole-file yaml re-encode reflows comment alignment and
// flow styles).
func SetScalar(src []byte, dottedKey, value string) ([]byte, error) {
	kind, ok := setKeys[dottedKey]
	if !ok {
		return nil, fmt.Errorf("config: unknown key %q; valid keys: %s", dottedKey, strings.Join(SetKeys(), ", "))
	}
	token, err := renderToken(kind, dottedKey, value)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	lines := strings.Split(string(src), "\n")
	segs := strings.Split(dottedKey, ".")

	node, matched := descend(&doc, segs)
	if node == nil {
		return nil, fmt.Errorf("config: top level must be a mapping")
	}
	switch {
	case matched == len(segs):
		if node.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("config: %s: not a scalar value", dottedKey)
		}
		lines, err = replaceScalarLine(lines, node, token)
		if err != nil {
			return nil, err
		}
	default:
		lines = insertUnder(lines, node, segs[matched:], token)
	}

	res := strings.Join(lines, "\n")
	if _, err := Parse([]byte(res)); err != nil {
		return nil, err
	}
	return []byte(res), nil
}

func descend(doc *yaml.Node, segs []string) (node *yaml.Node, matched int) {
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, 0
	}
	m := doc.Content[0]
	for i, seg := range segs {
		val := childValue(m, seg)
		if val == nil {
			return m, i
		}
		if i == len(segs)-1 {
			return val, len(segs)
		}
		if val.Kind != yaml.MappingNode {
			return m, i
		}
		m = val
	}
	return m, len(segs)
}

func childValue(m *yaml.Node, name string) *yaml.Node {
	_, v := mapEntry(m, name)
	return v
}

func replaceScalarLine(lines []string, leaf *yaml.Node, token string) ([]string, error) {
	i := leaf.Line - 1
	if i < 0 || i >= len(lines) {
		return nil, fmt.Errorf("config: value node points outside the file")
	}
	line := lines[i]
	col := leaf.Column - 1
	old := marshalScalar(leaf)
	if col < 0 || col+len(old) > len(line) || line[col:col+len(old)] != old {
		return nil, fmt.Errorf("config: cannot locate value on line %d", leaf.Line)
	}
	return replaceLine(lines, i, line[:col]+token+line[col+len(old):]), nil
}

func insertUnder(lines []string, parent *yaml.Node, suffix []string, token string) []string {
	base := childIndent(parent)
	block := make([]string, 0, len(suffix))
	for d, seg := range suffix[:len(suffix)-1] {
		block = append(block, strings.Repeat(" ", base+d*2)+seg+":")
	}
	block = append(block, strings.Repeat(" ", base+(len(suffix)-1)*2)+suffix[len(suffix)-1]+": "+token)

	at := maxLine(parent)
	out := append([]string{}, lines[:at]...)
	out = append(out, block...)
	return append(out, lines[at:]...)
}

func childIndent(m *yaml.Node) int {
	if len(m.Content) > 0 {
		return m.Content[0].Column - 1
	}
	return 0
}

func marshalScalar(n *yaml.Node) string {
	bare := yaml.Node{Kind: yaml.ScalarNode, Tag: n.Tag, Value: n.Value, Style: n.Style}
	b, err := yaml.Marshal(&bare)
	if err != nil {
		return n.Value
	}
	return strings.TrimRight(string(b), "\n")
}

func renderToken(kind setKind, dottedKey, value string) (string, error) {
	if kind == kindStr {
		return singleLineScalar(dottedKey, value)
	}
	if strings.ContainsAny(value, "\n\r") {
		return "", fmt.Errorf("config: value %q does not fit on one line", value)
	}
	return value, nil
}
