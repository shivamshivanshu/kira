package datamodel

import (
	"slices"

	"gopkg.in/yaml.v3"
)

type Person struct {
	Name string   `yaml:"name"`
	Git  []string `yaml:"git,omitempty"`
}

func (p *Person) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		return n.Decode(&p.Name)
	}
	type bare Person
	return n.Decode((*bare)(p))
}

type People struct {
	Known  []Person `yaml:"known"`
	Strict bool     `yaml:"strict"`
}

func (pl People) Names() []string {
	out := make([]string, len(pl.Known))
	for i, p := range pl.Known {
		out[i] = p.Name
	}
	return out
}

func (pl People) Vocab() Vocab {
	return Vocab{Known: pl.Names(), Strict: pl.Strict}
}

func (pl People) Canonical(identities ...string) (string, bool) {
	for _, id := range identities {
		if id == "" {
			continue
		}
		for _, p := range pl.Known {
			if slices.Contains(p.Git, id) {
				return p.Name, true
			}
		}
	}
	return "", false
}

type EnumVocab struct {
	Values []string
	Strict *bool
}

func (v *EnumVocab) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.SequenceNode {
		return n.Decode(&v.Values)
	}
	var m struct {
		Values []string `yaml:"values"`
		Strict *bool    `yaml:"strict"`
	}
	if err := n.Decode(&m); err != nil {
		return err
	}
	v.Values, v.Strict = m.Values, m.Strict
	return nil
}

func (v EnumVocab) StrictOr(fallback bool) bool {
	if v.Strict != nil {
		return *v.Strict
	}
	return fallback
}
