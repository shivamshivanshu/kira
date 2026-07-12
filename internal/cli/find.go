package cli

import (
	"fmt"
	"io"
	"slices"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

// The global `-C <path>` chdir is deliberately absent: inside `find`, `-C`
// means ripgrep's context flag and must pass through.
var knownGlobalFlags = []string{"--json", "--no-color", "--quiet"}

func newFindCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "find <pattern> [rg-flags...]",
		Short:              "Full-text search over ticket files (ripgrep wrapper)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if slices.Contains(args, "-h") || slices.Contains(args, "--help") {
				return cmd.Help()
			}
			fa := core.ParseFindArgs(args, knownGlobalFlags)
			if fa.Pattern == "" {
				return fmt.Errorf("find: a search pattern is required")
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			rows, err := s.Find(cfg, fa)
			if err != nil {
				return err
			}
			if g.json || slices.Contains(args, "--json") {
				return emitJSON(cmd.OutOrStdout(), core.NewFindResult(rows))
			}
			renderFind(cmd.OutOrStdout(), rows)
			return nil
		},
	}
	return cmd
}

func renderFind(w io.Writer, rows []core.FindRow) {
	for _, r := range rows {
		switch r.Kind {
		case core.RowSeparator:
			fmt.Fprintln(w, r.Text)
		case core.RowContext:
			fmt.Fprintf(w, "%s-%d-%s\n", r.Number, r.Line, r.Text)
		default:
			fmt.Fprintf(w, "%s:%d:%s\n", r.Number, r.Line, r.Text)
		}
	}
}
