package core

import "github.com/shivamshivanshu/kira/internal/datamodel"

func firstStateInCategory(wf datamodel.Workflow, cat datamodel.Category) (string, bool) {
	for _, st := range wf.States {
		if st.Category == cat {
			return st.Key, true
		}
	}
	return "", false
}

func stateKeys(wf datamodel.Workflow) []string {
	out := make([]string, 0, len(wf.States))
	for _, s := range wf.States {
		out = append(out, s.Key)
	}
	return out
}

func MoveTargets(cfg *datamodel.Config, typ, _ string) []string {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return nil
	}
	return stateKeys(wf)
}

func isDoneState(cfg *datamodel.Config, typ, state string) bool {
	cat, ok := cfg.CategoryOf(typ, state)
	return ok && cat == datamodel.CategoryDone
}

func clearStaleResolution(cfg *datamodel.Config, it *datamodel.Item) {
	if !isDoneState(cfg, it.Type, it.State) {
		it.Resolution = nil
	}
}

func categoryString(cfg *datamodel.Config, typ, state string) string {
	c, _ := cfg.CategoryOf(typ, state)
	return string(c)
}
