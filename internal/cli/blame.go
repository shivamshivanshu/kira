package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newBlameCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "blame <id>",
		Short: "Show, per field, its current value and the commit that last set it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Blame(cfg, args[0])
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderBlame(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func renderBlame(w io.Writer, res *datamodel.BlameResult) {
	tw := newTabWriter(w)
	fmt.Fprintln(tw, "FIELD\tVALUE\tWHEN\tBY\tSOURCE")
	for _, f := range res.Fields {
		source := f.SourceKind
		if f.Degraded {
			source += " (degraded)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", f.Field, f.Value, f.When, f.By, source)
	}
	tw.Flush()
}
