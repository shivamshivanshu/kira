package doctor

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

func Check(cfg *datamodel.Config, resolver *id.Resolver, it *datamodel.Item) []Finding {
	var out []Finding
	out = append(out, stateFindings(it)...)
	out = append(out, resolutionFindings(it)...)
	out = append(out, vocabFindings(cfg, it)...)
	out = append(out, scalarFindings(cfg, it)...)
	out = append(out, boardFindings(cfg, it)...)
	out = append(out, refFindings(resolver, it)...)
	return out
}

func boardFindings(cfg *datamodel.Config, it *datamodel.Item) []Finding {
	key := id.KeyOf(it.Number)
	if key == "" {
		return nil
	}
	for _, b := range cfg.EffectiveBoards() {
		if strings.EqualFold(b.Key, key) {
			return nil
		}
	}
	return []Finding{{
		Class:    ClassBoard,
		Severity: SeverityWarning,
		Field:    datamodel.KeyNumber,
		Message:  fmt.Sprintf("number prefix %q is not a configured board", key),
	}}
}

func stateFindings(it *datamodel.Item) []Finding {
	wf, _ := datamodel.WorkflowOf(it.Type)
	for _, st := range wf.States {
		if st.Key == it.State {
			return nil
		}
	}
	return []Finding{{Class: ClassState, Severity: SeverityError, Field: datamodel.KeyState,
		Message: fmt.Sprintf("%q is not a state in the %s workflow", it.State, it.Type)}}
}

func resolutionFindings(it *datamodel.Item) []Finding {
	wf, _ := datamodel.WorkflowOf(it.Type)
	for _, st := range wf.States {
		if st.Key != it.State {
			continue
		}
		switch {
		case st.Category == datamodel.CategoryDone && it.Resolution == nil:
			return []Finding{{Class: ClassState, Severity: SeverityWarning, Field: datamodel.KeyResolution,
				Message: fmt.Sprintf("done state %s has no resolution set", it.State)}}
		case st.Category != datamodel.CategoryDone && it.Resolution != nil:
			return []Finding{{Class: ClassState, Severity: SeverityWarning, Field: datamodel.KeyResolution,
				Message: fmt.Sprintf("resolution %q on non-done state %s; clear it with `kira edit --field resolution=`", *it.Resolution, it.State)}}
		}
		return nil
	}
	return nil
}

func vocabFindings(cfg *datamodel.Config, it *datamodel.Item) []Finding {
	violations := datamodel.VocabViolations(cfg, it)
	out := make([]Finding, len(violations))
	for i, v := range violations {
		out[i] = vocabFinding(v.Field, v.Value, vocabSeverity(v.Strict))
	}
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
		out = append(out, schemaErr(datamodel.KeyDue, fmt.Sprintf("invalid date %q, want YYYY-MM-DD", *it.Due)))
	}
	return out
}

func schemaErr(field, msg string) Finding {
	return Finding{Class: ClassSchema, Severity: SeverityError, Field: field, Message: msg}
}

// refFindings checks links, parent and blocked_by for dangling and
// self-references. Target-type correctness needs the whole item set and lives
// in RefTypeMismatches instead.
func refFindings(resolver *id.Resolver, it *datamodel.Item) []Finding {
	if resolver == nil {
		return nil
	}
	var out []Finding
	check := func(field, ref string) {
		if ref == "" {
			return
		}
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
	for _, typ := range slices.Sorted(maps.Keys(it.Links)) {
		for _, ref := range it.Links[typ] {
			check(fmt.Sprintf("%s.%s", datamodel.KeyLinks, typ), ref)
		}
	}
	if it.Parent != nil {
		check(datamodel.KeyParent, *it.Parent)
	}
	for _, ref := range it.BlockedBy {
		check(datamodel.KeyBlockedBy, ref)
	}
	return out
}

func vocabSeverity(strict bool) Severity {
	if strict {
		return SeverityError
	}
	return SeverityWarning
}
