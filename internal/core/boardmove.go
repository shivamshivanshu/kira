package core

import (
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func (s *Store) BoardMove(cfg *datamodel.Config, ref, targetKey string) (*datamodel.BoardMoveResult, error) {
	b, err := s.BeginBatch(cfg)
	if err != nil {
		return nil, err
	}
	defer b.Close()
	return b.BoardMove(ref, targetKey)
}

func (b *Batch) BoardMove(ref, targetKey string) (*datamodel.BoardMoveResult, error) {
	s, cfg := b.store, b.cfg
	board, ok := cfg.BoardByKey(targetKey)
	if !ok || board.Archived {
		return nil, errx.User("no such board %q", targetKey).
			WithHint("boards: %s", strings.Join(activeBoardKeys(cfg.ActiveBoards()), ", "))
	}

	orig, err := b.Resolve(ref)
	if err != nil {
		return nil, err
	}
	it := cloneItem(orig)
	if err := guardWritable(it); err != nil {
		return nil, err
	}

	from := it.Number
	if strings.EqualFold(id.KeyOf(from), board.Key) {
		return nil, errx.User("%s is already on board %s", from, board.Key)
	}

	if _, err := id.ParseULID(it.ID); err != nil {
		return nil, errx.User("item %s has an invalid ULID: %v", from, err)
	}
	to := id.NewAllocator(storage.Snapshot(cfg.Project.Key, b.items), board.Key).Alloc()

	it.Number = to
	it.Aliases = slices.DeleteFunc(it.Aliases, func(a string) bool { return strings.EqualFold(a, to) })
	if !slices.ContainsFunc(it.Aliases, func(a string) bool { return strings.EqualFold(a, from) }) {
		it.Aliases = append(it.Aliases, from)
	}
	// Deliberately bypasses the mutation pipeline: a renumber is identity
	// bookkeeping, not activity, so Updated must not move (close-reopen
	// detection compares against it).
	path, err := s.fs().WriteItem(it)
	if err != nil {
		return nil, err
	}
	subject := "board " + from + " -> " + to
	b.replace(it)
	b.resolver = id.NewResolver(storage.Snapshot(cfg.Project.Key, b.items))
	if _, err := s.finalize(cfg.Commit.Mode, commitSpec{subject: subject, trailerNumber: to}, path); err != nil {
		return nil, err
	}
	return &datamodel.BoardMoveResult{ID: it.ID, From: from, To: to, Board: board.Key}, nil
}
