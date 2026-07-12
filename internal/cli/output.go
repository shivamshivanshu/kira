package cli

import (
	"encoding/json"
	"io"
)

// emitJSON writes v as indented JSON with a trailing newline, HTML escaping
// off so URLs and quotes in titles stay literal. Deterministic output is what
// the golden --json contract tests depend on (docs/design/04-cli.md §7).
func emitJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
