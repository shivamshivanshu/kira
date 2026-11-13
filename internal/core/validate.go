package core

import (
	"fmt"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/ptr"
	"github.com/shivamshivanshu/kira/internal/setx"
)

func fieldPresent(it *datamodel.Item, field string) bool {
	d, ok := datamodel.Field(field)
	if !ok || d.Present == nil {
		return false
	}
	return d.Present(it)
}

// vocabViolationError keeps the message mutation output has always printed
// for an out-of-vocabulary value.
type vocabViolationError struct{ datamodel.VocabViolation }

func (e vocabViolationError) Error() string {
	return fmt.Sprintf("field %q: value %q is not in the %q vocabulary", e.Field, e.Value, e.Vocab)
}

// validationCtx bundles one mutation's validation inputs; byID is built once
// and shared across checks rather than rebuilt by each.
type validationCtx struct {
	cfg      *datamodel.Config
	resolver *id.Resolver
	byID     map[string]*datamodel.Item
	force    bool
}

func newValidationCtx(cfg *datamodel.Config, items []*datamodel.Item, resolver *id.Resolver, force bool) *validationCtx {
	return newValidationCtxByID(cfg, datamodel.IndexByID(items), resolver, force)
}

// newValidationCtxByID takes an already-built item index, for callers (a
// Batch) that build it once and reuse it across many mutations.
func newValidationCtxByID(cfg *datamodel.Config, byID map[string]*datamodel.Item, resolver *id.Resolver, force bool) *validationCtx {
	return &validationCtx{cfg: cfg, resolver: resolver, byID: byID, force: force}
}

func (c *validationCtx) validateItem(it *datamodel.Item) (errs, warns []error) {
	if it.Title == "" {
		errs = append(errs, fmt.Errorf("field %q: required, missing", datamodel.KeyTitle))
	}
	for _, v := range datamodel.VocabViolations(c.cfg, it) {
		if v.Strict && !c.force {
			errs = append(errs, vocabViolationError{v})
		} else {
			warns = append(warns, vocabViolationError{v})
		}
	}
	if it.Due != nil && *it.Due != "" && !datamodel.ValidDate(*it.Due) {
		errs = append(errs, fmt.Errorf("field %q: expected type date, got string", datamodel.KeyDue))
	}

	if wf, ok := datamodel.WorkflowOf(it.Type); !ok {
		errs = append(errs, fmt.Errorf("field %q: no workflow configured for type %q", datamodel.KeyType, it.Type))
	} else if _, defined := wf.StateByKey(it.State); !defined {
		errs = append(errs, fmt.Errorf("field %q: %q is not a state in the %s workflow", datamodel.KeyState, it.State, it.Type))
	}

	if it.Rank != nil && *it.Rank == "" {
		errs = append(errs, fmt.Errorf("field %q: must be a non-empty string when present", datamodel.KeyRank))
	}
	if it.Sprint != nil && !c.cfg.HasSprint(*it.Sprint) {
		errs = append(errs, errx.User("field %q: %q is not a key in the configured sprints", datamodel.KeySprint, *it.Sprint).WithHint("%s", sprintHint(c.cfg, *it.Sprint)))
	}
	return errs, warns
}

func (c *validationCtx) validateAssembled(it *datamodel.Item) (hard, warns []error) {
	dedupSetFields(it)
	hard = normalizeAndCheckRefs(it, c.resolver)
	v, w := c.validateItem(it)
	return append(hard, v...), w
}

// dedupSetFields normalizes the set-valued fields to unique members,
// keeping first-occurrence order, so a stored or newly-entered duplicate is
// healed rather than rejected.
func dedupSetFields(it *datamodel.Item) {
	it.Labels = dedupStrings(it.Labels)
	it.BlockedBy = dedupStrings(it.BlockedBy)
	it.Aliases = dedupStrings(it.Aliases)
}

func dedupStrings(xs []string) []string {
	if len(xs) < 2 {
		return xs
	}
	seen := setx.NewDeduper[string]()
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if seen.Add(x) {
			out = append(out, x)
		}
	}
	return out
}

