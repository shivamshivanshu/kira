package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/timex"
)

func newNowCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "now",
		Short: "Show the currently active ticket",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Now(cfg)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderNow(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func renderNow(w io.Writer, r *datamodel.NowResult) {
	fmt.Fprintf(w, "%s  %s  [%s]\n", r.Number, r.Title, r.State)
	line := func(label, value string) {
		if value != "" {
			fmt.Fprintf(w, "%-12s %s\n", label+":", value)
		}
	}
	line("in state", timex.HumanSince(r.StateSince, time.Now()))
	if r.Due != nil {
		due := *r.Due
		if r.Overdue {
			due += " overdue"
		}
		line("due", due)
	}
	line("branch", r.Branch)
	if len(r.Blockers) > 0 {
		refs := make([]string, len(r.Blockers))
		for i, b := range r.Blockers {
			refs[i] = fmt.Sprintf("%s [%s]", b.Number, b.State)
		}
		line("blocked by", strings.Join(refs, ", "))
	}
	line("commits", fmt.Sprintf("%d since last state change", len(r.Commits)))
	for _, c := range r.Commits {
		fmt.Fprintf(w, "  %s  %s\n", shortSHA(c.SHA), c.Subject)
	}
}
