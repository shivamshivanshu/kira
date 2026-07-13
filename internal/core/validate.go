package core

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

func fieldPresent(it *datamodel.Item, field string) bool {
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

func validateItem(cfg *datamodel.Config, it *datamodel.Item, force bool) (errs, warns []error) {
	if it.Title == "" {
		errs = append(errs, fmt.Errorf("field %q: required, missing", "title"))
	}
	if !datamodel.ValidType(it.Type) {
		errs = append(errs, fmt.Errorf("field %q: must be %s or %s, got %q", "type", datamodel.TypeTicket, datamodel.TypeEpic, it.Type))
	}

	if wf, ok := cfg.Workflows[it.Type]; ok {
		if _, defined := stateIn(wf, it.State); !defined {
			errs = append(errs, fmt.Errorf("field %q: %q is not a state in the %s workflow", "state", it.State, it.Type))
		}
	}

	vocabCheck := func(field, value string, v datamodel.Vocab) {
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

	enumCheck := func(field string, value *string) {
		if known, _ := cfg.VocabFor(field); value != nil && len(known) > 0 {
			vocabCheck(field, *value, datamodel.Vocab{Known: known, Strict: cfg.Labels.Strict})
		}
	}
	enumCheck("priority", it.Priority)
	enumCheck("subtype", it.Subtype)
	enumCheck("resolution", it.Resolution)

	if it.Rank != nil && *it.Rank == "" {
		errs = append(errs, fmt.Errorf("field %q: must be a non-empty string when present", "rank"))
	}
	if it.Sprint != nil && !cfg.HasSprint(*it.Sprint) {
		errs = append(errs, errx.User("field %q: %q is not a key in the configured sprints", "sprint", *it.Sprint).WithHint("%s", sprintHint(cfg, *it.Sprint)))
	}
	if it.Due != nil && !datamodel.ValidDate(*it.Due) {
		errs = append(errs, fmt.Errorf("field %q: invalid RFC3339 date %q", "due", *it.Due))
	}
	return errs, warns
}

func validateAssembled(cfg *datamodel.Config, it *datamodel.Item, resolver *id.Resolver, force bool) (hard, warns []error) {
	hard = normalizeRefs(it, resolver)
	v, w := validateItem(cfg, it, force)
	return append(hard, v...), w
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

func normalizeRefs(it *datamodel.Item, resolver *id.Resolver) []error {
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
			if ulid, ok := resolve(fmt.Sprintf("field %q: %s", "links", typ), ref); ok {
				targets[i] = ulid
			}
		}
	}
	return errs
}
