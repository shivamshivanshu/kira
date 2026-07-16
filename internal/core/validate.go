package core

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

type vocabWarning struct {
	field, value string
}

func (w *vocabWarning) Error() string {
	return fmt.Sprintf("field %q: %q is not in the known %s vocabulary", w.field, w.value, w.field)
}

func isSystemLabel(l string) bool { return l == capturedLabel }

func fieldPresent(it *datamodel.Item, field string) bool {
	d, ok := datamodel.Field(field)
	if !ok || d.Present == nil {
		return false
	}
	return d.Present(it)
}

func validateItem(cfg *datamodel.Config, it *datamodel.Item, force bool) (errs, warns []error) {
	if it.Title == "" {
		errs = append(errs, fmt.Errorf("field %q: required, missing", datamodel.KeyTitle))
	}
	if wf, ok := cfg.Workflows[it.Type]; !ok {
		errs = append(errs, fmt.Errorf("field %q: no workflow configured for type %q", datamodel.KeyType, it.Type))
	} else if _, defined := wf.StateByKey(it.State); !defined {
		errs = append(errs, fmt.Errorf("field %q: %q is not a state in the %s workflow", datamodel.KeyState, it.State, it.Type))
	}

	vocabCheck := func(field, value string, v datamodel.Vocab) {
		if value == "" || slices.Contains(v.Known, value) {
			return
		}
		e := &vocabWarning{field: field, value: value}
		if v.Strict && !force {
			errs = append(errs, e)
		} else {
			warns = append(warns, e)
		}
	}
	if it.Owner != nil {
		vocabCheck(datamodel.KeyOwner, *it.Owner, cfg.People.Vocab())
	}
	if it.Reporter != nil {
		vocabCheck(datamodel.KeyReporter, *it.Reporter, cfg.People.Vocab())
	}
	for _, l := range it.Labels {
		if isSystemLabel(l) {
			continue
		}
		vocabCheck(datamodel.KeyLabels, l, cfg.Labels)
	}

	enumCheck := func(field string, value *string, ev datamodel.EnumVocab) {
		if value != nil && len(ev.Values) > 0 {
			vocabCheck(field, *value, datamodel.Vocab{Known: ev.Values, Strict: ev.StrictOr(cfg.Labels.Strict)})
		}
	}
	enumCheck(datamodel.KeyPriority, it.Priority, cfg.Priorities)
	enumCheck(datamodel.KeySubtype, it.Subtype, cfg.Subtypes)
	enumCheck(datamodel.KeyResolution, it.Resolution, cfg.Resolutions)

	if it.Rank != nil && *it.Rank == "" {
		errs = append(errs, fmt.Errorf("field %q: must be a non-empty string when present", datamodel.KeyRank))
	}
	if it.Sprint != nil && !cfg.HasSprint(*it.Sprint) {
		errs = append(errs, errx.User("field %q: %q is not a key in the configured sprints", datamodel.KeySprint, *it.Sprint).WithHint("%s", sprintHint(cfg, *it.Sprint)))
	}
	if it.Due != nil && !datamodel.ValidDate(*it.Due) {
		errs = append(errs, fmt.Errorf("field %q: invalid RFC3339 date %q", datamodel.KeyDue, *it.Due))
	}
	return errs, warns
}

func validateAssembled(cfg *datamodel.Config, it *datamodel.Item, resolver *id.Resolver, force bool) (hard, warns []error) {
	hard = normalizeAndCheckRefs(it, resolver)
	v, w := validateItem(cfg, it, force)
	return append(hard, v...), w
}

func validateMutation(cfg *datamodel.Config, orig, it *datamodel.Item, resolver *id.Resolver, items []*datamodel.Item, force bool) (hard, warns []error) {
	hard, warns = validateAssembled(cfg, it, resolver, force)
	hard = append(hard, validateResolutionState(cfg, orig, it)...)
	if len(hard) == 0 {
		hard = validateGraph(it, items)
	}
	return hard, warns
}

func validateResolutionState(cfg *datamodel.Config, orig, it *datamodel.Item) []error {
	if it.Resolution == nil {
		return nil
	}
	if orig != nil && it.State == orig.State && ptr.Equal(it.Resolution, orig.Resolution) {
		return nil
	}
	if cat, known := cfg.CategoryOf(it.Type, it.State); known && cat != datamodel.CategoryDone {
		return []error{errx.User("field %q: only allowed on done-category states; %s is %s", datamodel.KeyResolution, it.State, cat).WithHint("clear it with `kira edit --field resolution=`")}
	}
	return nil
}

func validateBuffer(cfg *datamodel.Config, resolver *id.Resolver, force bool, build func(string) (*datamodel.Item, []error)) func(string) []error {
	return func(c string) []error {
		it, errs := build(c)
		if len(errs) > 0 {
			return errs
		}
		hard, _ := validateAssembled(cfg, it, resolver, force)
		return hard
	}
}

func validateGraph(updated *datamodel.Item, items []*datamodel.Item) []error {
	byID := make(map[string]*datamodel.Item, len(items)+1)
	for _, it := range items {
		if it.ID != "" {
			byID[it.ID] = it
		}
	}
	byID[updated.ID] = updated

	var errs []error
	if updated.Epic != nil && *updated.Epic != "" {
		if parent := byID[*updated.Epic]; parent != nil && parent.Type != datamodel.TypeEpic {
			errs = append(errs, fmt.Errorf("field %q: %s is not an epic", datamodel.KeyEpic, numberOrID(parent, *updated.Epic)))
		}
	}
	for _, rel := range datamodel.CycleRelations() {
		for _, cycle := range datamodel.FindCycles(byID, rel.Edges) {
			if slices.Contains(cycle, updated.ID) {
				errs = append(errs, fmt.Errorf("field %q forms a cycle: %s", rel.Field, cycleTrail(cycle, byID)))
				break
			}
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
			errs = append(errs, fmt.Errorf("%s: %v", label, err))
			return "", false
		}
		if ulid == it.ID {
			errs = append(errs, fmt.Errorf("%s: an item cannot link to itself", label))
			return "", false
		}
		return ulid, true
	}

	if it.Epic != nil {
		if ulid, ok := resolve(`field "epic"`, *it.Epic); ok {
			it.Epic = &ulid
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
