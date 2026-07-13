package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newLogCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "log <id>",
		Short: "Show an item's field history interleaved with linked commits",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Log(cfg, args[0])
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderLog(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func renderLog(w io.Writer, res *datamodel.LogResult) {
	if len(res.Entries) == 0 {
		fmt.Fprintln(w, "no history")
		return
	}
	for _, e := range res.Entries {
		if e.Kind == "commit" {
			fmt.Fprintf(w, "%s  commit %s %s (%s)\n", e.Ts, shortSHA(e.SHA), e.Subject, e.Author)
			continue
		}
		fmt.Fprintf(w, "%s  %s: %s -> %s\n", e.Ts, e.Field, e.Old, e.New)
	}
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
