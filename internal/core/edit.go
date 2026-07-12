package core

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/item"
)

// FieldEdit is one --field key=value assignment.
type FieldEdit struct {
	Key   string
	Value string
}

// EditOpts are the edit inputs (docs/design/04-cli.md edit). Exactly one input
// mode applies: Fields (flag-only), FromFile (round-trip), or neither (open
// $EDITOR on the current file).
type EditOpts struct {
	Fields   []FieldEdit
	FromFile string
	Force    bool
}

// Edit mutates the item ref refers to. It parses the new content (from flags,
// --from-file, or the $EDITOR validate-retry loop), restores the immutable
// system fields, normalizes cross-references, bumps updated, and commits only
// the fields that actually changed. A no-op edit neither writes nor commits.
func (s *Store) Edit(cfg *config.Config, ref string, opts EditOpts) (*MutationResult, error) {
	release, orig, resolver, err := s.lockAndResolve(cfg, ref)
	if err != nil {
		return nil, err
	}
	defer release()

	assemble := func(it *item.Item) (*item.Item, []error, []error) {
		restoreImmutable(it, orig)
		hard, warns := validateAssembled(cfg, it, resolver, opts.Force)
		return it, hard, warns
	}

	var updated *item.Item
	var warns []error
	// finish is the shared tail of the flag and from-file cases: assemble the
	// candidate, reject on any hard error, else record it as the result.
	finish := func(it *item.Item) error {
		it, errs, w := assemble(it)
		if len(errs) > 0 {
			return invalidErr(errs)
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
			return nil, invalidErr(errs)
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
			return nil, invalidErr(errs)
		}
		if err := finish(it); err != nil {
			return nil, err
		}
	default:
		content, err := runEditor(orig.Serialize(), func(c string) []error {
			it, errs := parseFullItem(c)
			if len(errs) > 0 {
				return errs // surface soft parse errors too, not just assembly errors
			}
			_, aerrs, _ := assemble(it)
			return aerrs
		})
		if err != nil {
			return nil, err
		}
		it, _ := parseFullItem(content)
		updated, _, warns = assemble(it)
	}

	changed := changedFields(orig, updated)
	subject := "kira: " + updated.Number + " edit " + strings.Join(changed, ",")
	if err := s.commitMutation(cfg, updated, changed, warns, subject); err != nil {
		return nil, err
	}
	return &MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed}, nil
}

// parseFullItem parses a complete item file, returning the item and any
// structural errors (from item.ParseError). A document too malformed to yield
// an item returns a nil item with the fatal error.
func parseFullItem(content string) (*item.Item, []error) {
	it, err := item.Parse(content)
	if it == nil {
		return nil, []error{err}
	}
	if err != nil {
		var pe *item.ParseError
		if errors.As(err, &pe) {
			return it, pe.Errs
		}
		return it, []error{err}
	}
	return it, nil
}

// restoreImmutable overwrites the system-managed fields on it with orig's, so an
// edit can never change identity, number, type, creation time, or aliases
// (docs/design/02-data-model.md §1 mutability column).
func restoreImmutable(it, orig *item.Item) {
	it.ID = orig.ID
	it.Number = orig.Number
	it.Type = orig.Type
	it.Created = orig.Created
	it.Aliases = orig.Aliases
}

func applyFieldEdit(it *item.Item, key, value string) error {
	switch key {
	case "title":
		it.Title = value
	case "state":
		it.State = value
	case "priority":
		it.Priority = ptrOrNil(value)
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
		return fmt.Errorf("--field: unknown or immutable field %q", key)
	}
	return nil
}

// splitList parses a comma-separated flag value into a trimmed, non-nil list.
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

func cloneItem(src *item.Item) *item.Item {
	dst := *src
	dst.Aliases = slices.Clone(src.Aliases)
	dst.Labels = slices.Clone(src.Labels)
	dst.BlockedBy = slices.Clone(src.BlockedBy)
	return &dst
}

// changedFields returns the mutable field names that differ between orig and
// updated, in canonical order, for the commit subject.
func changedFields(orig, updated *item.Item) []string {
	var changed []string
	add := func(cond bool, name string) {
		if cond {
			changed = append(changed, name)
		}
	}
	add(orig.Title != updated.Title, "title")
	add(orig.State != updated.State, "state")
	add(!equalPtr(orig.Priority, updated.Priority), "priority")
	add(!equalPtr(orig.Owner, updated.Owner), "owner")
	add(!equalPtr(orig.Reporter, updated.Reporter), "reporter")
	add(!slices.Equal(orig.Labels, updated.Labels), "labels")
	add(!equalPtr(orig.Epic, updated.Epic), "epic")
	add(!slices.Equal(orig.BlockedBy, updated.BlockedBy), "blocked_by")
	add(!equalPtr(orig.Estimate, updated.Estimate), "estimate")
	add(orig.Body != updated.Body, "body")
	return changed
}

// equalPtr reports whether two optional values are equal, treating nil (an
// absent field) as distinct from any set value.
func equalPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
