package core

import (
	"github.com/shivamshivanshu/kira/internal/config"
)

// Show resolves ref to an item and returns its full detail read directly from
// the ticket file (docs/design/04-cli.md show). The index-derived fields
// (blocks, linked_commits, history_tail) stay empty pre-M2.
func (s *Store) Show(cfg *config.Config, ref string) (*ShowResult, error) {
	it, _, _, err := s.resolveRef(cfg, ref)
	if err != nil {
		return nil, err
	}
	res := showResultOf(cfg, it)
	return &res, nil
}
