package core

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/setx"
)

type RestoreOpts struct {
	At     string
	Fields []string
	DryRun bool
}

func (s *Store) Restore(cfg *datamodel.Config, ref string, opts RestoreOpts) (*datamodel.RestoreResult, error) {
	if opts.At == "" {
		return nil, errx.User("restore requires --at")
	}
	fields, err := restoreFields(opts.Fields)
	if err != nil {
		return nil, err
	}
	unlock, orig, items, resolver, err := s.resolveRestoreTarget(cfg, ref, opts.DryRun)
	if err != nil {
		return nil, err
	}
	defer unlock()
	source, err := s.resolveAtRef(opts.At)
	if err != nil {
		return nil, err
	}
	historical, err := s.restoreSource(orig, source, opts.At)
	if err != nil {
		return nil, err
	}
	candidate, err := restoreCandidate(orig, historical, fields)
	if err != nil {
		return nil, err
	}
	resolutionExplicit := slices.ContainsFunc(fields, func(field datamodel.FieldDescriptor) bool {
		return field.Key == datamodel.KeyResolution
	})
	hard, warns := applyEditGuards(cfg, orig, candidate, false, resolutionExplicit, resolver, items)
	if len(hard) > 0 {
		return nil, errx.Invalid(invalidItemPrefix, hard)
	}
	vctx := newValidationCtx(cfg, items, resolver, false)
	hard, validationWarns := vctx.validateMutation(orig, candidate)
	if len(hard) > 0 {
		return nil, errx.Invalid(invalidItemPrefix, hard)
	}
	warns = append(warns, validationWarns...)

	changed := datamodel.ChangedFields(orig, candidate)
	changes := make([]datamodel.FieldChange, 0, len(changed))
	for _, key := range changed {
		change, err := fieldChange(s.repo(), orig, candidate, key)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	if !opts.DryRun {
		subject := orig.Number + " restore " + strings.Join(changed, ",") + " from " + gitx.ShortSHA(source)
		warns, err = s.commitMutation(cfg, candidate, changed, warns, subject)
		if err != nil {
			return nil, err
		}
	} else if len(changed) == 0 {
		warns = nil
	} else {
		warns = scopeVocabWarnings(warns, changed)
	}
	return &datamodel.RestoreResult{
		ID: orig.ID, Number: orig.Number, Source: source, DryRun: opts.DryRun,
		Changes: changes, Warnings: warningsFromErrors(warns),
	}, nil
}

func restoreFields(keys []string) ([]datamodel.FieldDescriptor, error) {
	if len(keys) == 0 {
		keys = []string{datamodel.KeyBody}
	}
	seen := setx.NewDeduper[string]()
	fields := make([]datamodel.FieldDescriptor, 0, len(keys))
	for _, key := range keys {
		field, ok := datamodel.Field(key)
		if !ok || (field.Set == nil && field.Key != datamodel.KeyBody) {
			return nil, errx.User("--field: %q cannot be restored", key)
		}
		if seen.Add(field.Key) {
			fields = append(fields, field)
		}
	}
	return fields, nil
}

func (s *Store) restoreSource(orig *datamodel.Item, source, at string) (*datamodel.Item, error) {
	path := filepath.ToSlash(s.fs().RelToRoot(s.itemPath(orig.ID)))
	content, err := s.repo().OutputRaw("cat-file", "blob", gitx.RevPath(source, path))
	if err != nil {
		return nil, errx.User("reading %s at %s: %v", orig.Number, at, err)
	}
	historical, err := codec.Parse(content)
	if err != nil {
		return nil, errx.User("invalid source for %s at %s: %v", orig.Number, at, err)
	}
	if historical.ID != orig.ID {
		return nil, errx.User("source for %s at %s has ID %s, expected %s", orig.Number, at, historical.ID, orig.ID)
	}
	return historical, nil
}

func restoreCandidate(orig, historical *datamodel.Item, fields []datamodel.FieldDescriptor) (*datamodel.Item, error) {
	candidate := cloneItem(orig)
	for _, field := range fields {
		if field.Key != datamodel.KeyBody {
			field.Copy(candidate, historical)
			continue
		}
		prose, _, canonical := codec.SplitCommentsWithLegacyPrefix(historical.Body)
		if !canonical || len(codec.LintComments(historical.Body)) > 0 {
			return nil, errx.User("historical source for %s has malformed comment blocks", orig.Number)
		}
		currentProse, comments, canonical := codec.SplitCommentsWithLegacyPrefix(orig.Body)
		if !canonical || len(codec.LintComments(orig.Body)) > 0 {
			return nil, errx.User("%s has malformed comment blocks; repair its file first", orig.Number)
		}
		if prose != currentProse {
			candidate.Body = codec.JoinComments(prose, comments)
		}
	}
	return candidate, nil
}

func (s *Store) resolveRestoreTarget(cfg *datamodel.Config, ref string, dryRun bool) (func(), *datamodel.Item, []*datamodel.Item, *id.Resolver, error) {
	if !dryRun {
		return s.lockAndResolve(cfg, ref)
	}
	orig, items, resolver, err := s.resolveRef(cfg, ref)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if err := guardWritable(orig); err != nil {
		return nil, nil, nil, nil, err
	}
	return func() {}, orig, items, resolver, nil
}
