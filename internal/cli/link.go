package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func newLinkCmd(g *globalFlags) *cobra.Command {
	var opts core.LinkOpts
	type edge struct {
		flag     string
		usage    string
		target   core.LinkTarget
		linkType string
		value    *string
	}
	edges := []edge{
		{"epic", "epic parent to set (or clear with --remove)", core.LinkEpic, "", new(string)},
		{"blocked-by", "blocking item to add (or remove with --remove)", core.LinkBlockedBy, "", new(string)},
	}
	for _, typ := range datamodel.LinkTypes {
		edges = append(edges, edge{
			flag:     core.FlagForLinkType(string(typ)),
			usage:    fmt.Sprintf("item to add to links.%s (or remove with --remove)", typ),
			target:   core.LinkTyped,
			linkType: string(typ),
			value:    new(string),
		})
	}
	flagNames := make([]string, len(edges))
	for i, e := range edges {
		flagNames[i] = "--" + e.flag
	}
	cmd := &cobra.Command{
		Use:   "link <id>",
		Short: "Set or remove an epic parent, blocked-by dependency, or typed link",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			given := 0
			for _, e := range edges {
				if cmd.Flags().Changed(e.flag) {
					given++
					opts.Target, opts.Type, opts.Ref = e.target, e.linkType, *e.value
				}
			}
			if given != 1 {
				return errx.User("give exactly one of %s", strings.Join(flagNames, ", "))
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Link(cfg, args[0], opts)
			if err != nil {
				return err
			}
			emitMutationWarnings(cmd.ErrOrStderr(), res.Warnings)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			if len(res.Changed) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: no changes\n", res.Number)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Linked %s\n", res.Number)
			}
			return nil
		},
	}
	f := cmd.Flags()
	for _, e := range edges {
		f.StringVar(e.value, e.flag, "", e.usage)
	}
	f.BoolVar(&opts.Remove, "remove", false, "remove the given edge instead of adding it")
	f.BoolVar(&opts.Force, "force", false, "accept field values outside the configured vocabulary")
	return cmd
}
