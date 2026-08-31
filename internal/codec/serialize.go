package codec

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func Serialize(it *datamodel.Item) string {
	var b strings.Builder
	b.Grow(256 + len(it.Body))
	b.WriteString(FenceLine)

	writeLine(&b, datamodel.KeyID, EmitScalar(it.ID))
	writeLine(&b, datamodel.KeyNumber, EmitScalar(it.Number))
	writeLine(&b, datamodel.KeyAliases, EmitList(it.Aliases))
	writeLine(&b, datamodel.KeyType, EmitScalar(it.Type))
	writeOptScalar(&b, datamodel.KeySubtype, it.Subtype)
	writeLine(&b, datamodel.KeyTitle, emitQuoted(it.Title))
	writeLine(&b, datamodel.KeyState, EmitScalar(it.State))
	writeOptScalar(&b, datamodel.KeyResolution, it.Resolution)
	writeOptScalar(&b, datamodel.KeyPriority, it.Priority)
	writeOptScalar(&b, datamodel.KeyRank, it.Rank)
	writeOptScalar(&b, datamodel.KeyOwner, it.Owner)
	writeOptScalar(&b, datamodel.KeyReporter, it.Reporter)
	writeLine(&b, datamodel.KeyLabels, EmitList(it.Labels))
	writeReqNullableScalar(&b, datamodel.KeyEpic, it.Epic)
	writeLine(&b, datamodel.KeyBlockedBy, EmitList(it.BlockedBy))
	writeLinks(&b, it.Links)
	writeOptScalar(&b, datamodel.KeySprint, it.Sprint)
	if it.Due != nil {
		writeLine(&b, datamodel.KeyDue, emitDate(*it.Due))
	}
	writeOptFloat(&b, datamodel.KeyEstimate, it.Estimate)
	writeLine(&b, datamodel.KeyCreated, emitTimestamp(it.Created))
	writeLine(&b, datamodel.KeyUpdated, emitTimestamp(it.Updated))
	WriteExtras(&b, it.Extras)

	b.WriteString(FenceLine)
	b.WriteString(it.Body)
	return b.String()
}

func writeLine(b *strings.Builder, key, value string) {
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteByte('\n')
}

func writeOptScalar(b *strings.Builder, key string, v *string) {
	if v != nil {
		writeLine(b, key, EmitScalar(*v))
	}
}

func writeReqNullableScalar(b *strings.Builder, key string, v *string) {
	if v == nil {
		writeLine(b, key, "null")
		return
	}
	writeLine(b, key, EmitScalar(*v))
}

func writeOptFloat(b *strings.Builder, key string, v *float64) {
	if v != nil {
		writeLine(b, key, EmitFloat(*v))
	}
}

func WriteExtras(b *strings.Builder, extras map[string]datamodel.ExtraValue) {
	for _, key := range slices.Sorted(maps.Keys(extras)) {
		v := extras[key]
		if v.List != nil {
			// elements hold renderFlowElem output: already flow-safe YAML text
			writeLine(b, EmitScalar(key), "["+strings.Join(v.List, ", ")+"]")
			continue
		}
		if v.Scalar != nil {
			writeLine(b, EmitScalar(key), *v.Scalar)
		}
	}
}

func writeLinks(b *strings.Builder, links map[string][]string) {
	first := true
	for _, typ := range datamodel.LinkTypes {
		targets := links[string(typ)]
		if len(targets) == 0 {
			continue
		}
		if first {
			b.WriteString(datamodel.KeyLinks + ":\n")
			first = false
		}
		b.WriteString("  ")
		writeLine(b, string(typ), EmitList(targets))
	}
}

func EmitFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func EmitList(xs []string) string {
	if len(xs) == 0 {
		return "[]"
	}
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = emitFlowScalar(x)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func EmitScalar(s string) string {
	if plainSafe(s) {
		return s
	}
	if plain, err := marshalScalar(s); err == nil && plain == s {
		return s
	}
	return emitQuoted(s)
}

func DecodeScalar(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || (trimmed[0] != '"' && trimmed[0] != '\'') {
		return trimmed
	}
	var v string
	if err := yaml.Unmarshal([]byte(trimmed), &v); err != nil {
		return trimmed
	}
	return v
}

func DecodeList(elems []string) []string {
	out := make([]string, len(elems))
	for i, e := range elems {
		out[i] = DecodeScalar(e)
	}
	return out
}

// DecodeExtras renders captured extras for --json output: scalars as decoded
// strings, lists as decoded string arrays; nil when the item has none.
func DecodeExtras(extras map[string]datamodel.ExtraValue) map[string]datamodel.ExtraJSON {
	if len(extras) == 0 {
		return nil
	}
	out := make(map[string]datamodel.ExtraJSON, len(extras))
	for k, v := range extras {
		switch {
		case v.List != nil:
			out[k] = DecodeList(v.List)
		case v.Scalar != nil:
			out[k] = DecodeScalar(*v.Scalar)
		}
	}
	return out
}

// RenderScalar is RenderScalarNode for plain text, giving `--field` writes the
// same canonical form parsing produces: an untagged node resolves its implicit
// tag from the text, so "42" stays an unquoted int rather than EmitScalar's
// string-typed "42". The result is re-decoded and rendered once more so it is
// a fixed point of parse-capture (a bare date, for example, re-renders quoted
// under its implicit !!timestamp tag). Null forms are refused because
// extraValue would drop them on the next parse.
func RenderScalar(s string) (string, bool) {
	out, node, ok := renderAndDecode(s, RenderScalarNode)
	if !ok || isNull(node) {
		return "", false
	}
	return out, true
}

// RenderFlowElem is RenderScalar inside a [] context (see renderFlowElem).
// Nulls pass through: extraValue keeps null list elements.
func RenderFlowElem(s string) (string, bool) {
	out, _, ok := renderAndDecode(s, renderFlowElem)
	return out, ok
}

func renderAndDecode(s string, render func(*yaml.Node) (string, bool)) (string, *yaml.Node, bool) {
	out, ok := render(&yaml.Node{Kind: yaml.ScalarNode, Value: s})
	if !ok {
		return "", nil, false
	}
	var doc yaml.Node
	if yaml.Unmarshal([]byte(out), &doc) != nil || len(doc.Content) == 0 {
		return "", nil, false
	}
	out, ok = render(doc.Content[0])
	return out, doc.Content[0], ok
}

func emitFlowScalar(s string) string {
	if plainSafe(s) {
		return s
	}
	if out := EmitScalar(s); out != s {
		return out
	}
	seq := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Style:   yaml.FlowStyle,
		Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: s}},
	}
	out, err := marshalScalar(seq)
	if err != nil {
		return emitQuoted(s)
	}
	if out == "["+s+"]" {
		return s
	}
	return strings.TrimSuffix(strings.TrimPrefix(out, "["), "]")
}

func plainSafe(s string) bool {
	if s == "" || !asciiLetter(s[0]) || yamlKeyword(s) {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !asciiLetter(c) && !asciiDigit(c) && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func asciiLetter(c byte) bool { return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' }

func asciiDigit(c byte) bool { return '0' <= c && c <= '9' }

func yamlKeyword(s string) bool {
	switch strings.ToLower(s) {
	case "null", "true", "false", "yes", "no", "on", "off", "y", "n":
		return true
	}
	return false
}

func emitQuoted(s string) string {
	out, err := marshalScalar(&yaml.Node{Kind: yaml.ScalarNode, Style: yaml.DoubleQuotedStyle, Value: s})
	if err != nil {
		return strconv.Quote(s)
	}
	return out
}

// verbatim, never EmitScalar: yaml.v3 tags an unquoted RFC3339 scalar as
// implicit !!timestamp, so re-marshaling would double-quote it and break the
// unquoted canonical form.
func emitTimestamp(s string) string { return s }

// RenderScalarNode returns a scalar node's canonical YAML text, preserving the
// implicit tag its style carries: an extra holds rendered text rather than a
// decoded value so that `points: 42` survives a rewrite as an int instead of
// being requoted to "42". Readers decode with DecodeScalar. Comments and
// anchors are stripped -- Serialize drops them from builtin fields too, and a
// head comment would otherwise force multi-line output. Reports false for
// anything that will not fit one frontmatter line (literal/folded styles).
func RenderScalarNode(n *yaml.Node) (string, bool) {
	out, err := marshalScalar(bareValue(n))
	if err != nil || strings.ContainsAny(out, "\n\r") {
		return "", false
	}
	return out, true
}

// renderFlowElem is RenderScalarNode for a list element, rendered inside a []
// context so the emitter quotes `,`/`]`-bearing values a bare render would
// leave plain.
func renderFlowElem(n *yaml.Node) (string, bool) {
	if n.Kind != yaml.ScalarNode {
		return "", false
	}
	seq := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle, Content: []*yaml.Node{bareValue(n)}}
	out, err := marshalScalar(seq)
	if err != nil || strings.ContainsAny(out, "\n\r") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(out, "["), "]"), true
}

func bareValue(n *yaml.Node) *yaml.Node {
	clean := *n
	clean.HeadComment, clean.LineComment, clean.FootComment, clean.Anchor = "", "", "", ""
	return &clean
}

// valid dates go verbatim for the same !!timestamp reason as emitTimestamp;
// invalid ones (due is parsed shape-only) still need EmitScalar's quoting to
// stay parseable YAML.
func emitDate(s string) string {
	if datamodel.ValidDate(s) {
		return s
	}
	return EmitScalar(s)
}

func marshalScalar(v any) (string, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}
