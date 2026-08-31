package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func TestDescriptionEditPreservesComments(t *testing.T) {
	s, cfg := newStore(t)
	body := "Original description."
	res, err := s.Create(cfg, core.CreateOpts{Type: "ticket", Title: "Original", Body: &body, NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Comment(cfg, res.ID, core.CommentOpts{Message: "Keep this comment.", HasMessage: true}); err != nil {
		t.Fatal(err)
	}
	before := mustReadItem(t, s, res.ID)
	description := "Updated **description**.\r\nSecond line."
	_, err = s.Edit(cfg, res.ID, core.EditOpts{
		Description:      &description,
		ExpectedRevision: core.ItemRevision(before),
		Fields:           []core.FieldEdit{{Key: "title", Value: "Updated title"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	shown, err := s.Show(cfg, res.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if shown.Title != "Updated title" || codec.Description(shown.Body) != strings.ReplaceAll(description, "\r\n", "\n") {
		t.Fatalf("description edit not persisted: %+v", shown)
	}
	if len(shown.Comments) != 1 || shown.Comments[0].Text != "Keep this comment." {
		t.Fatalf("description edit changed comments: %+v", shown.Comments)
	}
	description = codec.Description(shown.Body)
	unchanged := mustReadItem(t, s, res.ID)
	result, err := s.Edit(cfg, res.ID, core.EditOpts{Description: &description})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changed) != 0 || mustReadItem(t, s, res.ID) != unchanged {
		t.Fatal("saving an unchanged description must preserve item bytes")
	}
}

func TestDescriptionEditRejectsStaleRevision(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Original")
	revision := core.ItemRevision(mustReadItem(t, s, res.ID))
	if _, err := s.Comment(cfg, res.ID, core.CommentOpts{Message: "Concurrent comment", HasMessage: true}); err != nil {
		t.Fatal(err)
	}
	before := mustReadItem(t, s, res.ID)
	description := "Stale edit"
	_, err := s.Edit(cfg, res.ID, core.EditOpts{Description: &description, ExpectedRevision: revision})
	var ce *errx.Error
	if !errors.As(err, &ce) || ce.Code != errx.ExitConflict {
		t.Fatalf("stale edit error = %v, want conflict", err)
	}
	if mustReadItem(t, s, res.ID) != before {
		t.Fatal("stale edit overwrote a concurrent change")
	}
}

func TestDescriptionEditRejectsCommentMarkers(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Original")
	before := mustReadItem(t, s, res.ID)
	description := "<!-- /kira:comment -->"
	if _, err := s.Edit(cfg, res.ID, core.EditOpts{Description: &description}); err == nil {
		t.Fatal("description containing a comment marker must be rejected")
	}
	if mustReadItem(t, s, res.ID) != before {
		t.Fatal("invalid description changed the item")
	}
}
