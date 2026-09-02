package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/editorx"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/setx"
)

type FieldEdit struct {
	Key   string
	Value string
}

type EditOpts struct {
	Fields           []FieldEdit
	FromFile         string
	Force            bool
	Stdio            editorx.Stdio
	Description      *string
	ExpectedRevision string
}

func ItemRevision(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func (s *Store) diskRevision(orig *datamodel.Item) (string, error) {
	content, err := os.ReadFile(s.itemPath(orig.ID))
	if err != nil {
		return "", errx.User("reading %s: %v", orig.Number, err)
	}
	return ItemRevision(string(content)), nil
}

func (s *Store) requireRevision(orig *datamodel.Item, expected, hint string) error {
	rev, err := s.diskRevision(orig)
	if err != nil {
		return err
	}
	if rev != expected {
		return errx.Conflict("%s changed on disk during edit", orig.Number).WithHint("%s", hint)
	}
	return nil
}

func (s *Store) Edit(cfg *datamodel.Config, ref string, opts EditOpts) (*datamodel.MutationResult, error) {
	var edited *datamodel.Item
	var snapshot string
	if len(opts.Fields) == 0 && opts.FromFile == "" && opts.Description == nil {
		content, rev, err := s.editorContent(cfg, ref, opts)
		if err != nil {
			return nil, err
		}
		var errs []error
		edited, errs = parseFullItem(content)
		if len(errs) > 0 {
			return nil, errx.Invalid(invalidItemPrefix, errs)
		}
		snapshot = rev
	}

	release, orig, items, resolver, err := s.lockAndResolve(cfg, ref)
	if err != nil {
		return nil, err
	}
	defer release()
	if opts.ExpectedRevision != "" {
		if err := s.requireRevision(orig, opts.ExpectedRevision, "refresh the item and reapply your changes"); err != nil {
			return nil, err
		}
	}

	if edited != nil {
		if err := s.requireRevision(orig, snapshot, "re-run the edit to start from the current version"); err != nil {
			return nil, err
		}
	}

	schemas, err := s.schemas()
	if err != nil {
		return nil, err
	}
	schema := schemas[orig.Type]
	vctx := newValidationCtx(cfg, items, resolver, schema, entityschema.ContainerTypes(schemas), opts.Force)

	var updated *datamodel.Item
	var warns []error
	finish := func(it *datamodel.Item) error {
		restoreImmutable(it, orig, schema)
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
	case len(opts.Fields) > 0 || opts.Description != nil:
		it := cloneItem(orig)
		if errs := s.applyFieldEdits(cfg, it, opts.Fields, schema); len(errs) > 0 {
			return nil, errx.Invalid(invalidItemPrefix, errs)
		}
		if opts.Description != nil {
			if err := replaceDescription(it, orig, *opts.Description); err != nil {
				return nil, err
			}
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

	changed := changedKeys(orig, updated)
	subject := cfg.Commit.SubjectPrefix + updated.Number + " edit " + strings.Join(changed, ",")
	finalWarns, err := s.commitMutation(cfg, orig, updated, changed, warns, subject, datamodel.SourceCLI)
	if err != nil {
		return nil, err
	}
	return &datamodel.MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed, Warnings: warningsFromErrors(finalWarns)}, nil
}

// EditFields applies noninteractive field edits against the batch's live state.
func (b *Batch) EditFields(ref string, fields []FieldEdit, force bool) (*datamodel.MutationResult, error) {
	orig, err := b.Resolve(ref)
	if err != nil {
		return nil, err
	}
	schema := b.schemas[orig.Type]
	var changed []string
	apply := func(it *datamodel.Item, resolver *id.Resolver, items []*datamodel.Item) (hard, warns []error) {
		if errs := b.store.applyFieldEdits(b.cfg, it, fields, schema); len(errs) > 0 {
			return errs, nil
		}
		hard, warns = applyEditGuards(b.cfg, orig, it, force, resolver, items, schema)
		changed = changedKeys(orig, it)
		return hard, warns
	}
	subject := func(it *datamodel.Item) string {
		return b.cfg.Commit.SubjectPrefix + it.Number + " edit " + strings.Join(changed, ",")
	}
	updated, changed, warns, err := b.Mutate(ref, force, apply, subject, datamodel.SourceCLI)
	if err != nil {
		return nil, err
	}
	return &datamodel.MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed, Warnings: warningsFromErrors(warns)}, nil
}

func (s *Store) editorContent(cfg *datamodel.Config, ref string, opts EditOpts) (string, string, error) {
	orig, items, resolver, err := s.resolveRef(cfg, ref)
	if err != nil {
		return "", "", err
	}
	if err := guardWritable(orig); err != nil {
		return "", "", err
	}
	// Snapshot raw bytes, not `updated`: comment appends deliberately never bump it.
	snapshot, err := s.diskRevision(orig)
	if err != nil {
		return "", "", err
	}
	schemas, err := s.schemas()
	if err != nil {
		return "", "", err
	}
	schema := schemas[orig.Type]
	vctx := newValidationCtx(cfg, items, resolver, schema, entityschema.ContainerTypes(schemas), opts.Force)
	content, err := runEditor(cfg.UI.Editor, opts.Stdio, codec.Serialize(orig), func(c string) []error {
		it, errs := parseFullItem(c)
		if len(errs) > 0 {
			return errs
		}
		restoreImmutable(it, orig, schema)
		if hard, _ := applyEditGuards(cfg, orig, it, opts.Force, resolver, items, schema); len(hard) > 0 {
			return hard
		}
		hard, _ := vctx.validateMutation(orig, it)
		return hard
	})
	return content, snapshot, err
}

func applyEditGuards(cfg *datamodel.Config, orig, it *datamodel.Item, force bool, resolver *id.Resolver, items []*datamodel.Item, schema entityschema.Schema) (hard, warns []error) {
	if hard := normalizeAndCheckRefs(it, resolver, schema); len(hard) > 0 {
		return hard, nil
	}
	if it.State == orig.State {
		return nil, nil
	}
	edited := setx.ToSet(datamodel.ChangedFields(orig, it))
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
		if pe, ok := errors.AsType[*datamodel.ParseError](err); ok {
			return it, pe.Errs
		}
		return it, []error{err}
	}
	return it, nil
}

