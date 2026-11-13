package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/warningfmt"
)

func emitWarningLines(w io.Writer, warns []string) {
	for _, m := range warns {
		_, _ = fmt.Fprintln(w, msgPrefix, "warning:", m)
	}
}

func emitMutationWarnings(w io.Writer, warns []datamodel.Warning) {
	rendered := make([]string, len(warns))
	for i, wn := range warns {
		rendered[i] = warningfmt.Message(wn)
	}
	emitWarningLines(w, rendered)
}

func warnMutation(w io.Writer, res *datamodel.MutationResult) { emitMutationWarnings(w, res.Warnings) }

func emitJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
