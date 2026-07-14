package doctor

import (
	"fmt"
	"maps"
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

func Check(cfg *datamodel.Config, resolver *id.Resolver, it *datamodel.Item) []Finding {
	var out []Finding
	out = append(out, stateFindings(cfg, it)...)
	out = append(out, vocabFindings(cfg, it)...)
	out = append(out, scalarFindings(cfg, it)...)
	out = append(out, refFindings(resolver, it)...)
	return out
}

func stateFindings(cfg *datamodel.Config, it *datamodel.Item) []Finding {
	wf, ok := cfg.Workflows[it.Type]
	if !ok {
		return []Finding{schemaErr(datamodel.KeyType, fmt.Sprintf("no workflow configured for type %q", it.Type))}
	}
	for _, st := range wf.States {
		if st.Key == it.State {
			return nil
		}
	}
	return []Finding{{Class: ClassState, Severity: SeverityError, Field: datamodel.KeyState,
		Message: fmt.Sprintf("%q is not a state in the %s workflow", it.State, it.Type)}}
}

func vocabFindings(cfg *datamodel.Config, it *datamodel.Item) []Finding {
	var out []Finding
	for _, p := range []struct {
		field string
		value *string
	}{
		{datamodel.KeyOwner, it.Owner},
		{datamodel.KeyReporter, it.Reporter},
	} {
		if p.value != nil && *p.value != "" && !slices.Contains(cfg.People.Names(), *p.value) {
			out = append(out, vocabFinding(p.field, *p.value, vocabSeverity(cfg.People.Strict)))
		}
	}
	for _, l := range it.Labels {
		if !slices.Contains(cfg.Labels.Known, l) {
			out = append(out, vocabFinding(datamodel.KeyLabels, l, vocabSeverity(cfg.Labels.Strict)))
		}
	}
	enum := func(field string, value *string, ev datamodel.EnumVocab) {
		if value != nil && *value != "" && len(ev.Values) > 0 && !slices.Contains(ev.Values, *value) {
			out = append(out, vocabFinding(field, *value, vocabSeverity(ev.StrictOr(cfg.Labels.Strict))))
		}
	}
	enum(datamodel.KeyPriority, it.Priority, cfg.Priorities)
	enum(datamodel.KeySubtype, it.Subtype, cfg.Subtypes)
	enum(datamodel.KeyResolution, it.Resolution, cfg.Resolutions)
	return out
}

func vocabFinding(field, value string, sev Severity) Finding {
	return Finding{
		Class:    ClassVocab,
		Severity: sev,
		Field:    field,
		Message:  fmt.Sprintf("%q is not in the known %s vocabulary", value, field),
	}
}

func scalarFindings(cfg *datamodel.Config, it *datamodel.Item) []Finding {
	var out []Finding
	if it.Rank != nil && *it.Rank == "" {
		out = append(out, schemaErr(datamodel.KeyRank, "must be a non-empty string when present"))
	}
	if it.Sprint != nil && *it.Sprint != "" && !cfg.HasSprint(*it.Sprint) {
		out = append(out, schemaErr(datamodel.KeySprint, fmt.Sprintf("%q is not a key in the configured sprints", *it.Sprint)))
	}
	if it.Due != nil && *it.Due != "" && !datamodel.ValidDate(*it.Due) {
		out = append(out, schemaErr(datamodel.KeyDue, fmt.Sprintf("invalid RFC3339 date %q", *it.Due)))
	}
	return out
}

func schemaErr(field, msg string) Finding {
	return Finding{Class: ClassSchema, Severity: SeverityError, Field: field, Message: msg}
}

func refFindings(resolver *id.Resolver, it *datamodel.Item) []Finding {
	if resolver == nil {
		return nil
	}
	var out []Finding
	check := func(field, ref string) {
		ulid, err := resolver.Resolve(ref)
		if err != nil {
			out = append(out, Finding{Class: ClassRef, Severity: SeverityError, Field: field,
				Message: fmt.Sprintf("%s references %q, which resolves to no item", field, ref)})
			return
		}
		if ulid == it.ID {
			out = append(out, Finding{Class: ClassRef, Severity: SeverityError, Field: field,
				Message: field + " references the item itself"})
		}
	}
	if it.Epic != nil && *it.Epic != "" {
		check(datamodel.KeyEpic, *it.Epic)
	}
	for _, b := range it.BlockedBy {
		check(datamodel.KeyBlockedBy, b)
	}
	for _, typ := range slices.Sorted(maps.Keys(it.Links)) {
		for _, ref := range it.Links[typ] {
			check(fmt.Sprintf("%s.%s", datamodel.KeyLinks, typ), ref)
		}
	}
	return out
}

func vocabSeverity(strict bool) Severity {
	if strict {
		return SeverityError
	}
	return SeverityWarning
}
