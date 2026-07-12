package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

func newMoveCmd(g *globalFlags) *cobra.Command {
	var opts core.MoveOpts
	cmd := &cobra.Command{
		Use:   "move <id> <state>",
		Short: "Transition a ticket or epic to a new state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Move(cfg, args[0], args[1], opts)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Moved %s: %s -> %s\n", res.Number, res.From, res.To)
			if res.Activated {
				fmt.Fprintf(cmd.OutOrStdout(), "Activated %s\n", res.Number)
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&opts.Force, "force", false, "bypass the transition adjacency check")
	f.BoolVar(&opts.Activate, "activate", false, "set this item as the active ticket")
	return cmd
}
