package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/item"
)

func newCreateCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a ticket or epic",
	}
	cmd.AddCommand(newCreateSubCmd(g, item.TypeTicket), newCreateSubCmd(g, item.TypeEpic))
	return cmd
}

func newCreateSubCmd(g *globalFlags, typ string) *cobra.Command {
	var (
		opts          core.CreateOpts
		estimate      float64
		printTemplate bool
		reservedType  string
	)
	cmd := &cobra.Command{
		Use:   typ,
		Short: "Create a " + typ,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.Type = typ
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
	f.StringVar(&reservedType, "type", "", "reserved item subtype (accepted, unused in v1)")
	f.StringVar(&opts.Parent, "parent", "", "epic to set as this item's parent")
	f.StringVar(&opts.Owner, "owner", "", "owner")
	f.StringVar(&opts.Reporter, "reporter", "", "reporter")
	f.StringArrayVar(&opts.Labels, "label", nil, "label (repeatable)")
	f.Float64Var(&estimate, "estimate", 0, "estimate, in the configured unit")
	f.BoolVar(&opts.NoEdit, "no-edit", false, "create from flags only, no $EDITOR")
	f.StringVar(&opts.FromFile, "from-file", "", "read a complete item from a file (or - for stdin)")
	f.BoolVar(&opts.Force, "force", false, "bypass strict-vocabulary rejection")
	f.BoolVar(&printTemplate, "print-template", false, "print the resolved template and exit")
	return cmd
}
