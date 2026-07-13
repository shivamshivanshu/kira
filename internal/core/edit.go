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
	if len(opts.Fields) == 0 && opts.FromFile == "" {
		content, err := s.editorContent(cfg, ref, opts)
		if err != nil {
			return nil, err
		}
		edited, _ = parseFullItem(content)
	}

	release, orig, _, resolver, err := s.lockAndResolve(cfg, ref)
	if err != nil {
		return nil, err
	}
	defer release()

	assemble := func(it *datamodel.Item) (*datamodel.Item, []error, []error) {
		restoreImmutable(it, orig)
		hard, warns := validateAssembled(cfg, it, resolver, opts.Force)
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
			if err := applyFieldEdit(it, fe.Key, fe.Value); err != nil {
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
	subject := "kira: " + updated.Number + " edit " + strings.Join(changed, ",")
	if err := s.commitMutation(cfg, updated, changed, warns, subject); err != nil {
		return nil, err
	}
	return &datamodel.MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed}, nil
}

func (s *Store) editorContent(cfg *datamodel.Config, ref string, opts EditOpts) (string, error) {
	orig, _, resolver, err := s.resolveRef(cfg, ref)
	if err != nil {
		return "", err
	}
	if err := guardWritable(orig); err != nil {
		return "", err
	}
	return runEditor(codec.Serialize(orig), func(c string) []error {
		it, errs := parseFullItem(c)
		if len(errs) > 0 {
			return errs
		}
		restoreImmutable(it, orig)
		hard, _ := validateAssembled(cfg, it, resolver, opts.Force)
		return hard
	})
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
	case "title":
		it.Title = value
	case "state":
		it.State = value
	case "subtype":
		it.Subtype = ptrOrNil(value)
	case "resolution":
		it.Resolution = ptrOrNil(value)
	case "priority":
		it.Priority = ptrOrNil(value)
	case "rank":
		it.Rank = ptrOrNil(value)
	case "sprint":
		it.Sprint = ptrOrNil(value)
	case "due":
		it.Due = ptrOrNil(value)
	case "owner":
		it.Owner = ptrOrNil(value)
	case "reporter":
		it.Reporter = ptrOrNil(value)
	case "epic":
		it.Epic = ptrOrNil(value)
	case "estimate":
		if value == "" {
			it.Estimate = nil
			return nil
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("--field estimate: invalid number %q", value)
		}
		it.Estimate = &f
	case "labels":
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
