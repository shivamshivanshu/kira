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

type Option func(*Store)

func WithPrompter(p Prompter) Option {
	return func(s *Store) {
		if p != nil {
			s.prompter = p
		}
	}
}

func (s *Store) applyOptions(opts []Option) {
	for _, o := range opts {
		o(s)
	}
}
