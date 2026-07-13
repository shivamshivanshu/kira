package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/termx"
	"github.com/shivamshivanshu/kira/internal/tui"
)

const plainBoardWidth = 100

func newBoardCmd(g *globalFlags) *cobra.Command {
	var plain bool
	var at string
	cmd := &cobra.Command{
		Use:   "board [<epic-id>]",
		Short: "Kanban board of tickets grouped by workflow state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			epic := ""
			if len(args) == 1 {
				epic = args[0]
			}
			if !plain && !g.json && epic == "" && at == "" && termx.IsTerminal(os.Stdout) {
				return tui.Run(s, cfg, tui.Options{NoColor: g.noColor, RunCommand: commandRunner(g), InitialView: tui.ViewBoard})
			}
			res, err := s.Board(cfg, core.BoardOpts{Epic: epic, At: at})
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			return tui.RenderBoardPlain(cmd.OutOrStdout(), cfg, res, plainBoardWidth, g.noColor)
		},
	}
	f := cmd.Flags()
	f.BoolVar(&plain, "plain", false, "force the static table instead of launching the interactive board")
	f.StringVar(&at, "at", "", "render the board as of a git ref (static, read-only)")
	return cmd
}
