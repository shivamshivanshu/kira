package cli

import (
	"bytes"
	"errors"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/storage"
	"github.com/shivamshivanshu/kira/internal/termx"
	"github.com/shivamshivanshu/kira/internal/tui"
)

func newTUICmd(g *globalFlags) *cobra.Command {
	var injectPanic bool
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive terminal UI (also the default when kira is run with no command)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTUI(cmd, g, injectPanic, false)
		},
	}
	f := cmd.Flags()
	f.BoolVar(&injectPanic, "test-inject-panic", false, "")
	_ = f.MarkHidden("test-inject-panic")
	return cmd
}

func runTUI(cmd *cobra.Command, g *globalFlags, injectPanic, auto bool) error {
	if g.nonInteractive && !auto {
		return errx.User("cannot launch the tui in non-interactive mode")
	}
	s, cfg, err := openStore(g)
	if err != nil {
		if !canOfferInit(g, err) {
			return err
		}
		initialized, uerr := tui.RunUninit(g.chdir, tui.Options{NoColor: g.noColor})
		if uerr != nil || !initialized {
			return uerr
		}
		if s, cfg, err = openStore(g); err != nil {
			return err
		}
	}
	if auto && !shouldAutoTUI(cmd, g, cfg) {
		return cmd.Help()
	}
	return tui.Run(s.WithPrompter(core.SilentPrompter()), cfg, tui.Options{NoColor: g.noColor, InjectPanic: injectPanic, RunCommand: commandRunner(g)})
}

func shouldAutoTUI(cmd *cobra.Command, g *globalFlags, cfg *datamodel.Config) bool {
	return autoTUIAllowed(g, cfg) && termx.WriterIsTTY(cmd.OutOrStdout())
}

func autoTUIAllowed(g *globalFlags, cfg *datamodel.Config) bool {
	return cfg.UI.AutoTUI && !g.json && !g.nonInteractive
}

func canOfferInit(g *globalFlags, err error) bool {
	if g.nonInteractive || !errors.Is(err, storage.ErrStoreNotFound) || !termx.IsInteractive() {
		return false
	}
	return gitx.Repo{Dir: g.chdir}.InsideWorkTree() == nil
}

func commandRunner(g *globalFlags) func([]string) (string, error) {
	return func(argv []string) (string, error) {
		var buf bytes.Buffer
		root, bridgedG := newRootCmd()
		bridgedG.chdir = g.chdir
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs(append([]string{"--no-color", "--non-interactive"}, argv...))
		err := root.Execute()
		return buf.String(), err
	}
}
