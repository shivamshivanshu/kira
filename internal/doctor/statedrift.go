package doctor

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
)

// StateDrift reports schemas whose declared state values diverge from the
// configured workflow — two sources of truth until the transition graph is
// dropped, so the schema copy must not rot silently.
func StateDrift(cfg *datamodel.Config, schemas map[string]entityschema.Schema) []Finding {
	var out []Finding
	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		field, ok := stateFieldOf(schemas[name])
		if !ok {
			continue
		}
		wf, ok := cfg.Workflows[name]
		if !ok {
			continue
		}
		schemaStates := schemaStateKeys(field.States)
		wfStates := workflowStateKeys(wf.States)
		if sameStateSet(schemaStates, wfStates) {
			continue
		}
		out = append(out, Finding{
			Class:    ClassStateDrift,
			Severity: SeverityWarning,
			Field:    datamodel.KeyState,
			Message: fmt.Sprintf("schema %q declares states [%s], but the %s workflow has [%s]",
				name, strings.Join(schemaStates, ", "), name, strings.Join(wfStates, ", ")),
		})
	}
	return out
}

func stateFieldOf(s entityschema.Schema) (entityschema.FieldDef, bool) {
	for _, f := range s.Fields {
		if f.Type == entityschema.FieldState {
			return f, true
		}
	}
	return entityschema.FieldDef{}, false
}

func schemaStateKeys(states []entityschema.StateValue) []string {
	out := make([]string, len(states))
	for i, s := range states {
		out[i] = s.Key
	}
	return out
}

func workflowStateKeys(states []datamodel.State) []string {
	out := make([]string, len(states))
	for i, s := range states {
		out[i] = s.Key
	}
	return out
}

func sameStateSet(a, b []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(a)), slices.Sorted(slices.Values(b)))
}
