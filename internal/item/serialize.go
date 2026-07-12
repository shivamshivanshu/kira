package item

import (
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Serialize renders the item as a canonical file: opening fence, one key per
// line in schema order (block-style frontmatter, never flow mapping), closing
// fence, then the body verbatim. Parse and Serialize are inverses on canonical
// input, and editing one scalar changes exactly one frontmatter line.
func (it *Item) Serialize() string {
	var b strings.Builder
	b.Grow(256 + len(it.Body))
	b.WriteString(fenceLine)

	writeLine(&b, keyID, emitScalar(it.ID))
	writeLine(&b, keyNumber, emitScalar(it.Number))
	writeLine(&b, keyAliases, emitList(it.Aliases))
	writeLine(&b, keyType, emitScalar(it.Type))
	writeLine(&b, keyTitle, emitQuoted(it.Title))
	writeLine(&b, keyState, emitScalar(it.State))
	writeOptScalar(&b, keyPriority, it.Priority)
	writeOptScalar(&b, keyOwner, it.Owner)
	writeOptScalar(&b, keyReporter, it.Reporter)
	writeLine(&b, keyLabels, emitList(it.Labels))
	epic := "null" // required but nullable: always written, unlike the omit-if-nil optionals
	if it.Epic != nil {
		epic = emitScalar(*it.Epic)
	}
	writeLine(&b, keyEpic, epic)
	writeLine(&b, keyBlockedBy, emitList(it.BlockedBy))
	writeOptFloat(&b, keyEstimate, it.Estimate)
	writeLine(&b, keyCreated, emitTimestamp(it.Created))
	writeLine(&b, keyUpdated, emitTimestamp(it.Updated))

	b.WriteString(fenceLine)
	b.WriteString(it.Body)
	return b.String()
}

func writeLine(b *strings.Builder, key, value string) {
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteByte('\n')
}

// writeOptScalar writes an optional scalar line, or nothing when the field is
// absent (nil). Epic is not routed here: it is required and writes `null`.
func writeOptScalar(b *strings.Builder, key string, v *string) {
	if v != nil {
		writeLine(b, key, emitScalar(*v))
	}
}

func writeOptFloat(b *strings.Builder, key string, v *float64) {
	if v != nil {
		writeLine(b, key, EmitFloat(*v))
	}
}

// EmitFloat renders a float in kira's canonical numeric form (shortest decimal
// that round-trips). Exported so core's draft serializer emits estimate
// identically to a full item file.
func EmitFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// EmitScalar renders a scalar as canonical YAML: plain when it round-trips
// plainly, double-quoted otherwise. Exported for core's draft serializer, which
// must emit editable scalar fields identically to a full item file.
func EmitScalar(s string) string { return emitScalar(s) }

// EmitList renders a string slice as a canonical flow sequence (`[]` when
// empty). Exported for core's draft serializer.
func EmitList(xs []string) string { return emitList(xs) }

// emitList renders a flow sequence: `[]` when empty, else `[a, b, c]` — matching
// the canonical example in docs/design/02-data-model.md §8.
func emitList(xs []string) string {
	if len(xs) == 0 {
		return "[]"
	}
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = emitScalar(x)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// emitScalar renders a scalar as plain text when YAML can round-trip it plainly,
// otherwise double-quoted. This keeps ULIDs, KIRA-numbers, states and RFC3339
// timestamps unquoted (as the canonical example has them) while safely quoting
// anything ambiguous (empty, reserved words like "null"/"true", numeric-looking
// strings, values with structural characters).
func emitScalar(s string) string {
	if plain, err := marshalScalar(s); err == nil && plain == s {
		return s
	}
	return emitQuoted(s)
}

// emitQuoted forces a double-quoted scalar with correct YAML escaping. Used
// unconditionally for title (free-form text, always quoted in the canonical
// example) and as emitScalar's fallback.
func emitQuoted(s string) string {
	out, err := marshalScalar(&yaml.Node{Kind: yaml.ScalarNode, Style: yaml.DoubleQuotedStyle, Value: s})
	if err != nil {
		return strconv.Quote(s) // unreachable for scalars; keeps output valid
	}
	return out
}

// emitTimestamp emits an RFC3339 timestamp verbatim, deliberately NOT via
// emitScalar. yaml.v3 tags an unquoted RFC3339 scalar as implicit !!timestamp,
// so re-marshaling it would double-quote the value and break the unquoted
// canonical form (docs/design/02-data-model.md §8). Raw is safe because
// reqTimestamp reads the bare scalar text back unchanged; any future change
// that routes timestamps through emitScalar must preserve this invariant.
func emitTimestamp(s string) string { return s }

// marshalScalar YAML-encodes a single scalar value and strips the trailing
// newline yaml.Marshal always appends.
func marshalScalar(v any) (string, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}
