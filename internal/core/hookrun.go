package core

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/storage"
	"github.com/shivamshivanshu/kira/internal/workon"
)

func (s *Store) PrepareCommitMsg(cfg *datamodel.Config, msgFile string) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	repo := s.repo()
	number, ok := s.trailerNumber(cfg, repo)
	if !ok {
		return nil
	}
	if err := repo.AddTrailer(msgFile, cfg.Commit.Trailer, number); err != nil {
		return errx.User("%v", err)
	}
	return nil
}

func (s *Store) trailerNumber(cfg *datamodel.Config, repo gitx.Repo) (string, bool) {
	branch, _ := repo.CurrentBranch()
	items, _, resolver, err := s.load(cfg)
	if err != nil {
		return "", false
	}
	if ptr, ok := s.readActive(); ok && ptr.Branch != "" && ptr.Branch == branch {
		if number, ok := resolveNumber(items, resolver, ptr.Ticket); ok {
			return number, true
		}
	}
	if display, ok := workon.InferNumber(branch, cfg.Project.Key); ok {
		if number, ok := resolveNumber(items, resolver, display); ok {
			return number, true
		}
	}
	return "", false
}

func resolveNumber(items []*datamodel.Item, resolver *id.Resolver, ref string) (string, bool) {
	ulid, err := resolver.Resolve(ref)
	if err != nil {
		return "", false
	}
	if it := findByULID(items, ulid); it != nil {
		return it.Number, true
	}
	return "", false
}

func (s *Store) ValidateStaged(cfg *datamodel.Config) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	paths, err := s.repo().StagedPaths()
	if err != nil {
		return errx.User("%v", err)
	}
	var problems []error
	for _, p := range paths {
		if !storage.IsTicketPath(p) {
			continue
		}
		content, err := s.repo().ShowStaged(p)
		if err != nil {
			continue
		}
		it, err := codec.Parse(content)
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %v", p, err))
			continue
		}
		errs, _ := validateItem(cfg, it, false)
		for _, e := range errs {
			problems = append(problems, fmt.Errorf("%s: %v", p, e))
		}
	}
	if len(problems) > 0 {
		return errx.Invalid(problems)
	}
	return nil
}
