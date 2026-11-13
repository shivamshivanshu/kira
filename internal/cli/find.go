package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

func newFindCmd(g *globalFlags) *cobra.Command {
	var fa core.FindArgs
	cmd := &cobra.Command{
		Use:   "find <pattern>",
		Short: "Full-text regex search over ticket files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fa.Pattern = args[0]
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			rows, err := s.Find(cfg, fa)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), core.NewFindResult(rows))
			}
			renderFind(cmd.OutOrStdout(), rows)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&fa.IgnoreCase, "ignore-case", "i", false, "match case-insensitively")
	cmd.Flags().BoolVarP(&fa.Word, "word-regexp", "w", false, "match only whole words")
	return cmd
}

func renderFind(w io.Writer, rows []core.FindRow) {
	for _, r := range rows {
		_, _ = fmt.Fprintf(w, "%s:%d:%s\n", r.Number, r.Line, r.Text)
	}
}
