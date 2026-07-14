package core

type Prompter interface {
	Interactive() bool
	Confirm(prompt string) bool
	ReadLine(prompt, def string) string
}

type silentPrompter struct{}

func (silentPrompter) Interactive() bool             { return false }
func (silentPrompter) Confirm(string) bool           { return false }
func (silentPrompter) ReadLine(_, def string) string { return def }

func SilentPrompter() Prompter { return silentPrompter{} }

func firstPrompter(prompter []Prompter) Prompter {
	if len(prompter) > 0 && prompter[0] != nil {
		return prompter[0]
	}
	return silentPrompter{}
}
