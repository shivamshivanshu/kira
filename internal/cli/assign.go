package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

func newAssignCmd(g *globalFlags) *cobra.Command {
	var opts core.AssignOpts
	cmd := &cobra.Command{
		Use:   "assign <id> <user>",
		Short: "Assign a ticket's owner (or reporter)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Assign(cfg, args[0], args[1], opts)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			if len(res.Changed) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: no changes\n", res.Number)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Assigned %s: %s\n", res.Number, args[1])
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&opts.Reporter, "reporter", false, "assign the reporter field instead of owner")
	f.BoolVar(&opts.Force, "force", false, "bypass strict-vocabulary rejection")
	return cmd
}
