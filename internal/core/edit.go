package core

import (
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/editorx"
	"github.com/shivamshivanshu/kira/internal/entityschema"
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
			return nil, errx.Invalid(invalidItemPrefix, errs)
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

	schemas, err := s.schemas()
	if err != nil {
		return nil, err
	}
	schema := schemas[orig.Type]
	vctx := newValidationCtx(cfg, items, resolver, schema, opts.Force)

	var updated *datamodel.Item
	var warns []error
	finish := func(it *datamodel.Item) error {
		restoreImmutable(it, orig)
		guardErrs, guardWarns := applyEditGuards(cfg, orig, it, opts.Force, resolver, items, schema)
		if len(guardErrs) > 0 {
			return errx.Invalid(invalidItemPrefix, guardErrs)
		}
		errs, w := vctx.validateMutation(orig, it)
		if len(errs) > 0 {
			return errx.Invalid(invalidItemPrefix, errs)
		}
		updated, warns = it, append(guardWarns, w...)
		return nil
	}

	switch {
	case len(opts.Fields) > 0:
		it := cloneItem(orig)
		if err := s.applyFieldEdits(cfg, it, opts.Fields, schema); err != nil {
			return nil, err
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
			return nil, errx.Invalid(invalidItemPrefix, errs)
		}
		if err := finish(it); err != nil {
			return nil, err
		}
	default:
		if err := finish(edited); err != nil {
			return nil, err
		}
	}

	changed := append(datamodel.ChangedFields(orig, updated), changedExtraKeys(orig, updated)...)
	subject := cfg.Commit.SubjectPrefix + updated.Number + " edit " + strings.Join(changed, ",")
	finalWarns, err := s.commitMutation(cfg, orig, updated, changed, warns, subject, datamodel.SourceCLI)
	if err != nil {
		return nil, err
	}
	return &datamodel.MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed, Warnings: warningsFromErrors(finalWarns)}, nil
}

func (s *Store) editorContent(cfg *datamodel.Config, ref string, opts EditOpts) (string, string, error) {
	orig, items, resolver, err := s.resolveRef(cfg, ref)
	if err != nil {
		return "", "", err
	}
	if err := guardWritable(orig); err != nil {
		return "", "", err
	}
	schemas, err := s.schemas()
	if err != nil {
		return "", "", err
	}
	schema := schemas[orig.Type]
	vctx := newValidationCtx(cfg, items, resolver, schema, opts.Force)
	content, err := runEditor(cfg.UI.Editor, opts.Stdio, codec.Serialize(orig), func(c string) []error {
		it, errs := parseFullItem(c)
		if len(errs) > 0 {
			return errs
		}
		restoreImmutable(it, orig)
		if hard, _ := applyEditGuards(cfg, orig, it, opts.Force, resolver, items, schema); len(hard) > 0 {
			return hard
		}
		hard, _ := vctx.validateMutation(orig, it)
		return hard
	})
	return content, orig.Updated, err
}

func applyEditGuards(cfg *datamodel.Config, orig, it *datamodel.Item, force bool, resolver *id.Resolver, items []*datamodel.Item, schema entityschema.Schema) (hard, warns []error) {
	if hard := normalizeAndCheckRefs(it, resolver, schema); len(hard) > 0 {
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

func (s *Store) applyFieldEdits(cfg *datamodel.Config, it *datamodel.Item, fields []FieldEdit, schema entityschema.Schema) error {
	var errs []error
	for _, fe := range fields {
		value := fe.Value
		if fe.Key == datamodel.KeyOwner || fe.Key == datamodel.KeyReporter {
			resolved, err := s.resolveMe(cfg, fe.Value)
			if err != nil {
				return err
			}
			value = resolved
		}
		if err := applyFieldEdit(it, fe.Key, value, schema); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errx.Invalid(invalidItemPrefix, errs)
	}
	return nil
}

func applyFieldEdit(it *datamodel.Item, key, value string, schema entityschema.Schema) error {
	if d, ok := datamodel.Field(key); ok {
		if d.Set == nil {
			return errx.User("--field: unknown or immutable field %q", key).WithHint("%s", fieldHint(key, schema))
		}
		return d.Set(it, value)
	}
	f, ok := schema.Field(key)
	if !ok || !fieldEditableViaFlag(f) {
		return errx.User("--field: unknown or immutable field %q", key).WithHint("%s", fieldHint(key, schema))
	}
	if f.Placement == entityschema.PlacementBody {
		if datamodel.ContainsSectionHeading(value) {
			return errx.User("--field %q: a body-section value cannot contain a \"## \" heading line", key)
		}
		it.Body = datamodel.UpsertBodySection(it.Body, f.Section, value)
		return nil
	}
	return setSchemaExtra(it, f, value)
}

func cloneItem(src *datamodel.Item) *datamodel.Item {
	dst := *src
	dst.Aliases = slices.Clone(src.Aliases)
	dst.Labels = slices.Clone(src.Labels)
	dst.BlockedBy = slices.Clone(src.BlockedBy)
	dst.Links = datamodel.CloneStringListMap(src.Links)
	dst.ExtraScalars = maps.Clone(src.ExtraScalars)
	dst.ExtraLists = datamodel.CloneStringListMap(src.ExtraLists)
	return &dst
}
