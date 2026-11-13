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

func MoveTargets(typ string) []string {
	wf, ok := datamodel.WorkflowOf(typ)
	if !ok {
		return nil
	}
	return stateKeys(wf)
}

func isDoneState(typ, state string) bool {
	cat, ok := datamodel.CategoryOf(typ, state)
	return ok && cat == datamodel.CategoryDone
}

func clearStaleResolution(it *datamodel.Item) {
	if !isDoneState(it.Type, it.State) {
		it.Resolution = nil
	}
}

func categoryString(typ, state string) string {
	c, _ := datamodel.CategoryOf(typ, state)
	return string(c)
}
