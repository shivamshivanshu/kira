package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newMoveCmd(g *globalFlags) *cobra.Command {
	var opts core.MoveOpts
	cmd := &cobra.Command{
		Use:   "move <id>... <state>",
		Short: "Transition one or more tickets or epics to a new state",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, state := args[:len(args)-1], args[len(args)-1]
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			b, err := s.BeginBatch(cfg)
			if err != nil {
				return err
			}
			defer b.Close()
			apply := func(id string) (*datamodel.MoveResult, error) { return b.Move(id, state, opts) }
			warn := func(w io.Writer, res *datamodel.MoveResult) { emitWarningLines(w, res.Warnings) }
			out := cmd.OutOrStdout()
			return runSingleOrBulk(out, cmd.ErrOrStderr(), g.json, ids, apply, moveLine, warn)
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.Resolution, "resolution", "", "resolution to record, validated against config resolutions")
	f.BoolVar(&opts.Force, "force", false, "bypass the transition adjacency check and require: guards")
	f.BoolVar(&opts.Activate, "activate", false, "set this item as the active ticket")
	return cmd
}

func moveLine(res *datamodel.MoveResult) string {
	line := fmt.Sprintf("Moved %s: %s -> %s", res.Number, res.From, res.To)
	if res.Activated {
		line += fmt.Sprintf("\nActivated %s", res.Number)
	}
	return line
}
