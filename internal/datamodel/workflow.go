package datamodel

import "github.com/shivamshivanshu/kira/internal/ptr"

type Workflow struct {
	States  []State
	Initial string
	Custom  bool
}

func (w Workflow) StateByKey(key string) (State, bool) {
	for _, st := range w.States {
		if st.Key == key {
			return st, true
		}
	}
	return State{}, false
}

func (w Workflow) FirstInCategory(category Category) (string, bool) {
	for _, state := range w.States {
		if state.Category == category {
			return state.Key, true
		}
	}
	return "", false
}

func (w Workflow) CloseTarget() string {
	if w.Custom && len(w.States) > 0 {
		return w.States[len(w.States)-1].Key
	}
	target, _ := w.FirstInCategory(CategoryDone)
	return target
}

func (c *Config) ItemWorkflow(typ, subtype string) (Workflow, bool) {
	if states := c.configuredStates(typ, subtype); len(states) > 0 {
		return Workflow{States: states, Initial: states[0].Key, Custom: true}, true
	}
	return WorkflowOf(typ)
}

func (c *Config) configuredStates(typ, subtype string) []State {
	if c == nil || typ != TypeTicket {
		return nil
	}
	return c.TicketStates[subtype]
}

func (c *Config) ItemCategory(it *Item) (Category, bool) {
	workflow := Workflow{States: c.configuredStates(it.Type, ptr.Deref(it.Subtype))}
	if state, declared := workflow.StateByKey(it.State); declared {
		return state.Category, true
	}
	return CategoryOf(it.Type, it.State)
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
	Key        string   `yaml:"name"`
	Category   Category `yaml:"category"`
	Wip        int      `yaml:"-"`
	Resolution string   `yaml:"-"`
}
