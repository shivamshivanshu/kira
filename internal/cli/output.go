package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func emitWarningLines(w io.Writer, warns []string) {
	for _, m := range warns {
		fmt.Fprintln(w, msgPrefix, "warning:", m)
	}
}

func emitMutationWarnings(w io.Writer, warns []datamodel.Warning) {
	rendered := make([]string, len(warns))
	for i, wn := range warns {
		rendered[i] = renderWarning(wn)
	}
	emitWarningLines(w, rendered)
}

func renderWarning(w datamodel.Warning) string {
	switch w.Code {
	case datamodel.WarnIndexFallback:
		return "index unavailable, using linear scan"
	case datamodel.WarnNoActiveSprint:
		return "no active sprint set; sprint=active matches nothing (run `kira sprint activate <key>`)"
	case datamodel.WarnCloseUnknown:
		return fmt.Sprintf("unknown ticket %s in %s", w.Args[0], w.Args[1])
	case datamodel.WarnCloseFailed:
		return fmt.Sprintf("failed to close %s: %s", w.Args[0], w.Args[1])
	case datamodel.WarnLiteral:
		return w.Args[0]
	case datamodel.WarnOrphanType:
		return fmt.Sprintf("%s has no workflow for type %q; it is read-only until a workflow is configured", w.Args[0], w.Args[1])
	}
	return ""
}

func emitJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func newTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
}
