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

func transitionAllowed(wf datamodel.Workflow, from, to string) bool {
	return matchedTransition(wf, from, to) != nil
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
