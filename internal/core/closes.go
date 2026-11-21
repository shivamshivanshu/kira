package core

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/timex"
)

func (s *Store) applyCloses(cfg *datamodel.Config, scan index.CloseScan) (closed []string, notes []datamodel.Warning, err error) {
	for _, value := range scan.Unknown {
		notes = append(notes, datamodel.Warning{Code: datamodel.WarnCloseUnknown, Args: []string{value, datamodel.CloseTrailerKey}})
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
		target, _, eligibilityErr := closeEligibility(cfg, it, cand.CommitterTs)
		if eligibilityErr != nil {
			closeFailed(it.Number, eligibilityErr)
			continue
		}
		if target == "" {
			continue
		}
		mv, mvErr := b.Move(cand.ULID, target, MoveOpts{Force: true})
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
			notes = append(notes, literalWarnings([]string{
				fmt.Sprintf("recording the close-scan watermark failed (%v); the next scan will rescan from the previous one", wmErr),
			})...)
		}
	}
	return closed, notes, nil
}

func closeEligibility(cfg *datamodel.Config, it *datamodel.Item, committerTs string) (target, outcome string, err error) {
	if itemIsDone(cfg, it) {
		return "", datamodel.ReviewCloseAlreadyClosed, nil
	}
	touched, err := touchedSince(committerTs, it.Updated)
	if err != nil {
		return "", datamodel.ReviewCloseInvalidTimestamp, err
	}
	if touched {
		return "", datamodel.ReviewCloseTicketNewer, nil
	}
	target = workflowFor(cfg, it).CloseTarget()
	if target == "" {
		return "", datamodel.ReviewCloseNoTarget, nil
	}
	return target, datamodel.ReviewCloseEligible, nil
}

func touchedSince(committerTs, updated string) (bool, error) {
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