func (c *validationCtx) validateMutation(orig, it *datamodel.Item) (hard, warns []error) {
	hard, warns = c.validateAssembled(it)
	hard = append(hard, validateResolutionState(orig, it)...)
	if len(hard) == 0 {
		hard = validateGraph(it, c.byID)
	}
	return hard, warns
}

func validateResolutionState(orig, it *datamodel.Item) []error {
	if it.Resolution == nil {
		return nil
	}
	if orig != nil && it.State == orig.State && ptr.Equal(it.Resolution, orig.Resolution) {
		return nil
	}
	if cat, known := datamodel.CategoryOf(it.Type, it.State); known && cat != datamodel.CategoryDone {
		return []error{errx.User("field %q: only allowed on done-category states; %s is %s", datamodel.KeyResolution, it.State, cat).WithHint("clear it with `kira edit --field resolution=`")}
	}
	return nil
}

func (c *validationCtx) validateBuffer(build func(string) (*datamodel.Item, []error)) func(string) []error {
	return func(content string) []error {
		it, errs := build(content)
		if len(errs) > 0 {
			return errs
		}
		hard, _ := c.validateAssembled(it)
		return hard
	}
}

// validateGraph overlays updated onto byID for the duration of the check and
// restores the caller's entry afterwards, rather than cloning a map that grows
// with the repository on every mutation, then checks the parent's type and
// cycles.
func validateGraph(updated *datamodel.Item, byID map[string]*datamodel.Item) []error {
	prev, existed := byID[updated.ID]
	byID[updated.ID] = updated
	defer func() {
		if existed {
			byID[updated.ID] = prev
		} else {
			delete(byID, updated.ID)
		}
	}()

	var errs []error
	if updated.Parent != nil && *updated.Parent != "" {
		if parent := byID[*updated.Parent]; parent != nil && !datamodel.IsContainer(parent.Type) {
			errs = append(errs, fmt.Errorf("field %q: %s is type %s, which does not group other items",
				datamodel.KeyParent, numberOrID(parent, *updated.Parent), parent.Type))
		}
	}
	for _, rel := range datamodel.CycleRelations() {
		if cycle := datamodel.FindCycleFrom(byID, updated.ID, rel.Edges); len(cycle) > 0 {
			errs = append(errs, fmt.Errorf("field %q forms a cycle: %s", rel.Field, cycleTrail(cycle, byID)))
		}
	}
	return errs
}

func cycleTrail(cycle []string, byID map[string]*datamodel.Item) string {
	labels := make([]string, 0, len(cycle)+1)
	for _, u := range cycle {
		labels = append(labels, numberOrID(byID[u], u))
	}
	if len(cycle) > 0 {
		labels = append(labels, numberOrID(byID[cycle[0]], cycle[0]))
	}
	return strings.Join(labels, " -> ")
}

func numberOrID(it *datamodel.Item, ulid string) string {
	if it != nil && it.Number != "" {
		return it.Number
	}
	return ulid
}

func normalizeAndCheckRefs(it *datamodel.Item, resolver *id.Resolver) []error {
	var errs []error
	resolve := func(label, ref string) (string, bool) {
		ulid, err := resolver.Resolve(ref)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", label, err))
			return "", false
		}
		if ulid == it.ID {
			errs = append(errs, fmt.Errorf("%s: an item cannot link to itself", label))
			return "", false
		}
		return ulid, true
	}

	if it.Parent != nil {
		if ulid, ok := resolve(`field "parent"`, *it.Parent); ok {
			it.Parent = &ulid
		}
	}
	for i, b := range it.BlockedBy {
		if ulid, ok := resolve(`field "blocked_by"`, b); ok {
			it.BlockedBy[i] = ulid
		}
	}
	for typ, targets := range it.Links {
		for i, ref := range targets {
			if ulid, ok := resolve(fmt.Sprintf("field %q: %s", datamodel.KeyLinks, typ), ref); ok {
				targets[i] = ulid
			}
		}
	}
	return errs
}
