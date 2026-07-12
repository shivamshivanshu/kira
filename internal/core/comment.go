package core

import (
	"os"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// CommentOpts are the comment inputs (docs/design/04-cli.md comment). When
// HasMessage is false the text is gathered from $EDITOR.
type CommentOpts struct {
	Message    string
	HasMessage bool
}

// CommentResult reports the appended comment. Its --json shape is not pinned by
// the doc set; this is the chosen contract.
type CommentResult struct {
	ID        string `json:"id"`
	Number    string `json:"number"`
	CommentID string `json:"comment_id"`
}

// Comment appends an anchored comment block to ref's body
// (docs/design/02-data-model.md §4). Unlike every other mutation it does not go
// through the mutate pipeline: it is a pure byte-suffix append (the original
// file content is an exact prefix of the result) and deliberately does not bump
// `updated` or reserialize frontmatter, so concurrent comments on two branches
// stay disjoint appended regions and merge cleanly by construction. This is the
// documented exception to §1's "updated on every mutation" rule.
func (s *Store) Comment(cfg *config.Config, ref string, opts CommentOpts) (*CommentResult, error) {
	release, orig, _, err := s.lockAndResolve(cfg, ref)
	if err != nil {
		return nil, err
	}
	defer release()

	text, err := s.commentText(opts)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(s.itemPath(orig.ID))
	if err != nil {
		return nil, userErr("reading %s: %v", orig.Number, err)
	}

	c := item.Comment{
		ID:     id.Mint().String(),
		Author: s.gitAuthor(),
		Ts:     time.Now().Format(time.RFC3339),
		Body:   text,
	}
	path, err := s.writeItemRaw(orig.ID, item.AppendComment(string(raw), c))
	if err != nil {
		return nil, err
	}
	subject := "kira: " + orig.Number + " comment"
	if err := s.finalize(cfg.Commit.Mode, cfg.Commit.Trailer, subject, orig.Number, path); err != nil {
		return nil, err
	}
	return &CommentResult{ID: orig.ID, Number: orig.Number, CommentID: c.ID}, nil
}

// commentText returns the comment body from -m or, when absent, from $EDITOR.
// An empty body is rejected either way — an empty comment is never recorded.
func (s *Store) commentText(opts CommentOpts) (string, error) {
	if opts.HasMessage {
		if strings.TrimSpace(opts.Message) == "" {
			return "", userErr("empty comment")
		}
		return opts.Message, nil
	}
	content, err := runEditor("", func(c string) []error {
		if strings.TrimSpace(c) == "" {
			return []error{userErr("empty comment")}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(content, "\n"), nil
}

// gitAuthor derives a whitespace-free comment author from the repo's git
// identity: user.name (internal whitespace collapsed to '-' so it stays a
// single marker token), falling back to user.email, then "unknown". Comment
// authors are not vocabulary-checked, so this needs no people.known entry.
func (s *Store) gitAuthor() string {
	for _, key := range []string{"user.name", "user.email"} {
		if v, err := s.git("config", key); err == nil {
			if f := strings.Fields(v); len(f) > 0 {
				return strings.Join(f, "-")
			}
		}
	}
	return "unknown"
}
