package cli

import (
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/termx"
)

func (g *globalFlags) prompter() core.Prompter {
	if g.nonInteractive {
		return core.SilentPrompter()
	}
	return terminalPrompter{}
}

func (g *globalFlags) rejectStdinSource(fromFile string) error {
	if g.nonInteractive && fromFile == "-" {
		return errx.User("--from-file - reads stdin, which is unavailable in non-interactive mode").WithHint("write the item to a file and pass --from-file <path>")
	}
	return nil
}

type terminalPrompter struct{}

func (terminalPrompter) Interactive() bool { return termx.IsInteractive() }

func (terminalPrompter) Confirm(prompt string) bool { return termx.Confirm(prompt) }

func (terminalPrompter) ReadLine(prompt, def string) string {
	return termx.ReadLineDefault(prompt, def)
}
