package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/timex"
)

func newNowCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "now",
		Short: "Show the currently active ticket",
		Args:  cobra.NoArgs,
		RunE: storeActionRunE(g,
			func(s *core.Store, cfg *datamodel.Config, _ []string) (*datamodel.NowResult, error) {
				return s.Now(cfg)
			},
			renderNow),
	}
}

func renderNow(w io.Writer, r *datamodel.NowResult) {
	_, _ = fmt.Fprintf(w, "%s  %s  [%s]\n", r.Number, r.Title, r.State)
	line := func(label, value string) {
		if value != "" {
			_, _ = fmt.Fprintf(w, "%-12s %s\n", label+":", value)
		}
	}
	line("in state", timex.HumanSince(r.StateSince, timex.Now()))
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
		_, _ = fmt.Fprintf(w, "  %s  %s\n", gitx.ShortSHA(c.SHA), c.Subject)
	}
}
