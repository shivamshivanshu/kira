package core

import "github.com/shivamshivanshu/kira/internal/datamodel"

func stateIn(wf datamodel.Workflow, key string) (datamodel.State, bool) {
	for _, st := range wf.States {
		if st.Key == key {
			return st, true
		}
	}
	return datamodel.State{}, false
}

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

func categoryOf(cfg *datamodel.Config, typ, state string) (datamodel.Category, bool) {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return "", false
	}
	st, ok := stateIn(wf, state)
	return st.Category, ok
}

func isDoneState(cfg *datamodel.Config, typ, state string) bool {
	cat, ok := categoryOf(cfg, typ, state)
	return ok && cat == datamodel.CategoryDone
}
