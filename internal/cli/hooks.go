package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/hooks"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func newHooksCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage kira git hooks",
	}
	cmd.AddCommand(
		newHooksInstallCmd(g),
		newHooksUninstallCmd(g),
		newHooksStatusCmd(g),
		newHooksRunCmd(g),
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
				printHooksValidate(cmd.OutOrStdout(), res)
				return nil
			}
			res, err := s.InstallHooks(cfg, opts)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			printHooksInstall(cmd.OutOrStdout(), res)
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&opts.WithPreCommit, "with-pre-commit", false, "additionally install the pre-commit validation hook")
	f.BoolVar(&opts.IntoHooksPath, "into-hooks-path", false, "install into the core.hooksPath directory even when it lives inside the work tree")
	f.BoolVar(&validate, "validate", false, "verify installed hooks and merge-driver registration without modifying anything")
	return cmd
}

func newHooksUninstallCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove kira's hook shims, restore chained hooks, and unregister the merge driver",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := core.Discover(g.chdir, g.prompter())
			if err != nil {
				return err
			}
			res, err := s.UninstallHooks()
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			printHooksUninstall(cmd.OutOrStdout(), cmd.ErrOrStderr(), res)
			return nil
		},
	}
}

func newHooksStatusCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Report the state of each kira git hook: installed, chained, missing, drifted, or foreign",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := core.Discover(g.chdir, g.prompter())
			if err != nil {
				return err
			}
			res, err := s.HooksStatus()
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			printHooksStatus(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func newHooksRunCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "run <hook> [args...]",
		Short: "Run kira's logic for a git hook (the entry point installed shims call)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch name := args[0]; name {
			case "post-merge":
				return runHookPostMerge(cmd, g)
			case "prepare-commit-msg":
				if len(args) < 2 {
					return errx.User("prepare-commit-msg needs the commit message file argument")
				}
				return runHookPrepareCommitMsg(g, args[1])
			case hooks.PreCommit:
				return runHookPreCommit(g)
			default:
				return errx.User("unknown hook %q", name).
					WithHint("kira handles: %v", hooks.Known())
			}
		},
	}
}

func hookStore(g *globalFlags) (*core.Store, error) {
	s, err := core.Discover(g.chdir, g.prompter())
	if err != nil && errors.Is(err, storage.ErrStoreNotFound) {
		return nil, nil
	}
	return s, err
}

func runHookPostMerge(cmd *cobra.Command, g *globalFlags) error {
	s, err := hookStore(g)
	if s == nil {
		return err
	}
	cfg, err := s.Config()
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
}

func runHookPrepareCommitMsg(g *globalFlags, msgFile string) error {
	s, err := hookStore(g)
	if s == nil {
		return err
	}
	return s.PrepareCommitMsgHook(msgFile)
}

func runHookPreCommit(g *globalFlags) error {
	s, err := hookStore(g)
	if s == nil {
		return err
	}
	cfg, err := s.Config()
	if err != nil {
		return err
	}
	return s.ValidateStaged(cfg)
}

func newHooksPostMergeCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:    "post-merge",
		Short:  "post-merge hook entry point: reconcile ID collisions and fire landed closes",
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHookPostMerge(cmd, g)
		},
	}
}

func newHooksPrepareCommitMsgCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:    "prepare-commit-msg <file>",
		Short:  "prepare-commit-msg hook entry point: append the ticket trailer",
		Hidden: true,
		Args:   cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runHookPrepareCommitMsg(g, args[0])
		},
	}
}

func newHooksPreCommitCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:    "pre-commit",
		Short:  "pre-commit hook entry point: validate staged kira items",
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHookPreCommit(g)
		},
	}
}

func printHooksInstall(out io.Writer, res *datamodel.HooksInstallResult) {
	for _, h := range res.Hooks {
		switch {
		case h.Chained:
			fmt.Fprintf(out, "hook %s: chained onto existing hook\n", h.Name)
		case h.Installed:
			fmt.Fprintf(out, "hook %s: installed\n", h.Name)
		case h.Note != "":
			fmt.Fprintf(out, "hook %s: refused — %s\n", h.Name, h.Note)
		default:
			fmt.Fprintf(out, "hook %s: refused — .git/hooks/%s is an unrecognized existing hook; add kira's block by hand\n", h.Name, h.Name)
		}
	}
	if res.MergeDriver {
		fmt.Fprintln(out, "merge driver: registered")
	}
}

func printHooksValidate(out io.Writer, res *datamodel.HooksValidateResult) {
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

func printHooksStatus(out io.Writer, res *datamodel.HooksStatusResult) {
	for _, h := range res.Hooks {
		suffix := ""
		if h.State == string(hooks.StateDrifted) {
			suffix = " (differs from kira's current shim)"
		}
		fmt.Fprintf(out, "hook %s: %s%s\n", h.Name, h.State, suffix)
	}
	if res.MergeDriver {
		fmt.Fprintln(out, "merge driver: registered")
	} else {
		fmt.Fprintln(out, "merge driver: not registered (run `kira hooks install`)")
	}
	if res.HooksPath != "" {
		fmt.Fprintf(out, "core.hooksPath: %s (hooks live there)\n", res.HooksPath)
	}
}

func printHooksUninstall(out, errOut io.Writer, res *datamodel.HooksUninstallResult) {
	for _, h := range res.Hooks {
		fmt.Fprintf(out, "hook %s: %s\n", h.Name, h.State)
		if h.Note != "" {
			fmt.Fprintf(errOut, "%s warning: %s\n", msgPrefix, h.Note)
		}
	}
	if res.MergeDriver {
		fmt.Fprintln(out, "merge driver: unregistered")
	}
}
