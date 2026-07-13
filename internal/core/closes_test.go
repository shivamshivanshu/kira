package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
		if len(notes) != 1 || !strings.Contains(notes[0], "failed to close") {
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
		if len(notes) != 1 || !strings.Contains(notes[0], "unknown ticket KIRA-404") {
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
