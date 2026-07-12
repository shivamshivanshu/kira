package core

import (
	"fmt"
	"slices"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// stateInWorkflow reports whether key names a state in wf. It is the one home of
// the state-existence rule, shared by validateItem and the move adjacency check.
func stateInWorkflow(wf config.Workflow, key string) bool {
	return slices.ContainsFunc(wf.States, func(s config.State) bool { return s.Key == key })
}

// categoryOf returns the configured category of a state within an item type's
// workflow. Telemetry and filters key off category, never the state-name string
// (docs/design/02-data-model.md §6).
func categoryOf(cfg *config.Config, typ, state string) (config.Category, bool) {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return "", false
	}
	for _, st := range wf.States {
		if st.Key == state {
			return st.Category, true
		}
	}
	return "", false
}

// validateItem checks a fully-assembled item against config: type, state, epic
// reference, and controlled vocabularies. Vocabulary violations are rejected
// only when the list is strict and force is not set; a non-strict violation is
// returned as a warning instead. It returns hard errors (which block the write)
// and warnings (which are surfaced but do not block) separately.
func validateItem(cfg *config.Config, it *item.Item, force bool) (errs, warns []error) {
	if it.Title == "" {
		errs = append(errs, fmt.Errorf("field %q: required, missing", "title"))
	}
	if !item.ValidType(it.Type) {
		errs = append(errs, fmt.Errorf("field %q: must be %s or %s, got %q", "type", item.TypeTicket, item.TypeEpic, it.Type))
	}

	if wf, ok := cfg.Workflows[it.Type]; ok {
		if !stateInWorkflow(wf, it.State) {
			errs = append(errs, fmt.Errorf("field %q: %q is not a state in the %s workflow", "state", it.State, it.Type))
		}
	}

	vocabCheck := func(field, value string, v config.Vocab) {
		if value == "" || slices.Contains(v.Known, value) {
			return
		}
		e := fmt.Errorf("field %q: %q is not in the known %s vocabulary", field, value, field)
		if v.Strict && !force {
			errs = append(errs, e)
		} else {
			warns = append(warns, e)
		}
	}
	if it.Owner != nil {
		vocabCheck("owner", *it.Owner, cfg.People)
	}
	if it.Reporter != nil {
		vocabCheck("reporter", *it.Reporter, cfg.People)
	}
	for _, l := range it.Labels {
		vocabCheck("label", l, cfg.Labels)
	}
	return errs, warns
}

// validateAssembled is the shared tail of the create and edit assembly paths:
// normalize the item's cross-references to canonical ULIDs, then validate it
// against config. hard blocks the write; warns is surfaced but does not.
func validateAssembled(cfg *config.Config, it *item.Item, resolver *id.Resolver, force bool) (hard, warns []error) {
	hard = normalizeRefs(it, resolver)
	v, w := validateItem(cfg, it, force)
	return append(hard, v...), w
}

// normalizeRefs rewrites the epic and blocked_by cross-references to canonical
// ULIDs (a hand-typed KIRA-n is resolved to the underlying ULID), so
// cross-references never persist as display numbers
// (docs/design/02-data-model.md §7). An unresolvable reference is a hard error.
func normalizeRefs(it *item.Item, resolver *id.Resolver) []error {
	var errs []error
	if it.Epic != nil {
		if ulid, err := resolver.Resolve(*it.Epic); err == nil {
			it.Epic = &ulid
		} else {
			errs = append(errs, fmt.Errorf("field %q: %v", "epic", err))
		}
	}
	for i, b := range it.BlockedBy {
		if ulid, err := resolver.Resolve(b); err == nil {
			it.BlockedBy[i] = ulid
		} else {
			errs = append(errs, fmt.Errorf("field %q: %v", "blocked_by", err))
		}
	}
	return errs
}
