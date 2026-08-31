package core

import (
	"context"
	"encoding/hex"
	"errors"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func (s *Store) CommitPatch(ctx context.Context, cfg *datamodel.Config, ref, sha string) (string, error) {
	sha, err := normalizeCommitSHA(sha)
	if err != nil {
		return "", err
	}
	item, err := s.Show(cfg, ref, "")
	if err != nil {
		return "", err
	}
	return s.commitPatch(ctx, item, sha)
}

func (d *ItemDetail) CommitPatch(ctx context.Context, sha string) (string, error) {
	sha, err := normalizeCommitSHA(sha)
	if err != nil {
		return "", err
	}
	return d.store.commitPatch(ctx, d.Item, sha)
}

func normalizeCommitSHA(sha string) (string, error) {
	if len(sha) != 40 && len(sha) != 64 {
		return "", errx.User("a full commit SHA is required")
	}
	if _, err := hex.DecodeString(sha); err != nil {
		return "", errx.User("invalid commit SHA")
	}
	return strings.ToLower(sha), nil
}

func (s *Store) commitPatch(ctx context.Context, item *datamodel.ShowResult, sha string) (string, error) {
	matches := func(c datamodel.CommitLink) bool { return c.SHA == sha }
	if !slices.ContainsFunc(item.LinkedCommits, matches) && !slices.ContainsFunc(item.ReferencedBy, matches) {
		return "", errx.User("commit %s is not linked to %s", gitx.ShortSHA(sha), item.Number)
	}
	patch, err := s.repo().ShowPatch(ctx, sha)
	if errors.Is(err, gitx.ErrPatchTooLarge) {
		return "", errx.User("%v", err).WithHint("view the complete diff in your terminal with git show %s", sha)
	}
	if err != nil {
		return "", errx.User("reading commit: %v", err)
	}
	return patch, nil
}
