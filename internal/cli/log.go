package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func newLogCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "log <id>",
		Short: "Show an item's field history interleaved with linked commits",
		Args:  cobra.ExactArgs(1),
		RunE: storeActionRunE(g,
			func(s *core.Store, cfg *datamodel.Config, args []string) (*datamodel.LogResult, error) {
				return s.Log(cfg, args[0])
			},
			renderLog),
	}
}

func renderLog(w io.Writer, res *datamodel.LogResult) {
	if len(res.Entries) == 0 {
		_, _ = fmt.Fprintln(w, "no history")
		return
	}
	for _, e := range res.Entries {
		if e.Kind == "commit" {
			_, _ = fmt.Fprintf(w, "%s  commit %s %s (%s)\n", e.Ts, gitx.ShortSHA(e.SHA), e.Subject, e.Author)
			continue
		}
		_, _ = fmt.Fprintf(w, "%s  %s: %s -> %s\n", e.Ts, e.Field, e.Old, e.New)
	}
}
