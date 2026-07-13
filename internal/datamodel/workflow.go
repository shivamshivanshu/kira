package datamodel

import "gopkg.in/yaml.v3"

type Workflow struct {
	States             []State                 `yaml:"states"`
	Initial            string                  `yaml:"initial"`
	Transitions        map[string][]Transition `yaml:"transitions"`
	EnforceTransitions bool                    `yaml:"enforce_transitions"`
	CloseTarget        string                  `yaml:"close_target,omitempty"`
}

type State struct {
	Key        string   `yaml:"key"`
	Category   Category `yaml:"category"`
	Wip        int      `yaml:"wip,omitempty"`
	Resolution string   `yaml:"resolution,omitempty"`
}

type Transition struct {
	To      string            `yaml:"to"`
	Require []string          `yaml:"require,omitempty"`
	Set     map[string]string `yaml:"set,omitempty"`
}

func (t *Transition) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		t.To = n.Value
		return nil
	}
	type bareTransition Transition
	return n.Decode((*bareTransition)(t))
}

func TransitionsTo(states ...string) []Transition {
	ts := make([]Transition, len(states))
	for i, s := range states {
		ts[i] = Transition{To: s}
	}
	return ts
}
