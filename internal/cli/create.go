package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newCreateCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an item of a configured type",
	}
	for _, typ := range createTypes(g) {
		cmd.AddCommand(newCreateSubCmd(g, typ))
	}
	return cmd
}

func createTypes(g *globalFlags) []string {
	s, err := core.Discover(chdirArg(g))
	if err != nil {
		return []string{datamodel.TypeTicket, datamodel.TypeEpic}
	}
	cfg, err := config.Load(s.Root())
	if err != nil || len(cfg.Workflows) == 0 {
		return []string{datamodel.TypeTicket, datamodel.TypeEpic}
	}
	types := make([]string, 0, len(cfg.Workflows))
	for typ := range cfg.Workflows {
		types = append(types, typ)
	}
	slices.Sort(types)
	return types
}

func chdirArg(g *globalFlags) string {
	if g.chdir != "" {
		return g.chdir
	}
	args := os.Args[1:]
	for i, a := range args {
		switch {
		case a == "-C" || a == "--C":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(a, "-C="):
			return strings.TrimPrefix(a, "-C=")
		case strings.HasPrefix(a, "--C="):
			return strings.TrimPrefix(a, "--C=")
		}
	}
	return ""
}

func newCreateSubCmd(g *globalFlags, typ string) *cobra.Command {
	var (
		opts          core.CreateOpts
		estimate      float64
		printTemplate bool
		aliasType     string
	)
	cmd := &cobra.Command{
		Use:   typ,
		Short: "Create a " + typ,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.Type = typ
			if opts.Subtype == "" {
				opts.Subtype = aliasType
			}
			if cmd.Flags().Changed("estimate") {
				opts.Estimate = &estimate
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			if printTemplate {
				tmpl, err := s.ResolveTemplate(opts)
				if err != nil {
					return err
				}
				if g.json {
					return emitJSON(cmd.OutOrStdout(), map[string]string{"template": tmpl})
				}
				fmt.Fprint(cmd.OutOrStdout(), tmpl)
				return nil
			}
			res, err := s.Create(cfg, opts)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created %s %s\n", res.Number, res.Title)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.Title, "title", "", "item title")
	f.StringVar(&opts.Subtype, "subtype", "", "item subtype (e.g. bug/story), validated against config subtypes")
	f.StringVar(&aliasType, "type", "", "alias for --subtype (deprecated)")
	_ = f.MarkHidden("type")
	f.StringVar(&opts.Parent, "parent", "", "epic to set as this item's parent")
	f.StringVar(&opts.Owner, "owner", "", "owner")
	f.StringVar(&opts.Reporter, "reporter", "", "reporter")
	f.StringArrayVar(&opts.Labels, "label", nil, "label (repeatable)")
	f.StringVar(&opts.Priority, "priority", "", "priority, validated against config priorities")
	f.StringVar(&opts.Rank, "rank", "", "lexicographic grooming key")
	f.StringVar(&opts.Sprint, "sprint", "", "sprint key from config sprints")
	f.StringVar(&opts.Due, "due", "", "target completion date (RFC3339 date)")
	f.Float64Var(&estimate, "estimate", 0, "estimate, in the configured unit")
	f.BoolVar(&opts.NoEdit, "no-edit", false, "create from flags only, no $EDITOR")
	f.StringVar(&opts.FromFile, "from-file", "", "read a complete item from a file (or - for stdin)")
	f.BoolVar(&opts.Force, "force", false, "bypass strict-vocabulary rejection")
	f.BoolVar(&printTemplate, "print-template", false, "print the resolved template and exit")
	return cmd
}
