package codec

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const (
	FenceLine  = "---\n"
	closeFence = "\n" + FenceLine
)

func Parse(content string) (*datamodel.Item, error) {
	front, body, err := splitDocument(content)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(front), &doc); err != nil {
		return nil, fmt.Errorf("frontmatter yaml: %w", err)
	}

	nodes := frontmatterNodes(&doc)
	it := &datamodel.Item{Body: body}
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	it.ID = reqScalar(nodes, datamodel.KeyID, add)
	it.Number = reqScalar(nodes, datamodel.KeyNumber, add)
	it.Aliases = reqList(nodes, datamodel.KeyAliases, add)
	it.Type = reqScalar(nodes, datamodel.KeyType, add)
	if it.Type != "" && !datamodel.ValidType(it.Type) {
		add("field %q: must be %s or %s, got %q", datamodel.KeyType, datamodel.TypeTicket, datamodel.TypeEpic, it.Type)
	}
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

	for k := range nodes {
		if !datamodel.IsFrontmatterKey(k) {
			it.UnknownKeys = append(it.UnknownKeys, k)
		}
	}
	slices.Sort(it.UnknownKeys)

	if len(errs) > 0 {
		return it, &datamodel.ParseError{Errs: errs}
	}
	return it, nil
}

func DecodeFrontmatter(content string, out any) (body string, err error) {
	front, body, err := splitDocument(content)
	if err != nil {
		return "", err
	}
	if err := yaml.Unmarshal([]byte(front), out); err != nil {
		return "", fmt.Errorf("frontmatter yaml: %w", err)
	}
	return body, nil
}

func splitDocument(content string) (front, body string, err error) {
	if !strings.HasPrefix(content, FenceLine) {
		return "", "", fmt.Errorf("missing opening frontmatter fence")
	}
	rest := content[len(FenceLine):]
	if i := strings.Index(rest, closeFence); i >= 0 {
		return rest[:i+1], rest[i+len(closeFence):], nil
	}
	return "", "", fmt.Errorf("missing closing frontmatter fence")
}

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
