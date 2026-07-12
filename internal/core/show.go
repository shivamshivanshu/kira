package core

import "github.com/shivamshivanshu/kira/internal/datamodel"

func (s *Store) Show(cfg *datamodel.Config, ref string) (*datamodel.ShowResult, error) {
	it, _, _, err := s.resolveRef(cfg, ref)
	if err != nil {
		return nil, err
	}
	res := showResultOf(cfg, it)
	return &res, nil
}
