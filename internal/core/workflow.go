package core

import "github.com/shivamshivanshu/kira/internal/datamodel"

func matchedTransition(wf datamodel.Workflow, from, to string) *datamodel.Transition {
	for i, t := range wf.Transitions[from] {
		if t.To == to {
			return &wf.Transitions[from][i]
		}
	}
	return nil
}

func allowedTargets(wf datamodel.Workflow, from string) []string {
	ts := wf.Transitions[from]
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.To)
	}
	return out
}

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

func transitionAllowed(wf datamodel.Workflow, from, to string) bool {
	return matchedTransition(wf, from, to) != nil
}

func MoveTargets(cfg *datamodel.Config, typ, from string) []string {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return nil
	}
	if !wf.EnforceTransitions {
		return stateKeys(wf)
	}
	return allowedTargets(wf, from)
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
