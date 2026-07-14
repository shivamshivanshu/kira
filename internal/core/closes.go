package core

import (
	"fmt"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/index"
)

func (s *Store) applyCloses(cfg *datamodel.Config, scan index.CloseScan) (closed []string, notes []datamodel.Warning, err error) {
	for _, value := range scan.Unknown {
		notes = append(notes, datamodel.Warning{Code: datamodel.WarnCloseUnknown, Args: []string{value, cfg.Commit.CloseTrailer}})
	}
	failed := false
	closeFailed := func(ref string, cause error) {
		failed = true
		notes = append(notes, datamodel.Warning{Code: datamodel.WarnCloseFailed, Args: []string{ref, cause.Error()}})
	}
	for _, cand := range scan.Candidates {
		it, _, _, resErr := s.resolveRef(cfg, cand.ULID)
		if resErr != nil {
			closeFailed(cand.ULID, resErr)
			continue
		}
		if isDoneState(cfg, it.Type, it.State) {
			continue
		}
		reopened, tsErr := reopenedSince(cand.CommitterTs, it.Updated)
		if tsErr != nil {
			closeFailed(it.Number, tsErr)
			continue
		}
		if reopened {
			continue
		}
		target := closeTargetState(cfg, it.Type)
		if target == "" {
			continue
		}
		if _, mvErr := s.Move(cfg, cand.ULID, target, MoveOpts{Force: true, Source: datamodel.SourceTrailer}); mvErr != nil {
			closeFailed(it.Number, mvErr)
			continue
		}
		closed = append(closed, it.Number)
	}
	if scan.LandedHead != "" && !failed {
		if wmErr := index.PersistLandedWatermark(s.fs().CacheDir(), scan.LandedRef, scan.LandedHead); wmErr != nil {
			return nil, nil, wmErr
		}
	}
	return closed, notes, nil
}

func reopenedSince(committerTs, updated string) (bool, error) {
	ct, err := time.Parse(time.RFC3339, committerTs)
	if err != nil {
		return false, fmt.Errorf("parsing committer date %q: %w", committerTs, err)
	}
	ut, err := time.Parse(time.RFC3339, updated)
	if err != nil {
		return false, fmt.Errorf("parsing updated timestamp %q: %w", updated, err)
	}
	return ut.After(ct), nil
}

func (s *Store) landedRef(cfg *datamodel.Config) string {
	if cfg.Git.LandedRef != "" {
		return cfg.Git.LandedRef
	}
	if ref, err := s.repo().Output("rev-parse", "--abbrev-ref", "origin/HEAD"); err == nil && ref != "" {
		return ref
	}
	return "main"
}

func closeTargetState(cfg *datamodel.Config, typ string) string {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return ""
	}
	if wf.CloseTarget != "" {
		return wf.CloseTarget
	}
	key, _ := firstStateInCategory(wf, datamodel.CategoryDone)
	return key
}
