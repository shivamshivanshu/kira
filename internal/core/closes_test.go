package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/index"
)

func landedWatermark(t *testing.T, s *Store) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(s.fs().CacheDir(), "meta.json"))
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	var m struct {
		TrailerWatermarks map[string]string `json:"trailer_watermarks"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse meta: %v", err)
	}
	return m.TrailerWatermarks["main"]
}

func TestApplyClosesWatermark(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	cfg.Commit.Mode = datamodel.CommitManual

	created, err := s.Create(cfg, CreateOpts{Type: "ticket", Title: "closeme", NoEdit: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := index.Refresh(s.fs(), s.repo(), indexOptions(cfg), false); err != nil {
		t.Fatalf("index refresh: %v", err)
	}

	const future = "2999-01-01T00:00:00Z"

	t.Run("failed close leaves watermark unadvanced", func(t *testing.T) {
		scan := index.CloseScan{
			LandedRef:  "main",
			LandedHead: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
			Candidates: []index.CloseCandidate{{ULID: "01J8X8Q7RZTN5Y3VXW2A9K4E7Z", CommitterTs: future}},
		}
		closed, notes, err := s.applyCloses(cfg, scan)
		if err != nil {
			t.Fatalf("applyCloses: %v", err)
		}
		if len(closed) != 0 {
			t.Fatalf("failed close reported closed=%v", closed)
		}
		if len(notes) != 1 || notes[0].Code != datamodel.WarnCloseFailed {
			t.Fatalf("want a single failure note, got %v", notes)
		}
		if wm := landedWatermark(t, s); wm == scan.LandedHead {
			t.Fatalf("watermark advanced to %q despite a failed close", wm)
		}
	})

	t.Run("unknown ticket surfaced as a note", func(t *testing.T) {
		closed, notes, err := s.applyCloses(cfg, index.CloseScan{Unknown: []string{"KIRA-404"}})
		if err != nil {
			t.Fatalf("applyCloses: %v", err)
		}
		if len(closed) != 0 {
			t.Fatalf("unknown ticket reported closed=%v", closed)
		}
		if len(notes) != 1 || notes[0].Code != datamodel.WarnCloseUnknown || notes[0].Args[0] != "KIRA-404" {
			t.Fatalf("want an unknown-ticket note, got %v", notes)
		}
	})

	t.Run("successful close advances watermark", func(t *testing.T) {
		scan := index.CloseScan{
			LandedRef:  "main",
			LandedHead: "cafef00dcafef00dcafef00dcafef00dcafef00d",
			Candidates: []index.CloseCandidate{{ULID: created.ID, CommitterTs: future}},
		}
		closed, _, err := s.applyCloses(cfg, scan)
		if err != nil {
			t.Fatalf("applyCloses: %v", err)
		}
		if len(closed) != 1 || closed[0] != created.Number {
			t.Fatalf("want closed=[%s], got %v", created.Number, closed)
		}
		if wm := landedWatermark(t, s); wm != scan.LandedHead {
			t.Fatalf("watermark = %q, want %q after a successful close", wm, scan.LandedHead)
		}
	})
}

func TestApplyClosesMultipleCandidates(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	cfg.Commit.Mode = datamodel.CommitManual

	a, err := s.Create(cfg, CreateOpts{Type: "ticket", Title: "a", NoEdit: true})
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	b, err := s.Create(cfg, CreateOpts{Type: "ticket", Title: "b", NoEdit: true})
	if err != nil {
		t.Fatalf("create b: %v", err)
	}
	if _, err := index.Refresh(s.fs(), s.repo(), indexOptions(cfg), false); err != nil {
		t.Fatalf("index refresh: %v", err)
	}

	const future = "2999-01-01T00:00:00Z"
	scan := index.CloseScan{
		LandedRef:  "main",
		LandedHead: "0ddba11c0ddba11c0ddba11c0ddba11c0ddba11c",
		Candidates: []index.CloseCandidate{
			{ULID: a.ID, CommitterTs: future},
			{ULID: b.ID, CommitterTs: future},
		},
	}
	closed, notes, err := s.applyCloses(cfg, scan)
	if err != nil {
		t.Fatalf("applyCloses: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("unexpected notes: %v", notes)
	}
	want := map[string]bool{a.Number: true, b.Number: true}
	if len(closed) != 2 || !want[closed[0]] || !want[closed[1]] || closed[0] == closed[1] {
		t.Fatalf("want both %v closed, got %v", want, closed)
	}
}
