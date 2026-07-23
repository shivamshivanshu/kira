package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newBlameCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "blame <id>",
		Short: "Show, per field, its current value and the commit that last set it",
		Args:  cobra.ExactArgs(1),
		RunE: storeActionRunE(g,
			func(s *core.Store, cfg *datamodel.Config, args []string) (*datamodel.BlameResult, error) {
				return s.Blame(cfg, args[0])
			},
			renderBlame),
	}
}

func renderBlame(w io.Writer, res *datamodel.BlameResult) {
	tw := newTabWriter(w)
	_, _ = fmt.Fprintln(tw, "FIELD\tVALUE\tWHEN\tBY\tSOURCE")
	for _, f := range res.Fields {
		source := f.SourceKind
		if f.Degraded {
			source += " (degraded)"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", f.Field, f.Value, f.When, f.By, source)
	}
	_ = tw.Flush()
}
