package config

import (
	"maps"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/errx"
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

// SetKeys returns the list of config keys that can be set via SetScalar.
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
		return nil, errx.User("config: unknown key %q; valid keys: %s", dottedKey, strings.Join(SetKeys(), ", "))
	}
	token, err := renderToken(kind, dottedKey, value)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, errx.User("config: %w", err)
	}
	lines := strings.Split(string(src), "\n")
	segs := strings.Split(dottedKey, ".")

	node, matched, err := descend(&doc, segs)
	if err != nil {
		return nil, err
	}
	switch {
	case matched == len(segs):
		if node.Kind != yaml.ScalarNode {
			return nil, errx.User("config: %s: not a scalar value", dottedKey)
		}
		lines, err = replaceScalarLine(lines, node, token)
		if err != nil {
			return nil, err
		}
	default:
		lines = insertUnder(lines, node, segs[matched:], token)
	}

	res := []byte(strings.Join(lines, "\n"))
	if _, err := Parse(res); err != nil {
		return nil, err
	}
	if err := verifySet(res, segs, dottedKey, value); err != nil {
		return nil, err
	}
	return res, nil
}

func verifySet(res []byte, segs []string, dottedKey, want string) error {
	var doc yaml.Node
	if err := yaml.Unmarshal(res, &doc); err != nil {
		return errx.User("config: %w", err)
	}
	leaf, matched, err := descend(&doc, segs)
	if err != nil || matched != len(segs) || leaf.Kind != yaml.ScalarNode || leaf.Value != want {
		return errx.User("config: %s: edit did not round-trip cleanly (a value may need reformatting)", dottedKey)
	}
	return nil
}

func descend(doc *yaml.Node, segs []string) (node *yaml.Node, matched int, err error) {
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, 0, errx.User("config: top level must be a mapping")
	}
	m := doc.Content[0]
	if m.Style&yaml.FlowStyle != 0 {
		return nil, 0, errx.User("config: top level must be a block mapping")
	}
	for i, seg := range segs {
		val := childValue(m, seg)
		if val == nil {
			return m, i, nil
		}
		if i == len(segs)-1 {
			return val, len(segs), nil
		}
		if val.Kind != yaml.MappingNode || val.Style&yaml.FlowStyle != 0 {
			return nil, 0, errx.User("config: %s: not a block mapping; rewrite it as `%s:` with its keys indented on the lines below", strings.Join(segs[:i+1], "."), seg)
		}
		m = val
	}
	return m, len(segs), nil
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

func renderToken(kind setKind, dottedKey, value string) (string, error) {
	if kind == kindStr {
		return singleLineScalar(dottedKey, value)
	}
	if err := oneLine(dottedKey, value); err != nil {
		return "", err
	}
	if strings.ContainsRune(value, '#') {
		return "", errx.User("config: %s: value %q must not contain '#'", dottedKey, value)
	}
	return value, nil
}
