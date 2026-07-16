package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newResolveCmd(g *globalFlags) *cobra.Command {
	var interactive bool
	cmd := &cobra.Command{
		Use:   "resolve [id...]",
		Short: "Auto-resolve conflicted kira items in an in-progress merge",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Resolve(cfg, args, interactive)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			out := cmd.OutOrStdout()
			if len(res.Resolved) == 0 && len(res.Skipped) == 0 {
				_, _ = fmt.Fprintln(out, "No conflicted kira items to resolve")
				return nil
			}
			for _, r := range res.Resolved {
				if len(r.Arbitrated) > 0 {
					_, _ = fmt.Fprintf(out, "Resolved %s (auto-merged: %s)\n", r.Number, strings.Join(r.Arbitrated, ", "))
				} else {
					_, _ = fmt.Fprintf(out, "Resolved %s\n", r.Number)
				}
			}
			for _, sk := range res.Skipped {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Skipped %s (needs manual resolution)\n", sk)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&interactive, "interactive", false, "pick each conflicting field by hand")
	return cmd
}
