package core

import (
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

func (s *Store) BoardMove(cfg *datamodel.Config, ref, targetKey string) (*datamodel.BoardMoveResult, error) {
	board, ok := cfg.BoardByKey(targetKey)
	if !ok || board.Archived {
		return nil, errx.User("no such board %q", targetKey).
			WithHint("boards: %s", strings.Join(activeBoardKeys(cfg.ActiveBoards()), ", "))
	}

	release, err := s.fs().Lock()
	if err != nil {
		return nil, err
	}
	defer release()

	ld, err := s.load(cfg)
	if err != nil {
		return nil, err
	}
	it, err := findItem(ld.items, ld.resolver, ref)
	if err != nil {
		return nil, err
	}
	if err := guardWritable(it); err != nil {
		return nil, err
	}

	from := it.Number
	if strings.EqualFold(id.KeyOf(from), board.Key) {
		return nil, errx.User("%s is already on board %s", from, board.Key)
	}

	u, err := id.ParseULID(it.ID)
	if err != nil {
		return nil, errx.User("item %s has an invalid ULID: %v", from, err)
	}
	to := allocateNumber(cfg, ld.snap, board.Key, u)

	it.Number = to
	it.Aliases = slices.DeleteFunc(it.Aliases, func(a string) bool { return strings.EqualFold(a, to) })
	if !slices.ContainsFunc(it.Aliases, func(a string) bool { return strings.EqualFold(a, from) }) {
		it.Aliases = append(it.Aliases, from)
	}
	path, err := s.fs().WriteItem(it)
	if err != nil {
		return nil, err
	}
	subject := cfg.Commit.SubjectPrefix + "board " + from + " -> " + to
	if _, err := s.finalize(cfg.Commit.Mode, commitSpec{trailerKey: cfg.Commit.Trailer, subject: subject, trailerNumber: to}, path); err != nil {
		return nil, err
	}
	return &datamodel.BoardMoveResult{ID: it.ID, From: from, To: to, Board: board.Key}, nil
}
