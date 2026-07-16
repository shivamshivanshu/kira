package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newIndexCmd(g *globalFlags) *cobra.Command {
	var full, closes bool
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Refresh or rebuild the derived cache index",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Index(cfg, full, closes)
			if err != nil {
				return err
			}
			emitStderrNotes(cmd.ErrOrStderr(), res.StderrNotes)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			if !g.quiet {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "index %s (%s): %d items\n", res.Action, res.Reason, res.Items)
				for _, num := range res.Closed {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "closed %s (Kira-Closes)\n", num)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&full, "full", false, "force a full rebuild from scratch")
	cmd.Flags().BoolVar(&closes, "closes", false, "apply Kira-Closes transitions for commits landed on git.landed_ref (kira sync and the post-merge hook are the usual callers)")
	return cmd
}
