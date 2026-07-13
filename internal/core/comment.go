package core

import (
	"os"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

type CommentOpts struct {
	Message    string
	HasMessage bool
}

// Comment bypasses the mutate pipeline on purpose: a pure byte-suffix append
// that never bumps `updated` or reserializes frontmatter, so concurrent
// comments on two branches stay disjoint appended regions and merge cleanly.
func (s *Store) Comment(cfg *datamodel.Config, ref string, opts CommentOpts) (*datamodel.CommentResult, error) {
	if !opts.HasMessage {
		orig, _, _, err := s.resolveRef(cfg, ref)
		if err != nil {
			return nil, err
		}
		if err := guardWritable(orig); err != nil {
			return nil, err
		}
	}

	text, err := s.commentText(opts)
	if err != nil {
		return nil, err
	}

	release, orig, _, _, err := s.lockAndResolve(cfg, ref)
	if err != nil {
		return nil, err
	}
	defer release()

	raw, err := os.ReadFile(s.itemPath(orig.ID))
	if err != nil {
		return nil, errx.User("reading %s: %v", orig.Number, err)
	}

	c := datamodel.Comment{
		ID:     id.Mint().String(),
		Author: s.commentAuthorToken(),
		Ts:     time.Now().Format(time.RFC3339),
		Body:   text,
	}
	path, err := s.writeItemRaw(orig.ID, codec.AppendComment(string(raw), c))
	if err != nil {
		return nil, err
	}
	subject := "kira: " + orig.Number + " comment"
	cs := &datamodel.ChangeSet{
		Kind:    datamodel.ChangeCommented,
		Before:  orig,
		After:   orig,
		Paths:   []string{path},
		Subject: subject,
		Source:  datamodel.SourceCLI,
	}
	if err := s.commit(cfg, cs, nil); err != nil {
		return nil, err
	}
	return &datamodel.CommentResult{ID: orig.ID, Number: orig.Number, CommentID: c.ID}, nil
}

func (s *Store) commentText(opts CommentOpts) (string, error) {
	if opts.HasMessage {
		if strings.TrimSpace(opts.Message) == "" {
			return "", errx.User("empty comment")
		}
		return opts.Message, nil
	}
	content, err := runEditor("", func(c string) []error {
		if strings.TrimSpace(c) == "" {
			return []error{errx.User("empty comment")}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(content, "\n"), nil
}

func (s *Store) commentAuthorToken() string {
	for _, key := range []string{"user.name", "user.email"} {
		if v, err := s.repo().Output("config", key); err == nil {
			if f := strings.Fields(v); len(f) > 0 {
				return strings.Join(f, "-")
			}
		}
	}
	return "unknown"
}
