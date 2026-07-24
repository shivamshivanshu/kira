package core

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
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

// enumStrict resolves a field's vocabulary strictness; a declared schema extra
// with no matching config vocab defaults to Labels.Strict.
func enumStrict(cfg *datamodel.Config, field string) bool {
	switch field {
	case datamodel.KeyOwner, datamodel.KeyReporter:
		return cfg.People.Strict
	case datamodel.KeyLabels:
		return cfg.Labels.Strict
	case datamodel.KeyPriority:
		return cfg.Priorities.StrictOr(cfg.Labels.Strict)
	case datamodel.KeySubtype:
		return cfg.Subtypes.StrictOr(cfg.Labels.Strict)
	case datamodel.KeyResolution:
		return cfg.Resolutions.StrictOr(cfg.Labels.Strict)
	default:
		return cfg.Labels.Strict
	}
}

// validationCtx bundles one item type's validation inputs; byID is built once
// and shared across checks rather than rebuilt by each.
type validationCtx struct {
	cfg      *datamodel.Config
	resolver *id.Resolver
	schema   entityschema.Schema
	enums    map[string][]string
	byID     map[string]*datamodel.Item
	force    bool
}

func newValidationCtx(cfg *datamodel.Config, items []*datamodel.Item, resolver *id.Resolver, schema entityschema.Schema, force bool) *validationCtx {
	enums := entityschema.ResolveEnums(schema, entityschema.ConfigVocab(cfg))
	return newValidationCtxWithEnums(cfg, items, resolver, schema, enums, force)
}

// newValidationCtxWithEnums takes an already-resolved enum vocab, for callers
// (a Batch) that resolve it once per type and reuse it across many mutations
// rather than recomputing it per item.
func newValidationCtxWithEnums(cfg *datamodel.Config, items []*datamodel.Item, resolver *id.Resolver, schema entityschema.Schema, enums map[string][]string, force bool) *validationCtx {
	return &validationCtx{cfg: cfg, resolver: resolver, schema: schema, enums: enums, byID: datamodel.IndexByID(items), force: force}
}

func (c *validationCtx) validateItem(it *datamodel.Item) (errs, warns []error) {
	values := entityschema.ProjectItem(c.schema, it, codec.DecodeExtraScalar)

	var refs entityschema.RefResolver
	if c.resolver != nil {
		refs = refResolver{resolver: c.resolver, byID: c.byID}
	}
	for _, viol := range entityschema.Validate(c.schema, values, c.enums, refs) {
		switch viol.Code {
		case entityschema.CodeEnum:
			if enumStrict(c.cfg, viol.Field) && !c.force {
				errs = append(errs, viol)
			} else {
				warns = append(warns, viol)
			}
		case entityschema.CodeRequired, entityschema.CodeType:
			errs = append(errs, viol)
		case entityschema.CodeRef:
			// epic/blocked_by existence and target-type are checked by
			// normalizeAndCheckRefs and validateGraph with clearer messages;
			// only declared ref extras need this generic check.
			if viol.Field != datamodel.KeyEpic && viol.Field != datamodel.KeyBlockedBy {
				errs = append(errs, viol)
			}
		}
		// CodeState is deliberately ignored: state membership is validated
		// below against cfg.Workflows, not the schema's static state list.
	}

	if wf, ok := c.cfg.Workflows[it.Type]; !ok {
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
	hard = normalizeAndCheckRefs(it, c.resolver, c.schema)
	v, w := c.validateItem(it)
	return append(hard, v...), w
}

// dedupSetFields normalizes the set-valued fields the schema marks unique,
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
	hard = append(hard, validateResolutionState(c.cfg, orig, it)...)
	if len(hard) == 0 {
		hard = validateGraph(it, c.byID, c.schema)
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

// validateGraph overlays updated onto a clone of byID so the caller's shared
// map is never mutated, then checks epic-is-epic, blocked_by target-type, and
// cycles.
func validateGraph(updated *datamodel.Item, byID map[string]*datamodel.Item, schema entityschema.Schema) []error {
	graph := maps.Clone(byID)
	graph[updated.ID] = updated

	var errs []error
	if updated.Epic != nil && *updated.Epic != "" {
		if parent := graph[*updated.Epic]; parent != nil && parent.Type != datamodel.TypeEpic {
			errs = append(errs, fmt.Errorf("field %q: %s is not an epic", datamodel.KeyEpic, numberOrID(parent, *updated.Epic)))
		}
	}
	if bf, ok := schema.Field(datamodel.KeyBlockedBy); ok && bf.Target != "" {
		for _, b := range updated.BlockedBy {
			if blocker := graph[b]; blocker != nil && blocker.Type != bf.Target {
				errs = append(errs, fmt.Errorf("field %q: %s is type %s, expected %s", datamodel.KeyBlockedBy, numberOrID(blocker, b), blocker.Type, bf.Target))
			}
		}
	}
	for _, rel := range datamodel.CycleRelations() {
		for _, cycle := range datamodel.FindCycles(graph, rel.Edges) {
			if slices.Contains(cycle, updated.ID) {
				errs = append(errs, fmt.Errorf("field %q forms a cycle: %s", rel.Field, cycleTrail(cycle, graph)))
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

func normalizeAndCheckRefs(it *datamodel.Item, resolver *id.Resolver, schema entityschema.Schema) []error {
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
	normalizeRefExtras(it, resolve, schema)
	return errs
}

// normalizeRefExtras binds a schema-declared entity-target ref extra (Source
// == "" distinguishes it from a config-backed ref like owner/reporter, which
// normalizeAndCheckRefs's config-driven fields already handle elsewhere) to
// its immutable ULID on write, the same as the built-in epic/blocked_by
// fields above, so it survives a renumber instead of going stale.
func normalizeRefExtras(it *datamodel.Item, resolve func(label, ref string) (string, bool), schema entityschema.Schema) {
	for _, f := range schema.Fields {
		if f.Type != entityschema.FieldRef || f.Source != "" {
			continue
		}
		label := fmt.Sprintf("field %q", f.Name)
		if f.List {
			targets, ok := it.ExtraLists[f.Name]
			if !ok {
				continue
			}
			for i, ref := range targets {
				if ulid, ok := resolve(label, ref); ok {
					targets[i] = ulid
				}
			}
			continue
		}
		if ref, ok := it.ExtraScalars[f.Name]; ok {
			if ulid, ok := resolve(label, ref); ok {
				it.ExtraScalars[f.Name] = ulid
			}
		}
	}
}
