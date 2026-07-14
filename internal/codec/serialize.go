package codec

import (
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
