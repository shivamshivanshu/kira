package core

import (
	"errors"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/editorx"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

type FieldEdit struct {
	Key   string
	Value string
}

type EditOpts struct {
	Fields   []FieldEdit
	FromFile string
	Force    bool
	Stdio    editorx.Stdio
}

func (s *Store) Edit(cfg *datamodel.Config, ref string, opts EditOpts) (*datamodel.MutationResult, error) {
	var edited *datamodel.Item
	var snapshot string
	if len(opts.Fields) == 0 && opts.FromFile == "" {
		content, updatedAt, err := s.editorContent(cfg, ref, opts)
		if err != nil {
			return nil, err
		}
		var errs []error
		edited, errs = parseFullItem(content)
		if len(errs) > 0 {
			return nil, errx.Invalid(errs)
		}
		snapshot = updatedAt
	}

	release, orig, items, resolver, err := s.lockAndResolve(cfg, ref)
	if err != nil {
		return nil, err
	}
	defer release()

	if edited != nil && orig.Updated != snapshot {
		return nil, errx.Conflict("%s changed on disk during edit", orig.Number).WithHint("re-run the edit to start from the current version")
	}

	var updated *datamodel.Item
	var warns []error
	finish := func(it *datamodel.Item) error {
		restoreImmutable(it, orig)
		guardErrs, guardWarns := applyEditGuards(cfg, orig, it, opts.Force, resolver, items)
		if len(guardErrs) > 0 {
			return errx.Invalid(guardErrs)
		}
		errs, w := validateMutation(cfg, orig, it, resolver, items, opts.Force)
		if len(errs) > 0 {
			return errx.Invalid(errs)
		}
		updated, warns = it, append(guardWarns, w...)
		return nil
	}

	switch {
	case len(opts.Fields) > 0:
		it := cloneItem(orig)
		var errs []error
		for _, fe := range opts.Fields {
			value := fe.Value
			if fe.Key == datamodel.KeyOwner || fe.Key == datamodel.KeyReporter {
				if value, err = s.resolveMe(cfg, fe.Value); err != nil {
					return nil, err
				}
			}
			if err := applyFieldEdit(it, fe.Key, value); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return nil, errx.Invalid(errs)
		}
		if err := finish(it); err != nil {
			return nil, err
		}
	case opts.FromFile != "":
		content, err := readSource(opts.FromFile)
		if err != nil {
			return nil, err
		}
		it, errs := parseFullItem(stripErrorBanner(content))
		if len(errs) > 0 {
			return nil, errx.Invalid(errs)
		}
		if err := finish(it); err != nil {
			return nil, err
		}
	default:
		if err := finish(edited); err != nil {
			return nil, err
		}
	}

	changed := datamodel.ChangedFields(orig, updated)
	subject := cfg.Commit.SubjectPrefix + updated.Number + " edit " + strings.Join(changed, ",")
	if err := s.commitMutation(cfg, orig, updated, changed, warns, subject, datamodel.SourceCLI); err != nil {
		return nil, err
	}
	return &datamodel.MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed}, nil
}

func (s *Store) editorContent(cfg *datamodel.Config, ref string, opts EditOpts) (string, string, error) {
	orig, items, resolver, err := s.resolveRef(cfg, ref)
	if err != nil {
		return "", "", err
	}
	if err := guardWritable(orig); err != nil {
		return "", "", err
	}
	content, err := runEditor(cfg.UI.Editor, opts.Stdio, codec.Serialize(orig), func(c string) []error {
		it, errs := parseFullItem(c)
		if len(errs) > 0 {
			return errs
		}
		restoreImmutable(it, orig)
		if hard, _ := applyEditGuards(cfg, orig, it, opts.Force, resolver, items); len(hard) > 0 {
			return hard
		}
		hard, _ := validateMutation(cfg, orig, it, resolver, items, opts.Force)
		return hard
	})
	return content, orig.Updated, err
}

func applyEditGuards(cfg *datamodel.Config, orig, it *datamodel.Item, force bool, resolver *id.Resolver, items []*datamodel.Item) (hard, warns []error) {
	if hard := normalizeAndCheckRefs(it, resolver); len(hard) > 0 {
		return hard, nil
	}
	if it.State == orig.State {
		return nil, nil
	}
	edited := make(map[string]bool)
	for _, f := range datamodel.ChangedFields(orig, it) {
		edited[f] = true
	}
	mo := MoveOpts{Force: force}
	if edited[datamodel.KeyResolution] && it.Resolution != nil {
		mo.Resolution = *it.Resolution
	}
	h, w, wipWarns := applyStateChange(cfg, it, orig.State, mo, edited, items)
	if len(h) > 0 {
		return h, nil
	}
	return nil, append(w, wipWarns...)
}

func parseFullItem(content string) (*datamodel.Item, []error) {
	it, err := codec.Parse(content)
	if it == nil {
		return nil, []error{err}
	}
	if err != nil {
		var pe *datamodel.ParseError
		if errors.As(err, &pe) {
			return it, pe.Errs
		}
		return it, []error{err}
	}
	return it, nil
}

func restoreImmutable(it, orig *datamodel.Item) {
	it.ID = orig.ID
	it.Number = orig.Number
	it.Type = orig.Type
	it.Created = orig.Created
	it.Aliases = orig.Aliases
}

func applyFieldEdit(it *datamodel.Item, key, value string) error {
	d, ok := datamodel.Field(key)
	if !ok || d.Set == nil {
		return errx.User("--field: unknown or immutable field %q", key).WithHint("%s", fieldHint(key))
	}
	return d.Set(it, value)
}

func cloneItem(src *datamodel.Item) *datamodel.Item {
	dst := *src
	dst.Aliases = slices.Clone(src.Aliases)
	dst.Labels = slices.Clone(src.Labels)
	dst.BlockedBy = slices.Clone(src.BlockedBy)
	dst.Links = datamodel.CloneLinks(src.Links)
	return &dst
}
