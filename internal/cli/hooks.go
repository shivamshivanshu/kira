package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newHooksCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage kira git hooks",
	}
	cmd.AddCommand(
		newHooksInstallCmd(g),
		newHooksPostMergeCmd(g),
		newHooksPrepareCommitMsgCmd(g),
		newHooksPreCommitCmd(g),
	)
	return cmd
}

func newHooksInstallCmd(g *globalFlags) *cobra.Command {
	var opts core.HooksInstallOpts
	var validate bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install kira git hooks and register the merge driver",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			if validate {
				res, err := s.ValidateHooks(cfg)
				if err != nil {
					return err
				}
				if g.json {
					return emitJSON(cmd.OutOrStdout(), res)
				}
				printHooksValidate(cmd, res)
				return nil
			}
			res, err := s.InstallHooks(cfg, opts)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			printHooksInstall(cmd, res)
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&opts.WithPreCommit, "with-pre-commit", false, "additionally install the pre-commit validation hook")
	f.BoolVar(&validate, "validate", false, "verify installed hooks and merge-driver registration without modifying anything")
	return cmd
}

func newHooksPostMergeCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:    "post-merge",
		Short:  "post-merge hook entry point: reconcile ID collisions and fire landed closes",
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Reconcile(cfg)
			if err != nil {
				return err
			}
			for _, r := range res.Renumbered {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s renumbered %s -> %s\n", msgPrefix, r.From, r.To)
			}
			idx, err := s.Index(cfg, false, true)
			if err != nil {
				return err
			}
			emitStderrNotes(cmd.ErrOrStderr(), idx.StderrNotes)
			for _, num := range idx.Closed {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s closed %s\n", msgPrefix, num)
			}
			return nil
		},
	}
}

func newHooksPrepareCommitMsgCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:    "prepare-commit-msg <file>",
		Short:  "prepare-commit-msg hook entry point: append the ticket trailer",
		Hidden: true,
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			return s.PrepareCommitMsg(cfg, args[0])
		},
	}
}

func newHooksPreCommitCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:    "pre-commit",
		Short:  "pre-commit hook entry point: validate staged kira items",
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			return s.ValidateStaged(cfg)
		},
	}
}

func printHooksInstall(cmd *cobra.Command, res *datamodel.HooksInstallResult) {
	out := cmd.OutOrStdout()
	for _, h := range res.Hooks {
		switch {
		case h.Chained:
			fmt.Fprintf(out, "hook %s: chained onto existing hook\n", h.Name)
		case h.Installed:
			fmt.Fprintf(out, "hook %s: installed\n", h.Name)
		default:
			fmt.Fprintf(out, "hook %s: refused — .git/hooks/%s is an unrecognized existing hook; add kira's block by hand\n", h.Name, h.Name)
		}
	}
	if res.MergeDriver {
		fmt.Fprintln(out, "merge driver: registered")
	}
}

func printHooksValidate(cmd *cobra.Command, res *datamodel.HooksValidateResult) {
	out := cmd.OutOrStdout()
	for _, h := range res.Hooks {
		state := "missing"
		if h.Installed {
			state = "installed"
		}
		fmt.Fprintf(out, "hook %s: %s\n", h.Name, state)
	}
	if res.MergeDriver {
		fmt.Fprintln(out, "merge driver: registered")
	} else {
		fmt.Fprintln(out, "merge driver: not registered (run `kira hooks install`)")
	}
}
