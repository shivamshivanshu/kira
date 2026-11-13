package datamodel

type Workflow struct {
	States  []State
	Initial string
}

func (w Workflow) StateByKey(key string) (State, bool) {
	for _, st := range w.States {
		if st.Key == key {
			return st, true
		}
	}
	return State{}, false
}

func IsDropped(resolution *string) bool {
	return resolution != nil && *resolution == ResolutionDropped
}

func CategoryOf(typ, state string) (Category, bool) {
	wf, ok := WorkflowOf(typ)
	if !ok {
		return "", false
	}
	st, ok := wf.StateByKey(state)
	return st.Category, ok
}

type State struct {
	Key        string
	Category   Category
	Wip        int
	Resolution string
}
