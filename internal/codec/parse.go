// Package codec parses and serializes kira items to and from their on-disk Markdown-with-frontmatter form.
package codec

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

const (
	FenceLine  = "---\n"
	closeFence = "\n" + FenceLine
)

func Parse(content string) (*datamodel.Item, error) {
	crlf := strings.Contains(content, "\r\n")
	front, body, err := splitDocument(content)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(front), &doc); err != nil {
		return nil, fmt.Errorf("frontmatter yaml: %w", err)
	}

	if len(doc.Content) > 0 && doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("frontmatter: expected a mapping")
	}

	nodes, duplicates := frontmatterNodes(&doc)
	it := &datamodel.Item{Body: body, CRLF: crlf}
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	for _, key := range duplicates {
		add("field %q: duplicate key", key)
	}

	it.ID = reqULID(nodes, datamodel.KeyID, add)
	it.Number = reqScalar(nodes, datamodel.KeyNumber, add)
	it.Aliases = reqList(nodes, datamodel.KeyAliases, add)
	it.Type = reqScalar(nodes, datamodel.KeyType, add)
	it.Subtype = optScalar(nodes, datamodel.KeySubtype, add)
	it.Title = reqScalar(nodes, datamodel.KeyTitle, add)
	it.State = reqScalar(nodes, datamodel.KeyState, add)
	it.Resolution = optScalar(nodes, datamodel.KeyResolution, add)
	it.Priority = optScalar(nodes, datamodel.KeyPriority, add)
	it.Rank = optScalar(nodes, datamodel.KeyRank, add)
	it.Owner = optScalar(nodes, datamodel.KeyOwner, add)
	it.Reporter = optScalar(nodes, datamodel.KeyReporter, add)
	it.Labels = reqList(nodes, datamodel.KeyLabels, add)
	it.Epic = reqNullableScalar(nodes, datamodel.KeyEpic, add)
	it.BlockedBy = reqList(nodes, datamodel.KeyBlockedBy, add)
	it.Links, it.UnknownLinkTypes = optLinks(nodes, datamodel.KeyLinks, add)
	it.Sprint = optScalar(nodes, datamodel.KeySprint, add)
	it.Due = optScalar(nodes, datamodel.KeyDue, add)
	it.Estimate = optFloat(nodes, datamodel.KeyEstimate, add)
	it.Created = reqTimestamp(nodes, datamodel.KeyCreated, add)
	it.Updated = reqTimestamp(nodes, datamodel.KeyUpdated, add)

	it.Extras, it.UnknownKeys = captureExtras(nodes)

	if len(errs) > 0 {
		return it, &datamodel.ParseError{Errs: errs}
	}
	return it, nil
}

// DecodeFrontmatter decodes content's frontmatter into out and captures
// non-builtin keys the way Parse does: capturable values land in extras, and
// every non-builtin key — capturable or not — is returned sorted in unknown.
func DecodeFrontmatter(content string, out any) (body string, extras map[string]datamodel.ExtraValue, unknown []string, err error) {
	front, body, err := splitDocument(content)
	if err != nil {
		return "", nil, nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(front), &doc); err != nil {
		return "", nil, nil, fmt.Errorf("frontmatter yaml: %w", err)
	}
	if len(doc.Content) == 0 {
		return body, nil, nil, nil
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return "", nil, nil, fmt.Errorf("frontmatter: expected a mapping")
	}
	if err := doc.Decode(out); err != nil {
		return "", nil, nil, fmt.Errorf("frontmatter yaml: %w", err)
	}
	nodes, _ := frontmatterNodes(&doc)
	extras, unknown = captureExtras(nodes)
	return body, extras, unknown, nil
}

func splitDocument(content string) (front, body string, err error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	rest, ok := strings.CutPrefix(content, FenceLine)
	if !ok {
		return "", "", fmt.Errorf("missing opening frontmatter fence")
	}
	if body, ok := strings.CutPrefix(rest, FenceLine); ok {
		return "", body, nil
	}
	if i := strings.Index(rest, closeFence); i >= 0 {
		return rest[:i+1], rest[i+len(closeFence):], nil
	}
	if front, ok := strings.CutSuffix(rest, "\n---"); ok {
		return front + "\n", "", nil
	}
	if rest == "---" {
		return "", "", nil
	}
	return "", "", fmt.Errorf("missing closing frontmatter fence")
}

func cutAfterFrontmatter(doc string) (head, body string, ok bool) {
	doc = strings.ReplaceAll(doc, "\r\n", "\n")
	rest, found := strings.CutPrefix(doc, FenceLine)
	if !found {
		return "", "", false
	}
	if b, found := strings.CutPrefix(rest, FenceLine); found {
		return doc[:len(doc)-len(b)], b, true
	}
	if _, b, ok := strings.Cut(rest, closeFence); ok {
		return doc[:len(doc)-len(b)], b, true
	}
	if strings.HasSuffix(rest, "\n---") || rest == "---" {
		return doc + "\n", "", true
	}
	return "", "", false
}

