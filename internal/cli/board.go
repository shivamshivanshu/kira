package cli

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/termx"
	"github.com/shivamshivanshu/kira/internal/tui"
)

const fallbackBoardWidth = 80

func boardWidth() int {
	if w, ok := termx.Width(os.Stdout); ok {
		return w
	}
	if w, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && w > 0 {
		return w
	}
	return fallbackBoardWidth
}

func newBoardCmd(g *globalFlags) *cobra.Command {
	var plain bool
	var at string
	var owner string
	var label string
	var query string
	var filter string
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
			if !plain && !g.json && epic == "" && at == "" && owner == "" && label == "" && query == "" && filter == "" && termx.IsTerminal(os.Stdout) {
				return tui.Run(s, cfg, tui.Options{NoColor: g.noColor, RunCommand: commandRunner(g), InitialView: tui.ViewBoard})
			}
			res, err := s.Board(cfg, core.BoardOpts{Epic: epic, Owner: owner, Label: label, Query: query, Filter: filter, At: at})
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			return tui.RenderBoardPlain(cmd.OutOrStdout(), cfg, res, boardWidth(), g.noColor)
		},
	}
	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "filter to one owner ('@me' resolves to the git user)")
	f.StringVar(&label, "label", "", "filter by label")
	f.StringVar(&filter, "filter", "", "apply a named saved query from config filters")
	f.StringVar(&query, "query", "", "filter by a query expression (ANDed with the flags)")
	f.BoolVar(&plain, "plain", false, "force the static table instead of launching the interactive board")
	f.StringVar(&at, "at", "", "render the board as of a git ref (static, read-only)")
	return cmd
}
