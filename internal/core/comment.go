package core

import (
	"os"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/editorx"
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

	text, err := s.commentText(cfg.UI.Editor, opts)
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
		Author: s.currentUser(cfg),
		Ts:     time.Now().Format(time.RFC3339),
		Body:   text,
	}
	content, err := codec.AppendCommentToDocument(string(raw), c)
	if err != nil {
		return nil, errx.User("appending comment to %s: %v", orig.Number, err)
	}
	path, err := s.fs().WriteItemRaw(orig.ID, content)
	if err != nil {
		return nil, err
	}
	subject := cfg.Commit.SubjectPrefix + orig.Number + " comment"
	cs := &datamodel.ChangeSet{
		Kind:    datamodel.ChangeCommented,
		Before:  orig,
		After:   orig,
		Paths:   []string{path},
		Subject: subject,
		Source:  datamodel.SourceCLI,
	}
	if err := s.commit(cfg, cs); err != nil {
		return nil, err
	}
	return &datamodel.CommentResult{ID: orig.ID, Number: orig.Number, CommentID: c.ID}, nil
}

func (s *Store) commentText(editor string, opts CommentOpts) (string, error) {
	if opts.HasMessage {
		msg := normalizeEOL(opts.Message)
		if err := validateCommentText(msg); err != nil {
			return "", err
		}
		return msg, nil
	}
	content, err := runEditor(editor, editorx.Stdio{}, "", func(c string) []error {
		if err := validateCommentText(normalizeEOL(c)); err != nil {
			return []error{err}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(normalizeEOL(content), "\n"), nil
}

func normalizeEOL(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

func validateCommentText(text string) error {
	if strings.TrimSpace(text) == "" {
		return errx.User("empty comment")
	}
	for _, line := range strings.Split(text, "\n") {
		if codec.IsCommentMarker(line) {
			return errx.User("comment text cannot contain kira comment markers")
		}
	}
	return nil
}

func (s *Store) currentUser(cfg *datamodel.Config) string {
	if id, ok := s.identity(cfg); ok {
		return id
	}
	return "unknown"
}
