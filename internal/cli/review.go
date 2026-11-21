package cli

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func newReviewCmd(g *globalFlags) *cobra.Command {
	var base string
	cmd := &cobra.Command{
		Use:   "review --base <ref>",
		Short: "Summarize the current branch for review",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if base == "" {
				return errx.User("review: --base <ref> is required").WithHint("example: kira review --base main")
			}
			s, err := core.Discover(g.chdir)
			if err != nil {
				return err
			}
			result, err := s.Review(base)
			if err != nil {
				return err
			}
			emitStderrNotes(cmd.ErrOrStderr(), result.StderrNotes)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), result)
			}
			renderReview(cmd.OutOrStdout(), result)
			return nil
		},
	}
	cmd.Flags().StringVar(&base, "base", "", "git ref the branch will be reviewed against")
	_ = cmd.RegisterFlagCompletionFunc("base", completeReviewBase(g))
	return cmd
}

func completeReviewBase(g *globalFlags) completer {
	return func(_ *cobra.Command, _ []string, prefix string) ([]string, cobra.ShellCompDirective) {
		s, err := core.Discover(g.chdir)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		branches, err := (gitx.Repo{Dir: s.Root()}).Branches()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return prefixed(branches, prefix), cobra.ShellCompDirectiveNoFileComp
	}
}

var reviewMarkdown = strings.NewReplacer("\\", "\\\\", "`", "\\`", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_", "~", "\\~", "&", "&amp;", "<", "&lt;", ">", "&gt;")

func reviewText(text string) string {
	line := strings.Join(strings.Fields(text), " ")
	return reviewMarkdown.Replace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, line))
}

func renderReview(w io.Writer, res *datamodel.ReviewResult) {
	_, _ = fmt.Fprintf(w, "# Branch review\n\nBase %s · Merge base %s · Head %s\n", gitx.ShortSHA(res.Base), gitx.ShortSHA(res.MergeBase), gitx.ShortSHA(res.Head))
	if len(res.Linked) > 0 {
		_, _ = fmt.Fprint(w, "\n## Linked tickets\n\n")
		for _, link := range res.Linked {
			_, _ = fmt.Fprintf(w, "- **%s** %s\n", reviewText(link.Ticket.Number), reviewText(link.Ticket.Title))
			for _, commit := range link.Commits {
				_, _ = fmt.Fprintf(w, "  - %s %s\n", gitx.ShortSHA(commit.SHA), reviewText(commit.Subject))
			}
		}
	}
	if len(res.StateChanges) > 0 {
		_, _ = fmt.Fprint(w, "\n## Net state changes\n\n")
		for _, change := range res.StateChanges {
			_, _ = fmt.Fprintf(w, "- **%s** %s → %s\n", reviewText(change.Ticket.Number), reviewText(change.From), reviewText(change.To))
		}
	}
	if len(res.StateEvents) > 0 {
		_, _ = fmt.Fprint(w, "\n## State changes from patches\n\n")
		for _, history := range res.StateEvents {
			for _, event := range history.Events {
				_, _ = fmt.Fprintf(w, "- **%s** %s → %s (%s)\n", reviewText(history.Ticket.Number), reviewText(event.Old), reviewText(event.New), gitx.ShortSHA(event.CommitSHA))
			}
		}
	}
	if len(res.Closes) > 0 {
		_, _ = fmt.Fprintf(w, "\n## Kira-Closes preview\n\nEligible closes assume the trailer reaches %s and the ticket and config stay unchanged.\n\n", reviewText(res.LandedRef))
		for _, preview := range res.Closes {
			_, _ = fmt.Fprintf(w, "- **%s** %s (%s)\n", reviewText(preview.Ticket.Number), reviewCloseText(preview), gitx.ShortSHA(preview.CommitSHA))
		}
	}
	if len(res.Unresolved) > 0 {
		_, _ = fmt.Fprint(w, "\n## Unresolved references\n\n")
		for _, ref := range res.Unresolved {
			outcome := "unknown"
			if ref.Ambiguous {
				outcome = "ambiguous"
			}
			_, _ = fmt.Fprintf(w, "- %s in %s is %s (%s)\n", reviewText(ref.Ref), reviewText(ref.Source), outcome, gitx.ShortSHA(ref.CommitSHA))
		}
	}
}

func reviewCloseText(preview datamodel.ReviewClose) string {
	switch preview.Outcome {
	case datamodel.ReviewCloseEligible:
		return "would move to " + reviewText(preview.Target)
	case datamodel.ReviewCloseAlreadyLanded:
		return "trailer already reachable from the landed ref"
	case datamodel.ReviewCloseAlreadyClosed:
		return "ticket already closed"
	case datamodel.ReviewCloseInvalidTimestamp:
		return "invalid ticket or committer timestamp"
	case datamodel.ReviewCloseTicketNewer:
		return "ticket changed after the trailer commit"
	case datamodel.ReviewCloseNoTarget:
		return "workflow has no done state"
	default:
		return reviewText(preview.Outcome)
	}
}
