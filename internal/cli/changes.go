package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newChangesCmd(g *globalFlags) *cobra.Command {
	var since string
	cmd := &cobra.Command{
		Use:   "changes --since <ref>",
		Short: "Show field-level changes across all items since a git ref",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if since == "" {
				return fmt.Errorf("changes: --since <ref> is required")
			}
			s, _, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Changes(since)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderChanges(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "", "git ref or sha to compare from (exclusive)")
	return cmd
}

func renderChanges(w io.Writer, res *datamodel.ChangesResult) {
	if len(res.Items) == 0 {
		fmt.Fprintln(w, "no changes")
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
		for _, e := range it.Events {
			fmt.Fprintf(w, "  %s  %s: %s -> %s\n", e.Ts, e.Field, e.Old, e.New)
		}
		if it.Body != nil {
			fmt.Fprintf(w, "  body: +%d/-%d lines\n", it.Body.Added, it.Body.Removed)
		}
	}
}
