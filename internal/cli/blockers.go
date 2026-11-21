package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newBlockersCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "blockers <ticket-or-epic>",
		Short: "Find actionable blockers across dependency chains",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			result, err := store.Blockers(cfg, args[0])
			if err != nil {
				return err
			}
			emitStderrNotes(cmd.ErrOrStderr(), result.StderrNotes)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), result)
			}
			renderBlockers(cmd.OutOrStdout(), result)
			return nil
		},
	}
}

func renderBlockers(w io.Writer, result *datamodel.BlockersResult) {
	_, _ = fmt.Fprintf(w, "%s  %s  [%s]\n", result.Item.Number, result.Item.Title, result.Item.State)
	if len(result.Actionable)+len(result.Direct)+len(result.Incomplete) == 0 {
		_, _ = fmt.Fprintln(w, "no open blockers")
		return
	}
	if len(result.Actionable) > 0 {
		_, _ = fmt.Fprintln(w, "Actionable blockers")
		for _, blocker := range result.Actionable {
			_, _ = fmt.Fprintf(w, "  %s  %s  [affected: %d]\n    %s\n",
				blocker.Item.Number, blocker.Item.Title, blocker.Affected, strings.Join(blocker.Path, " -> "))
		}
	}
	if len(result.Direct) > 0 {
		_, _ = fmt.Fprintln(w, "Actionable blockers of the epic itself")
		for _, blocker := range result.Direct {
			_, _ = fmt.Fprintf(w, "  %s  %s\n    %s\n",
				blocker.Item.Number, blocker.Item.Title, strings.Join(blocker.Path, " -> "))
		}
	}
	if len(result.Incomplete) > 0 {
		_, _ = fmt.Fprintln(w, "Incomplete blocker chains")
		for _, issue := range result.Incomplete {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", strings.Join(issue.Path, " -> "), issue.Reason)
		}
	}
}