// restoreImmutable reverts every field the editor and --from-file paths may
// not change: the builtin identity fields, plus the extras orig's schema
// declares immutable. copyExtra also covers a deletion, since a key absent
// from orig is cleared rather than left as the buffer wrote it.
func restoreImmutable(it, orig *datamodel.Item, schema entityschema.Schema) {
	it.ID = orig.ID
	it.Number = orig.Number
	it.Type = orig.Type
	it.Created = orig.Created
	it.Aliases = orig.Aliases
	for _, f := range schema.Fields {
		if f.Immutable && f.IsCustomFrontmatter() {
			copyExtra(it, orig, f.Name)
		}
	}
}

// applyFieldEdits reports one error per offending field so a caller can surface
// them together; an unresolvable @me stops the pass, since every field after it
// would fail identically.
func (s *Store) applyFieldEdits(cfg *datamodel.Config, it *datamodel.Item, fields []FieldEdit, schema entityschema.Schema) []error {
	var errs []error
	for _, fe := range fields {
		value := fe.Value
		if fe.Key == datamodel.KeyOwner || fe.Key == datamodel.KeyReporter {
			resolved, err := s.resolveMe(cfg, fe.Value)
			if err != nil {
				return append(errs, err)
			}
			value = resolved
		}
		if err := applyFieldEdit(it, fe.Key, value, schema); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func applyFieldEdit(it *datamodel.Item, key, value string, schema entityschema.Schema) error {
	if d, ok := datamodel.Field(key); ok {
		if d.Set == nil {
			return fieldEditError(key, schema)
		}
		return d.Set(it, value)
	}
	f, ok := schema.Field(key)
	if !ok || !fieldEditableViaFlag(f) {
		return fieldEditError(key, schema)
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

func fieldEditError(key string, schema entityschema.Schema) error {
	return errx.User("--field: unknown or immutable field %q", key).WithHint("%s", fieldHint(key, schema))
}

func replaceDescription(it, orig *datamodel.Item, description string) error {
	_, comments, canonical := codec.SplitComments(codec.CanonicalizeCommentBody(orig.Body))
	if !canonical {
		return errx.User("%s has non-canonical comment blocks; edit its file to preserve them", orig.Number)
	}
	prose := normalizeEOL(description)
	if len(codec.LintComments(prose)) > 0 || len(codec.ParseComments(prose)) > 0 {
		return errx.User("description cannot contain kira comment markers")
	}
	if prose != codec.Description(orig.Body) {
		it.Body = codec.JoinComments(prose, comments)
	}
	return nil
}

func cloneItem(src *datamodel.Item) *datamodel.Item {
	dst := *src
	dst.Aliases = slices.Clone(src.Aliases)
	dst.Labels = slices.Clone(src.Labels)
	dst.BlockedBy = slices.Clone(src.BlockedBy)
	dst.Links = datamodel.CloneStringListMap(src.Links)
	dst.ExtraLinks = datamodel.CloneStringListMap(src.ExtraLinks)
	dst.ExtraScalars = maps.Clone(src.ExtraScalars)
	dst.ExtraLists = datamodel.CloneStringListMap(src.ExtraLists)
	dst.ExtraBlocks = maps.Clone(src.ExtraBlocks)
	return &dst
}