func frontmatterNodes(doc *yaml.Node) (nodes map[string]*yaml.Node, duplicates []string) {
	if len(doc.Content) == 0 {
		return map[string]*yaml.Node{}, nil
	}
	m := doc.Content[0]
	nodes = make(map[string]*yaml.Node, len(m.Content)/2)
	for i := 0; i+1 < len(m.Content); i += 2 {
		key := m.Content[i].Value
		if _, seen := nodes[key]; seen {
			duplicates = append(duplicates, key)
		}
		nodes[key] = m.Content[i+1]
	}
	return nodes, duplicates
}

func isNull(n *yaml.Node) bool { return n.Kind == yaml.ScalarNode && n.Tag == "!!null" }

type addFunc func(format string, args ...any)

const (
	errMissing   = "field %q: required, missing"
	errNotScalar = "field %q: expected a scalar"
)

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

// reqULID rejects a non-ULID id outright: the id becomes the item's filename,
// so a value carrying path separators would write outside the store.
func reqULID(nodes map[string]*yaml.Node, key string, add addFunc) string {
	v := reqScalar(nodes, key, add)
	if v == "" {
		return ""
	}
	if _, err := id.ParseULIDLoose(v); err != nil {
		add("field %q: %q is not a valid ULID", key, v)
		return ""
	}
	return v
}

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

func reqNullableScalar(nodes map[string]*yaml.Node, key string, add addFunc) *string {
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
	return scalarSeq(n, fmt.Sprintf("field %q", key), add)
}

func scalarSeq(n *yaml.Node, label string, add addFunc) []string {
	out := make([]string, 0, len(n.Content))
	for _, c := range n.Content {
		if c.Kind != yaml.ScalarNode {
			add("%s: list elements must be scalars", label)
			continue
		}
		out = append(out, c.Value)
	}
	return out
}

func captureExtras(nodes map[string]*yaml.Node) (extras map[string]datamodel.ExtraValue, unknown []string) {
	for k, n := range nodes {
		if datamodel.IsFrontmatterKey(k) {
			continue
		}
		unknown = append(unknown, k)
		v, ok := extraValue(n)
		if !ok {
			continue
		}
		if extras == nil {
			extras = map[string]datamodel.ExtraValue{}
		}
		extras[k] = v
	}
	slices.Sort(unknown)
	return extras, unknown
}

// Mappings, null, and sequences with non-scalar elements are not captured:
// nested objects are a declared non-goal, so those stay in UnknownKeys only.
func extraValue(n *yaml.Node) (datamodel.ExtraValue, bool) {
	switch n.Kind {
	case yaml.ScalarNode:
		if isNull(n) {
			return datamodel.ExtraValue{}, false
		}
		v, ok := renderScalarNode(n)
		if !ok {
			return datamodel.ExtraValue{}, false
		}
		return datamodel.ExtraValue{Scalar: &v}, true
	case yaml.SequenceNode:
		vals := make([]string, 0, len(n.Content))
		for _, c := range n.Content {
			v, ok := renderFlowElem(c)
			if !ok {
				return datamodel.ExtraValue{}, false
			}
			vals = append(vals, v)
		}
		return datamodel.ExtraValue{List: vals}, true
	}
	return datamodel.ExtraValue{}, false
}

func optLinks(nodes map[string]*yaml.Node, key string, add addFunc) (map[string][]string, []string) {
	n, ok := nodes[key]
	if !ok || isNull(n) {
		return nil, nil
	}
	if n.Kind != yaml.MappingNode {
		add("field %q: expected a map of link type to id list", key)
		return nil, nil
	}
	var links map[string][]string
	var unknown []string
	for i := 0; i+1 < len(n.Content); i += 2 {
		typ, val := n.Content[i].Value, n.Content[i+1]
		if !datamodel.ValidLinkType(typ) {
			unknown = append(unknown, typ)
			continue
		}
		if isNull(val) {
			continue
		}
		if val.Kind != yaml.SequenceNode {
			add("field %q: %s: expected a list", key, typ)
			continue
		}
		targets := scalarSeq(val, fmt.Sprintf("field %q: %s", key, typ), add)
		if len(targets) == 0 {
			continue
		}
		if links == nil {
			links = make(map[string][]string, len(datamodel.LinkTypes))
		}
		links[typ] = targets
	}
	slices.Sort(unknown)
	return links, unknown
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
