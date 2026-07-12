package core

import (
	"fmt"
	"slices"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// stateIn returns the definition of state key within wf — the one home of the
// state-existence rule, shared by validateItem and move (which reads the
// state's Category, Wip, and Resolution tag).
func stateIn(wf config.Workflow, key string) (config.State, bool) {
	for _, st := range wf.States {
		if st.Key == key {
			return st, true
		}
	}
	return config.State{}, false
}

// matchedTransition returns the configured from -> to edge, or nil when the
// move is off-graph — the one home of the adjacency rule over both transition
// forms (bare target and guard map). The returned edge carries the require:/set:
// guards move enforces (docs/design/02-data-model.md §6).
func matchedTransition(wf config.Workflow, from, to string) *config.Transition {
	for i, t := range wf.Transitions[from] {
		if t.To == to {
			return &wf.Transitions[from][i]
		}
	}
	return nil
}

// transitionAllowed reports whether to is reachable from from in one move.
func transitionAllowed(wf config.Workflow, from, to string) bool {
	return matchedTransition(wf, from, to) != nil
}

// categoryOf returns the configured category of a state within an item type's
// workflow. Telemetry and filters key off category, never the state-name string
// (docs/design/02-data-model.md §6).
func categoryOf(cfg *config.Config, typ, state string) (config.Category, bool) {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return "", false
	}
	st, ok := stateIn(wf, state)
	return st.Category, ok
}

// fieldPresent reports whether a require:-guarded frontmatter field is set on
// the candidate item: non-nil and non-empty for scalars, non-empty for lists
// (docs/design/02-data-model.md §6). The field name is one of
// item.MutableFields — config validation guarantees a guard can name nothing
// else — so the default arm is unreachable.
func fieldPresent(it *item.Item, field string) bool {
	set := func(p *string) bool { return p != nil && *p != "" }
	switch field {
	case "title":
		return it.Title != ""
	case "subtype":
		return set(it.Subtype)
	case "resolution":
		return set(it.Resolution)
	case "priority":
		return set(it.Priority)
	case "rank":
		return set(it.Rank)
	case "owner":
		return set(it.Owner)
	case "reporter":
		return set(it.Reporter)
	case "labels":
		return len(it.Labels) > 0
	case "epic":
		return set(it.Epic)
	case "sprint":
		return set(it.Sprint)
	case "due":
		return set(it.Due)
	case "estimate":
		return it.Estimate != nil
	default:
		return false
	}
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
		if _, defined := stateIn(wf, it.State); !defined {
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

	// The enum-ish scalars validate against their config list (VocabFor is the
	// one home of the field→list mapping) only when the list is non-empty
	// (empty = free-form), mirroring the labels strict/warn convention —
	// labels.strict governs (docs/design/02-data-model.md §10).
	enumCheck := func(field string, value *string) {
		if known, _ := cfg.VocabFor(field); value != nil && len(known) > 0 {
			vocabCheck(field, *value, config.Vocab{Known: known, Strict: cfg.Labels.Strict})
		}
	}
	enumCheck("priority", it.Priority)
	enumCheck("subtype", it.Subtype)
	enumCheck("resolution", it.Resolution)

	if it.Rank != nil && *it.Rank == "" {
		errs = append(errs, fmt.Errorf("field %q: must be a non-empty string when present", "rank"))
	}
	if it.Sprint != nil && !cfg.HasSprint(*it.Sprint) {
		errs = append(errs, fmt.Errorf("field %q: %q is not a key in the configured sprints", "sprint", *it.Sprint))
	}
	if it.Due != nil && !item.ValidDate(*it.Due) {
		errs = append(errs, fmt.Errorf("field %q: invalid RFC3339 date %q", "due", *it.Due))
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

// normalizeRefs rewrites the epic, blocked_by, and links cross-references to
// canonical ULIDs (a hand-typed KIRA-n is resolved to the underlying ULID), so
// cross-references never persist as display numbers
// (docs/design/02-data-model.md §7). An unresolvable reference is a hard error,
// as is any reference pointing back at the item itself — checked here, the one
// gate every write path (link, edit --from-file, $EDITOR) funnels through,
// rather than in kira link alone.
func normalizeRefs(it *item.Item, resolver *id.Resolver) []error {
	var errs []error
	// resolve maps one reference to its canonical ULID, rejecting a self-link;
	// label names the field (and link type) in errors.
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
			if ulid, ok := resolve(fmt.Sprintf("field %q: %s", "links", typ), ref); ok {
				targets[i] = ulid
			}
		}
	}
	return errs
}
