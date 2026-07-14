package core

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

type FieldEdit struct {
	Key   string
	Value string
}

type EditOpts struct {
	Fields   []FieldEdit
	FromFile string
	Force    bool
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

	assemble := func(it *datamodel.Item) (*datamodel.Item, []error, []error) {
		restoreImmutable(it, orig)
		hard, warns := validateMutation(cfg, it, resolver, items, opts.Force)
		return it, hard, warns
	}

	var updated *datamodel.Item
	var warns []error
	finish := func(it *datamodel.Item) error {
		it, errs, w := assemble(it)
		if len(errs) > 0 {
			return errx.Invalid(errs)
		}
		updated, warns = it, w
		return nil
	}

	switch {
	case len(opts.Fields) > 0:
		it := cloneItem(orig)
		var errs []error
		for _, fe := range opts.Fields {
			value := fe.Value
			if fe.Key == datamodel.KeyOwner || fe.Key == datamodel.KeyReporter {
				if value, err = s.resolveMe(fe.Value); err != nil {
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
	subject := subjectPrefix + updated.Number + " edit " + strings.Join(changed, ",")
	if err := s.commitMutation(cfg, orig, updated, changed, warns, subject, datamodel.SourceCLI); err != nil {
		return nil, err
	}
	return &datamodel.MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed}, nil
}

func (s *Store) editorContent(cfg *datamodel.Config, ref string, opts EditOpts) (string, string, error) {
	orig, _, resolver, err := s.resolveRef(cfg, ref)
	if err != nil {
		return "", "", err
	}
	if err := guardWritable(orig); err != nil {
		return "", "", err
	}
	content, err := runEditor(codec.Serialize(orig), validateBuffer(cfg, resolver, opts.Force, func(c string) (*datamodel.Item, []error) {
		it, errs := parseFullItem(c)
		if len(errs) > 0 {
			return nil, errs
		}
		restoreImmutable(it, orig)
		return it, nil
	}))
	return content, orig.Updated, err
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
	switch key {
	case datamodel.KeyTitle:
		it.Title = value
	case datamodel.KeyState:
		it.State = value
	case datamodel.KeySubtype:
		it.Subtype = ptrOrNil(value)
	case datamodel.KeyResolution:
		it.Resolution = ptrOrNil(value)
	case datamodel.KeyPriority:
		it.Priority = ptrOrNil(value)
	case datamodel.KeyRank:
		it.Rank = ptrOrNil(value)
	case datamodel.KeySprint:
		it.Sprint = ptrOrNil(value)
	case datamodel.KeyDue:
		it.Due = ptrOrNil(value)
	case datamodel.KeyOwner:
		it.Owner = ptrOrNil(value)
	case datamodel.KeyReporter:
		it.Reporter = ptrOrNil(value)
	case datamodel.KeyEpic:
		it.Epic = ptrOrNil(value)
	case datamodel.KeyEstimate:
		if value == "" {
			it.Estimate = nil
			return nil
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("--field estimate: invalid number %q", value)
		}
		it.Estimate = &f
	case datamodel.KeyLabels:
		it.Labels = splitList(value)
	default:
		return errx.User("--field: unknown or immutable field %q", key).WithHint("%s", fieldHint(key))
	}
	return nil
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func ptrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func cloneItem(src *datamodel.Item) *datamodel.Item {
	dst := *src
	dst.Aliases = slices.Clone(src.Aliases)
	dst.Labels = slices.Clone(src.Labels)
	dst.BlockedBy = slices.Clone(src.BlockedBy)
	if src.Links != nil {
		dst.Links = make(map[string][]string, len(src.Links))
		for typ, targets := range src.Links {
			dst.Links[typ] = slices.Clone(targets)
		}
	}
	return &dst
}
