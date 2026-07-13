package cli

import "github.com/shivamshivanshu/kira/internal/termx"

type terminalPrompter struct{}

func (terminalPrompter) Interactive() bool { return termx.IsInteractive() }

func (terminalPrompter) Confirm(prompt string) bool { return termx.Confirm(prompt) }

func (terminalPrompter) ReadLine(prompt, def string) string {
	return termx.ReadLineDefault(prompt, def)
}
