package cli

import (
	"bytes"
	"errors"

	"github.com/spf13/cobra"

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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(g, injectPanic)
		},
	}
	f := cmd.Flags()
	f.BoolVar(&injectPanic, "test-inject-panic", false, "")
	_ = f.MarkHidden("test-inject-panic")
	return cmd
}

func runTUI(g *globalFlags, injectPanic bool) error {
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
	return tui.Run(s, cfg, tui.Options{NoColor: g.noColor, InjectPanic: injectPanic, RunCommand: commandRunner(g)})
}

func canOfferInit(g *globalFlags, err error) bool {
	if !errors.Is(err, storage.ErrStoreNotFound) || !termx.IsInteractive() {
		return false
	}
	return gitx.Repo{Dir: g.chdir}.InsideWorkTree() == nil
}

func commandRunner(g *globalFlags) func([]string) (string, error) {
	return func(argv []string) (string, error) {
		var buf bytes.Buffer
		root, _ := newRootCmd()
		root.SetOut(&buf)
		root.SetErr(&buf)
		full := []string{"--no-color"}
		if g.chdir != "" {
			full = append(full, "-C", g.chdir)
		}
		root.SetArgs(append(full, argv...))
		err := root.Execute()
		return buf.String(), err
	}
}
