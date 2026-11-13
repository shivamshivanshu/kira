package datamodel

import (
	"slices"

	"gopkg.in/yaml.v3"
)

// CapturedLabel marks items created via `kira create --here`. It is a system
// label kira manages itself, not part of the user-configured label vocabulary.
const CapturedLabel = "captured"

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
