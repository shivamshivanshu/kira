package core

import (
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const (
	mergeDriverName = "kira field-level auto-merge"
	mergeDriverCmd  = "kira merge-file %O %A %B"
)

var mergeAttrLine = storage.TicketsPrefix + "/*.md merge=kira"

func (s *Store) RegisterMergeDriver() error {
	repo := s.repo()
	if err := repo.SetConfig("merge.kira.name", mergeDriverName); err != nil {
		return errx.User("%v", err)
	}
	if err := repo.SetConfig("merge.kira.driver", mergeDriverCmd); err != nil {
		return errx.User("%v", err)
	}
	if err := repo.AppendInfoAttribute(mergeAttrLine); err != nil {
		return errx.User("%v", err)
	}
	return nil
}
