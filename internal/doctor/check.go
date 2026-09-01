package doctor

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/id"
)

func Check(cfg *datamodel.Config, resolver *id.Resolver, it *datamodel.Item, schemas map[string]entityschema.Schema) []Finding {
	var out []Finding
	out = append(out, stateFindings(cfg, it)...)
	out = append(out, resolutionFindings(cfg, it)...)
	out = append(out, vocabFindings(cfg, it)...)
	out = append(out, scalarFindings(cfg, it)...)
	out = append(out, boardFindings(cfg, it)...)
	out = append(out, refFindings(resolver, it, schemas, itemTypes(cfg))...)
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

func resolutionFindings(cfg *datamodel.Config, it *datamodel.Item) []Finding {
	wf, ok := cfg.Workflows[it.Type]
	if !ok {
		return nil
	}
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
	var out []Finding
	for _, p := range []struct {
		field string
		value *string
	}{
		{datamodel.KeyOwner, it.Owner},
		{datamodel.KeyReporter, it.Reporter},
	} {
		if p.value != nil && *p.value != "" && !slices.ContainsFunc(cfg.People.Known, func(person datamodel.Person) bool { return person.Name == *p.value }) {
			out = append(out, vocabFinding(p.field, *p.value, vocabSeverity(cfg.People.Strict)))
		}
	}
	for _, l := range it.Labels {
		if l != datamodel.CapturedLabel && !slices.Contains(cfg.Labels.Known, l) {
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
		out = append(out, schemaErr(datamodel.KeyDue, fmt.Sprintf("invalid date %q, want YYYY-MM-DD", *it.Due)))
	}
	return out
}

// extrasFindings reports schema violations on declared frontmatter extras;
// builtin fields stay with the config-driven checks above.
func extrasFindings(cfg *datamodel.Config, schemas map[string]entityschema.Schema, it *datamodel.Item) []Finding {
	var out []Finding
	for _, v := range entityschema.ValidateExtras(schemas, cfg, it, codec.DecodeExtraScalar) {
		out = append(out, schemaErr(v.Field, v.Message))
	}
	return out
}

func schemaErr(field, msg string) Finding {
	return Finding{Class: ClassSchema, Severity: SeverityError, Field: field, Message: msg}
}

// refFindings walks every ref field the item's schema declares, so a custom
// ref gets the same dangling and self-reference checks as the built-in three.
// Target-type correctness needs the whole item set and lives in
// RefTypeMismatches instead.
func refFindings(resolver *id.Resolver, it *datamodel.Item, schemas map[string]entityschema.Schema, itemTypes map[string]bool) []Finding {
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
	schema, ok := schemas[it.Type]
	if !ok {
		return out
	}
	values := entityschema.ProjectItem(schema, it, codec.DecodeExtraScalar)
	for _, f := range schema.Fields {
		if !f.TargetsItemType(itemTypes) {
			continue
		}
		for _, ref := range refStrings(values[f.Name]) {
			check(f.Name, ref)
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

// itemTypes is the set of types config declares a workflow for, which is what
// makes a type exist at all.
func itemTypes(cfg *datamodel.Config) map[string]bool {
	out := make(map[string]bool, len(cfg.Workflows))
	for typ := range cfg.Workflows {
		out[typ] = true
	}
	return out
}
