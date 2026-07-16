package config

import (
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/errx"
)

// AppendKnownLabels appends label names to the known labels list in the config data.
func AppendKnownLabels(data []byte, names []string) ([]byte, error) {
	for _, name := range names {
		var err error
		if data, err = appendKnownLabel(data, name); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func appendKnownLabel(data []byte, name string) ([]byte, error) {
	entry, err := flowScalar("labels.known", name)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, errx.User("config: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	labelsKey, labelsVal := findTopLevel(&doc, "labels")
	if labelsKey == nil || labelsVal.Kind != yaml.MappingNode {
		return nil, errx.User("config: labels.known: no labels section; add a `labels:` block with `known: []` to .kira/config.yaml")
	}
	knownKey, knownVal := mapEntry(labelsVal, "known")
	if knownKey == nil {
		return nil, errx.User("config: labels.known: missing; add `known: []` under `labels:` in .kira/config.yaml")
	}

	var out []string
	switch {
	case knownVal.Kind == yaml.SequenceNode && knownVal.Style&yaml.FlowStyle != 0:
		if out, err = appendToFlowList(lines, "labels.known", knownVal, entry); err != nil {
			return nil, err
		}
	case knownVal.Kind == yaml.SequenceNode && len(knownVal.Content) > 0:
		out = appendToBlockList(lines, knownVal, entry)
	default:
		return nil, errx.User("config: labels.known: initialize it as [] before registering labels")
	}

	res := strings.Join(out, "\n")
	cfg, err := Parse([]byte(res))
	if err != nil {
		return nil, err
	}
	if !slices.Contains(cfg.Labels.Known, name) {
		return nil, errx.User("config: labels.known: appended entry did not round-trip")
	}
	return []byte(res), nil
}
