// Package codec parses and serializes kira items to and from their on-disk Markdown-with-frontmatter form.
package codec

import (
	"fmt"
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
	it.Links, it.ExtraLinks = optLinks(nodes, datamodel.KeyLinks, add)
	it.Sprint = optScalar(nodes, datamodel.KeySprint, add)
	it.Due = optScalar(nodes, datamodel.KeyDue, add)
	it.Estimate = optFloat(nodes, datamodel.KeyEstimate, add)
	it.Created = reqTimestamp(nodes, datamodel.KeyCreated, add)
	it.Updated = reqTimestamp(nodes, datamodel.KeyUpdated, add)

	it.ExtraScalars, it.ExtraLists = captureExtras(nodes, add)

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

// ExtraFrontmatter reports content's non-builtin frontmatter in the shape
// Parse captures it, for a partial document Parse itself would reject (an
// editor draft). A value that is neither a scalar nor a scalar list is
// reported as a ParseError rather than dropped.
func ExtraFrontmatter(content string) (scalars map[string]datamodel.Rendered, lists map[string][]datamodel.Rendered, err error) {
	front, _, err := splitDocument(content)
	if err != nil {
		return nil, nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(front), &doc); err != nil {
		return nil, nil, fmt.Errorf("frontmatter yaml: %w", err)
	}
	if len(doc.Content) > 0 && doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("frontmatter: expected a mapping")
	}
	nodes, _ := frontmatterNodes(&doc)
	var errs []error
	scalars, lists = captureExtras(nodes, func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	})
	if len(errs) > 0 {
		return scalars, lists, &datamodel.ParseError{Errs: errs}
	}
	return scalars, lists, nil
}

func captureExtras(nodes map[string]*yaml.Node, add addFunc) (scalars map[string]datamodel.Rendered, lists map[string][]datamodel.Rendered) {
	for k, n := range nodes {
		if datamodel.IsFrontmatterKey(k) {
			continue
		}
		switch n.Kind {
		case yaml.ScalarNode:
			if scalars == nil {
				scalars = map[string]datamodel.Rendered{}
			}
			scalars[k] = renderExtraScalar(n)
		case yaml.SequenceNode:
			if lists == nil {
				lists = map[string][]datamodel.Rendered{}
			}
			lists[k] = extraScalarSeq(n, fmt.Sprintf("field %q", k), add)
		default:
			add("field %q: unsupported extra field shape, expected a scalar or list", k)
		}
	}
	return scalars, lists
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
	if _, b, found := strings.Cut(rest, closeFence); found {
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

// renderExtraScalar computes the faithful single-line YAML rendering of an
// extra scalar node, to be emitted verbatim (never re-decoded through
// EmitScalar, which would treat every value as a string and quote it).
func renderExtraScalar(n *yaml.Node) datamodel.Rendered {
	return datamodel.Rendered(renderExtra(n, EmitScalar))
}

// renderExtraListElement is renderExtraScalar for an element of a flow list,
// where a plain scalar holding a comma or a bracket would split the list.
func renderExtraListElement(n *yaml.Node) datamodel.Rendered {
	return datamodel.Rendered(renderExtra(n, emitFlowScalar))
}

func renderExtra(n *yaml.Node, emit func(string) string) string {
	var s string
	switch n.Tag {
	case "!!null":
		if n.Value == "" {
			s = "null"
		} else {
			s = n.Value
		}
	case "!!bool", "!!int", "!!float", "!!timestamp":
		s = n.Value
	default:
		s = emit(n.Value)
	}
	if strings.ContainsRune(s, '\n') {
		s = emit(n.Value)
	}
	return s
}

// DecodeExtraScalar recovers the Go value (int/bool/float64/string/nil) a
// faithfully-rendered extra scalar represents, for callers that need it typed
// rather than as its raw on-disk text.
// Decoder satisfies entityschema.ExtraDecoder. It carries no state; it exists
// so callers can hand the decoding seam across a package boundary that must not
// import codec.
type Decoder struct{}

func (Decoder) DecodeExtraScalar(s datamodel.Rendered) any { return DecodeExtraScalar(s) }

func (Decoder) DecodeExtraList(stored []datamodel.Rendered) []string { return DecodeExtraList(stored) }

// DecodeExtraList recovers the displayable values a stored extra list holds.
func DecodeExtraList(stored []datamodel.Rendered) []string {
	out := make([]string, len(stored))
	for i, v := range stored {
		out[i] = DecodeScalar(v)
	}
	return out
}

func DecodeExtraScalar(s datamodel.Rendered) any {
	var v any
	if err := yaml.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	return v
}

// EncodeExtraScalar renders a typed Go value to on-disk scalar form, quoting a
// string only where needed so it isn't read back as another type (a date-shaped
// string, or "5").
func EncodeExtraScalar(v any) (datamodel.Rendered, error) {
	var n yaml.Node
	if err := n.Encode(v); err != nil {
		return "", err
	}
	return renderExtraScalar(&n), nil
}

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
	return mapScalarSeq(n, label, add, func(c *yaml.Node) string { return c.Value })
}

// extraScalarSeq captures a list extra as each element's faithful YAML text,
// the way renderExtraScalar captures a scalar extra, so an element's type and
// quoting survive a rewrite of the item.
func extraScalarSeq(n *yaml.Node, label string, add addFunc) []datamodel.Rendered {
	return mapScalarSeq(n, label, add, renderExtraListElement)
}

func mapScalarSeq[T ~string](n *yaml.Node, label string, add addFunc, render func(*yaml.Node) T) []T {
	out := make([]T, 0, len(n.Content))
	for _, c := range n.Content {
		if c.Kind != yaml.ScalarNode {
			add("%s: list elements must be scalars", label)
			continue
		}
		out = append(out, render(c))
	}
	return out
}

func optLinks(nodes map[string]*yaml.Node, key string, add addFunc) (links, extraLinks map[string][]string) {
	n, ok := nodes[key]
	if !ok || isNull(n) {
		return nil, nil
	}
	if n.Kind != yaml.MappingNode {
		add("field %q: expected a map of link type to id list", key)
		return nil, nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		typ, val := n.Content[i].Value, n.Content[i+1]
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
		if datamodel.ValidLinkType(typ) {
			if links == nil {
				links = make(map[string][]string, len(datamodel.LinkTypes))
			}
			links[typ] = targets
		} else {
			if extraLinks == nil {
				extraLinks = map[string][]string{}
			}
			extraLinks[typ] = targets
		}
	}
	return links, extraLinks
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
