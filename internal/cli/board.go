package cli

import (
	"fmt"
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
			if cfg.UI.AutoTUI && !plain && !g.json && epic == "" && at == "" && owner == "" && label == "" && query == "" && filter == "" && termx.IsTerminal(os.Stdout) {
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
	cmd.AddCommand(newBoardCreateCmd(g), newBoardListCmd(g), newBoardMoveCmd(g), newBoardRenameCmd(g), newBoardArchiveCmd(g), newBoardUnarchiveCmd(g))
	return cmd
}

func newBoardUnarchiveCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "unarchive <KEY>",
		Short: "Restore an archived board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.BoardUnarchive(cfg, args[0])
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Unarchived board %s\n", res.Board.Key)
			return nil
		},
	}
}

func newBoardRenameCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <KEY> <name>",
		Short: "Change a board's display name",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.BoardRename(cfg, args[0], args[1])
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Renamed board %s to %s\n", res.Board.Key, res.Board.Name)
			return nil
		},
	}
}

func newBoardArchiveCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "archive <KEY>",
		Short: "Archive a board (hidden from pickers; its tickets still resolve and list)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.BoardArchive(cfg, args[0])
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Archived board %s\n", res.Board.Key)
			return nil
		},
	}
}

func newBoardMoveCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "move <item> <KEY>",
		Short: "Move an item to another board (renumbers; old number is kept as an alias)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.BoardMove(cfg, args[0], args[1])
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Moved %s -> %s\n", res.From, res.To)
			return nil
		},
	}
}

func newBoardCreateCmd(g *globalFlags) *cobra.Command {
	var name, description string
	cmd := &cobra.Command{
		Use:   "create <KEY>",
		Short: "Add a board to the config (committed like any config mutation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.BoardCreate(cfg, args[0], name, description)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created board %s\n", res.Board.Key)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&name, "name", "", "display name (defaults to the key)")
	f.StringVar(&description, "description", "", "optional description")
	return cmd
}

func newBoardListCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured boards",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.BoardList(cfg)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			out := cmd.OutOrStdout()
			for _, b := range res.Boards {
				line := fmt.Sprintf("%s  %s", b.Key, b.Name)
				if b.Default {
					line += " (default)"
				}
				if b.Archived {
					line += " (archived)"
				}
				fmt.Fprintln(out, line)
			}
			return nil
		},
	}
}
