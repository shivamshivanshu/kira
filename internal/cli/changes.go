package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func newChangesCmd(g *globalFlags) *cobra.Command {
	var since string
	cmd := &cobra.Command{
		Use:   "changes --since <ref>",
		Short: "Show field-level changes across all items since a git ref",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if since == "" {
				return errx.User("changes: --since <ref> is required").WithHint("example: kira changes --since HEAD~10")
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
		renderDiffHeader(w, it.Status, it.Number, it.Title)
		for _, e := range it.Events {
			fmt.Fprintf(w, "  %s  %s: %s -> %s\n", e.Ts, e.Field, e.Old, e.New)
		}
		if it.Body != nil {
			fmt.Fprintf(w, "  body: +%d/-%d lines\n", it.Body.Added, it.Body.Removed)
		}
	}
}
