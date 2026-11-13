// Package codec parses and serializes kira items to and from their on-disk Markdown-with-frontmatter form.
package codec

import (
	"fmt"
	"maps"
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
	it.Type = itemType(nodes, add)
	it.Subtype = optScalar(nodes, datamodel.KeySubtype, add)
	it.Title = reqScalar(nodes, datamodel.KeyTitle, add)
	it.State = reqScalar(nodes, datamodel.KeyState, add)
	it.Resolution = optScalar(nodes, datamodel.KeyResolution, add)
	it.Priority = optScalar(nodes, datamodel.KeyPriority, add)
	it.Rank = optScalar(nodes, datamodel.KeyRank, add)
	it.Owner = optScalar(nodes, datamodel.KeyOwner, add)
	it.Reporter = optScalar(nodes, datamodel.KeyReporter, add)
	it.Labels = reqList(nodes, datamodel.KeyLabels, add)
	it.Parent = parentField(nodes, add)
	it.BlockedBy = reqList(nodes, datamodel.KeyBlockedBy, add)
	it.Links = optLinks(nodes, datamodel.KeyLinks, add)
	it.Sprint = optScalar(nodes, datamodel.KeySprint, add)
	it.Due = optScalar(nodes, datamodel.KeyDue, add)
	it.Estimate = optFloat(nodes, datamodel.KeyEstimate, add)
	it.Created = reqTimestamp(nodes, datamodel.KeyCreated, add)
	it.Updated = reqTimestamp(nodes, datamodel.KeyUpdated, add)

	rejectUnknownKeys(nodes, add)

	if len(errs) > 0 {
		return it, &datamodel.ParseError{Errs: errs}
	}
	return it, nil
}

// DecodeFrontmatter decodes a partial document Parse itself would reject (an
// editor draft or template) into out, refusing any key an item file could not
// carry.
func DecodeFrontmatter(content string, out any) (body string, err error) {
	front, body, err := splitDocument(content)
	if err != nil {
		return "", err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(front), &doc); err != nil {
		return "", fmt.Errorf("frontmatter yaml: %w", err)
	}
	if len(doc.Content) == 0 {
		return body, nil
	}
	if err := doc.Decode(out); err != nil {
		return "", fmt.Errorf("frontmatter yaml: %w", err)
	}
	nodes, _ := frontmatterNodes(&doc)
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	rejectUnknownKeys(nodes, add)
	rejectUnknownTypeNames(nodes, add)
	if len(errs) > 0 {
		return "", &datamodel.ParseError{Errs: errs}
	}
	return body, nil
}

func rejectUnknownKeys(nodes map[string]*yaml.Node, add addFunc) {
	for _, k := range slices.Sorted(maps.Keys(nodes)) {
		if !datamodel.IsFrontmatterKey(k) && k != legacyEpicKey {
			add("field %q: unknown frontmatter key", k)
		}
	}
}

// rejectUnknownTypeNames checks the type and link-type names a partial
// document supplies, which Parse checks while reading those fields.
func rejectUnknownTypeNames(nodes map[string]*yaml.Node, add addFunc) {
	if n, ok := nodes[datamodel.KeyType]; ok && n.Kind == yaml.ScalarNode {
		checkItemType(n.Value, add)
	}
	if n, ok := nodes[datamodel.KeyLinks]; ok && n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			knownLinkType(datamodel.KeyLinks, n.Content[i].Value, add)
		}
	}
}

func itemType(nodes map[string]*yaml.Node, add addFunc) string {
	typ := reqScalar(nodes, datamodel.KeyType, add)
	checkItemType(typ, add)
	return typ
}

func checkItemType(typ string, add addFunc) {
	if typ != "" && !slices.Contains(datamodel.ItemTypes, typ) {
		add("field %q: unknown type %q", datamodel.KeyType, typ)
	}
}

func knownLinkType(key, typ string, add addFunc) bool {
	if datamodel.ValidLinkType(typ) {
		return true
	}
	add("field %q: unknown link type %q", key, typ)
	return false
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
	front, body, err := splitDocument(doc)
	if err != nil {
		return "", "", false
	}
	return FenceLine + front + FenceLine, body, true
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

// legacyEpicKey is the frontmatter key `parent:` was renamed from. Read
// support for it is permanent: this repo's own tickets, and any item file
// written before the rename, carry `epic:` on disk forever.
const legacyEpicKey = "epic"

// parentField reads the parent pointer, accepting legacyEpicKey as an alias
// for datamodel.KeyParent. A file carrying both is rejected rather than
// silently preferring one.
func parentField(nodes map[string]*yaml.Node, add addFunc) *string {
	_, hasParent := nodes[datamodel.KeyParent]
	_, hasEpic := nodes[legacyEpicKey]
	switch {
	case hasParent && hasEpic:
		add("field %q: cannot be combined with legacy field %q", datamodel.KeyParent, legacyEpicKey)
		return nil
	case hasEpic:
		return reqNullableScalar(nodes, legacyEpicKey, add)
	default:
		return reqNullableScalar(nodes, datamodel.KeyParent, add)
	}
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

func optLinks(nodes map[string]*yaml.Node, key string, add addFunc) map[string][]string {
	n, ok := nodes[key]
	if !ok || isNull(n) {
		return nil
	}
	if n.Kind != yaml.MappingNode {
		add("field %q: expected a map of link type to id list", key)
		return nil
	}
	var links map[string][]string
	for i := 0; i+1 < len(n.Content); i += 2 {
		typ, val := n.Content[i].Value, n.Content[i+1]
		if !knownLinkType(key, typ, add) {
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
	return links
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
