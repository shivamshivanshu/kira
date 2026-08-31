package core

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/timex"
)

func (s *Store) applyCloses(cfg *datamodel.Config, scan index.CloseScan) (closed []string, notes []datamodel.Warning, err error) {
	for _, value := range scan.Unknown {
		notes = append(notes, datamodel.Warning{Code: datamodel.WarnCloseUnknown, Args: []string{value, cfg.Commit.CloseTrailer}})
	}
	b, err := s.BeginBatch(cfg)
	if err != nil {
		return nil, nil, err
	}
	defer b.Close()
	failed := false
	closeFailed := func(ref string, cause error) {
		failed = true
		notes = append(notes, datamodel.Warning{Code: datamodel.WarnCloseFailed, Args: []string{ref, cause.Error()}})
	}
	for _, cand := range scan.Candidates {
		it, resErr := b.Resolve(cand.ULID)
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
		mv, mvErr := b.Move(cand.ULID, target, MoveOpts{Force: true, Source: datamodel.SourceTrailer})
		if mvErr != nil {
			closeFailed(it.Number, mvErr)
			continue
		}
		notes = append(notes, literalWarnings(mv.Warnings)...)
		closed = append(closed, it.Number)
	}
	if scan.LandedHead != "" && !failed {
		// The watermark lives in the disposable cache: failing to record it
		// only costs a rescan, while erroring here would hide closes that are
		// already committed.
		if wmErr := index.PersistLandedWatermark(s.fs().CacheDir(), scan.LandedRef, scan.LandedHead); wmErr != nil {
			notes = append(notes, datamodel.Warning{Code: datamodel.WarnLiteral, Args: []string{fmt.Sprintf("recording the close-scan watermark failed (%v); the next scan will rescan from the previous one", wmErr)}})
		}
	}
	return closed, notes, nil
}

func reopenedSince(committerTs, updated string) (bool, error) {
	cmp, ctOK, utOK := timex.CompareRFC3339(committerTs, updated)
	if !ctOK {
		return false, fmt.Errorf("parsing committer date %q", committerTs)
	}
	if !utOK {
		return false, fmt.Errorf("parsing updated timestamp %q", updated)
	}
	return cmp < 0, nil
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
