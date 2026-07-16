package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/termx"
)

func newCreateCmd(g *globalFlags) *cobra.Command {
	var (
		opts          core.CreateOpts
		estimate      float64
		printTemplate bool
		aliasType     string
	)
	cmd := &cobra.Command{
		Use:   "create <type> [<title>]",
		Short: "Create an item of a configured type",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Type = args[0]
			opts.NoEdit = opts.NoEdit || g.nonInteractive
			if err := g.rejectStdinSource(opts.FromFile); err != nil {
				return err
			}
			if len(args) == 2 {
				if cmd.Flags().Changed("title") {
					return errx.User("provide the title positionally or via --title, not both")
				}
				opts.Title = args[1]
			}
			if opts.Subtype == "" {
				opts.Subtype = aliasType
			}
			if cmd.Flags().Changed("estimate") {
				opts.Estimate = &estimate
			}
			if opts.Here && (!opts.NoEdit || opts.Title == "") {
				return errx.User("--here requires --no-edit and --title")
			}
			if opts.Blocking && !opts.Here {
				return errx.User("--blocking requires --here")
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			if _, ok := cfg.Workflows[opts.Type]; !ok {
				return errx.User("%q is not a configured type", opts.Type).WithHint("configured types: %s", strings.Join(workflowTypes(cfg), ", "))
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
			if err := pickBoardIfAmbiguous(cmd, g, cfg, &opts); err != nil {
				return err
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
	f.BoolVar(&opts.Force, "force", false, "accept field values outside the configured vocabulary")
	f.StringVar(&opts.Board, "board", "", "board key to create the item under")
	f.BoolVar(&printTemplate, "print-template", false, "print the resolved template and exit")
	f.BoolVar(&opts.Here, "here", false, "capture under the active ticket's epic and sprint (requires --no-edit and --title)")
	f.BoolVar(&opts.Blocking, "blocking", false, "with --here, mark the new item as blocking the active ticket")
	return cmd
}

func pickBoardIfAmbiguous(cmd *cobra.Command, g *globalFlags, cfg *datamodel.Config, opts *core.CreateOpts) error {
	if opts.Board != "" {
		return nil
	}
	boards := cfg.ActiveBoards()
	if len(boards) <= 1 {
		return nil
	}
	if _, ok := cfg.DefaultBoard(); ok {
		return nil
	}
	if g.json || !g.prompter().Interactive() {
		return nil
	}
	out := cmd.ErrOrStderr()
	fmt.Fprintln(out, "Select a board:")
	for i, b := range boards {
		fmt.Fprintf(out, "  %d) %s  %s\n", i+1, b.Key, b.Name)
	}
	answer := strings.TrimSpace(termx.ReadLineDefault("board", boards[0].Key))
	if n, err := strconv.Atoi(answer); err == nil {
		if n < 1 || n > len(boards) {
			return errx.User("board choice %d out of range", n)
		}
		opts.Board = boards[n-1].Key
		return nil
	}
	if b, ok := cfg.BoardByKey(answer); ok && !b.Archived {
		opts.Board = b.Key
		return nil
	}
	return errx.User("no such board %q", answer)
}
