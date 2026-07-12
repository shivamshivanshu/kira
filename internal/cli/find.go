package cli

import (
	"fmt"
	"io"
	"slices"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

// knownGlobalFlags are the kira persistent flags to strip from find's rg
// passthrough — they are kira's, not rg's. Kept in one place so it tracks the
// flags registered in newRootCmd. The global `-C <path>` chdir is deliberately
// omitted: inside `find`, `-C` means ripgrep's context (docs/design/04-cli.md find).
var knownGlobalFlags = []string{"--json", "--no-color", "--quiet"}

// newFindCmd builds `kira find`. Flag parsing is disabled so every rg flag
// (`-i`, `-w`, `-C N`, and any other) forwards verbatim to ripgrep — including
// `-C`, which would otherwise collide with the global `-C <path>` shorthand
// (docs/design/04-cli.md find).
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

// renderFind prints the human view of find: one line per row, with the file
// path already rewritten to the display number. Matches use rg's `number:line:`
// form, context lines its `number-line-` form, separators pass through.
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
