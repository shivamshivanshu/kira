package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

func newCommentCmd(g *globalFlags) *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   "comment <id>",
		Short: "Append a comment to a ticket or epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := core.CommentOpts{Message: message, HasMessage: cmd.Flags().Changed("message")}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Comment(cfg, args[0], opts)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Commented on %s\n", res.Number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "comment text; opens $EDITOR when omitted")
	return cmd
}
