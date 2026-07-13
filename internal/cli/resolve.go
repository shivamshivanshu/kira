package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

func newResolveCmd(g *globalFlags) *cobra.Command {
	var interactive bool
	cmd := &cobra.Command{
		Use:   "resolve [id...]",
		Short: "Auto-resolve conflicted kira items in an in-progress merge",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := core.Discover(g.chdir)
			if err != nil {
				return err
			}
			res, err := s.Resolve(args, interactive)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			out := cmd.OutOrStdout()
			if len(res.Resolved) == 0 && len(res.Skipped) == 0 {
				fmt.Fprintln(out, "No conflicted kira items to resolve")
				return nil
			}
			for _, r := range res.Resolved {
				if len(r.Arbitrated) > 0 {
					fmt.Fprintf(out, "Resolved %s (auto-merged: %s)\n", r.Number, strings.Join(r.Arbitrated, ", "))
				} else {
					fmt.Fprintf(out, "Resolved %s\n", r.Number)
				}
			}
			for _, sk := range res.Skipped {
				fmt.Fprintf(cmd.ErrOrStderr(), "Skipped %s (needs manual resolution)\n", sk)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&interactive, "interactive", false, "pick each conflicting field by hand")
	return cmd
}
