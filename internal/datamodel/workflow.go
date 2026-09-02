package datamodel

import "slices"

type Workflow struct {
	States      []State   `yaml:"states"`
	Initial     string    `yaml:"initial"`
	CloseTarget string    `yaml:"close_target,omitempty"`
	WipPolicy   WipPolicy `yaml:"wip_policy,omitempty"`
}

func (w Workflow) EffectiveWipPolicy() WipPolicy {
	if w.WipPolicy == WipBlock {
		return WipBlock
	}
	return WipWarn
}

func (w Workflow) StateByKey(key string) (State, bool) {
	for _, st := range w.States {
		if st.Key == key {
			return st, true
		}
	}
	return State{}, false
}

// IsDropped takes the dropped set rather than a Config because the renderer
// copies it in once and never carries config any further.
func IsDropped(resolution *string, dropped []string) bool {
	return resolution != nil && slices.Contains(dropped, *resolution)
}

func (c *Config) CategoryOf(typ, state string) (Category, bool) {
	wf, ok := c.Workflows[typ]
	if !ok {
		return "", false
	}
	st, ok := wf.StateByKey(state)
	return st.Category, ok
}

type State struct {
	Key        string   `yaml:"key"`
	Category   Category `yaml:"category"`
	Wip        int      `yaml:"wip,omitempty"`
	Resolution string   `yaml:"resolution,omitempty"`
}
