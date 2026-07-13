package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newDiffCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "diff <ref>",
		Short: "Show the semantic backlog change from merge-base to <ref>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Diff(args[0])
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderDiff(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func renderDiff(w io.Writer, res *datamodel.DiffResult) {
	if len(res.Items) == 0 {
		fmt.Fprintln(w, "no backlog differences")
		return
	}
	for _, it := range res.Items {
		switch it.Status {
		case datamodel.DiffCreated:
			fmt.Fprintf(w, "created %s  %s\n", it.Number, it.Title)
		case datamodel.DiffDeleted:
			fmt.Fprintf(w, "deleted %s  %s\n", it.Number, it.Title)
		default:
			fmt.Fprintf(w, "%s  %s\n", it.Number, it.Title)
		}
		if it.Renumbered != nil {
			fmt.Fprintf(w, "  renumbered %s -> %s\n", it.Renumbered.From, it.Renumbered.To)
		}
		for _, c := range it.Changes {
			if c.Field == datamodel.KeyBody {
				fmt.Fprintf(w, "  body edited (%s)\n", c.To)
				continue
			}
			fmt.Fprintf(w, "  %s: %s -> %s\n", c.Field, c.From, c.To)
		}
	}
}
