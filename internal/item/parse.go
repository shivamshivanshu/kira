package item

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	fenceLine  = "---\n"
	closeFence = "\n" + fenceLine
)

// Parse decodes a kira item file (frontmatter + markdown body). It collects
// every validation error into a *ParseError instead of failing on the first;
// on success it returns a fully populated *Item whose Serialize reproduces a
// canonical file. A malformed document (no frontmatter fences, unparseable
// YAML) returns a plain error, since no per-field recovery is possible.
func Parse(content string) (*Item, error) {
	front, body, err := splitDocument(content)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(front), &doc); err != nil {
		return nil, fmt.Errorf("frontmatter yaml: %w", err)
	}

	nodes := frontmatterNodes(&doc)

	it := &Item{Body: body}
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	it.ID = reqScalar(nodes, keyID, add)
	it.Number = reqScalar(nodes, keyNumber, add)
	it.Aliases = reqList(nodes, keyAliases, add)
	it.Type = reqScalar(nodes, keyType, add)
	if it.Type != "" && it.Type != TypeTicket && it.Type != TypeEpic {
		add("field %q: must be %s or %s, got %q", keyType, TypeTicket, TypeEpic, it.Type)
	}
	it.Title = reqScalar(nodes, keyTitle, add)
	it.State = reqScalar(nodes, keyState, add)
	it.Priority = optScalar(nodes, keyPriority, add)
	it.Owner = optScalar(nodes, keyOwner, add)
	it.Reporter = optScalar(nodes, keyReporter, add)
	it.Labels = reqList(nodes, keyLabels, add)
	it.Epic = nullableScalar(nodes, keyEpic, add)
	it.BlockedBy = reqList(nodes, keyBlockedBy, add)
	it.Estimate = optFloat(nodes, keyEstimate, add)
	it.Created = reqTimestamp(nodes, keyCreated, add)
	it.Updated = reqTimestamp(nodes, keyUpdated, add)

	if len(errs) > 0 {
		return it, &ParseError{Errs: errs}
	}
	return it, nil
}

// splitDocument splits raw file content into the frontmatter YAML text and the
// markdown body (everything after the closing fence, verbatim). The frontmatter
// closer is the first "\n---\n" after the opening fence; a "---" line only ever
// means a horizontal rule once it appears in the body, which is past that point.
func splitDocument(content string) (front, body string, err error) {
	if !strings.HasPrefix(content, fenceLine) {
		return "", "", fmt.Errorf("missing opening frontmatter fence")
	}
	rest := content[len(fenceLine):]
	if i := strings.Index(rest, closeFence); i >= 0 {
		return rest[:i+1], rest[i+len(closeFence):], nil
	}
	return "", "", fmt.Errorf("missing closing frontmatter fence")
}

// frontmatterNodes maps each top-level key to its value node. A non-mapping or
// empty document yields an empty map, so every required field is reported
// missing rather than panicking.
func frontmatterNodes(doc *yaml.Node) map[string]*yaml.Node {
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return map[string]*yaml.Node{}
	}
	m := doc.Content[0]
	nodes := make(map[string]*yaml.Node, len(m.Content)/2)
	for i := 0; i+1 < len(m.Content); i += 2 {
		nodes[m.Content[i].Value] = m.Content[i+1]
	}
	return nodes
}

func isNull(n *yaml.Node) bool { return n.Kind == yaml.ScalarNode && n.Tag == "!!null" }

type addFunc func(format string, args ...any)

const (
	errMissing   = "field %q: required, missing"
	errNotScalar = "field %q: expected a scalar"
)

// reqScalar reads a required scalar as its raw source text (node.Value), which
// preserves the exact bytes — critical for timestamp-shaped strings that a
// decode-then-encode round-trip would re-quote.
func reqScalar(nodes map[string]*yaml.Node, key string, add addFunc) string {
	n, ok := nodes[key]
	if !ok {
		add(errMissing, key)
		return ""
	}
	if isNull(n) || n.Kind != yaml.ScalarNode {
		add(errNotScalar, key)
		return ""
	}
	if n.Value == "" {
		add("field %q: expected a non-empty scalar", key)
		return ""
	}
	return n.Value
}

// optScalar reads an optional scalar; absent or null yields nil so the writer
// omits the line.
func optScalar(nodes map[string]*yaml.Node, key string, add addFunc) *string {
	n, ok := nodes[key]
	if !ok || isNull(n) {
		return nil
	}
	if n.Kind != yaml.ScalarNode {
		add(errNotScalar, key)
		return nil
	}
	v := n.Value
	return &v
}

// nullableScalar reads a required-but-nullable scalar (epic): the key must be
// present, but an explicit null is valid and yields nil.
func nullableScalar(nodes map[string]*yaml.Node, key string, add addFunc) *string {
	n, ok := nodes[key]
	if !ok {
		add(errMissing, key)
		return nil
	}
	if isNull(n) {
		return nil
	}
	if n.Kind != yaml.ScalarNode {
		add("field %q: expected a scalar or null", key)
		return nil
	}
	v := n.Value
	return &v
}

// reqList reads a required list; null or an empty sequence yields a non-nil
// empty slice (matching how the writer emits []).
func reqList(nodes map[string]*yaml.Node, key string, add addFunc) []string {
	n, ok := nodes[key]
	if !ok {
		add(errMissing, key)
		return []string{}
	}
	if isNull(n) {
		return []string{}
	}
	if n.Kind != yaml.SequenceNode {
		add("field %q: expected a list", key)
		return []string{}
	}
	out := make([]string, 0, len(n.Content))
	for _, c := range n.Content {
		if c.Kind != yaml.ScalarNode {
			add("field %q: list elements must be scalars", key)
			continue
		}
		out = append(out, c.Value)
	}
	return out
}

func optFloat(nodes map[string]*yaml.Node, key string, add addFunc) *float64 {
	s := optScalar(nodes, key, add)
	if s == nil {
		return nil
	}
	f, err := strconv.ParseFloat(*s, 64)
	if err != nil {
		add("field %q: invalid number %q", key, *s)
		return nil
	}
	return &f
}

func reqTimestamp(nodes map[string]*yaml.Node, key string, add addFunc) string {
	s := reqScalar(nodes, key, add)
	if s == "" {
		return ""
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		add("field %q: invalid RFC3339 timestamp %q", key, s)
	}
	return s
}
