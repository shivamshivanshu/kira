package config

import (
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	entry, err := singleLineScalar("labels.known", name)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	labelsKey, labelsVal := findTopLevel(&doc, "labels")
	if labelsKey == nil || labelsVal.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config: labels.known: no labels section; add a `labels:` block with `known: []` to .kira/config.yaml")
	}
	knownKey, knownVal := mapEntry(labelsVal, "known")
	if knownKey == nil {
		return nil, fmt.Errorf("config: labels.known: missing; add `known: []` under `labels:` in .kira/config.yaml")
	}

	var out []string
	switch {
	case knownVal.Kind == yaml.SequenceNode && knownVal.Style&yaml.FlowStyle != 0:
		if out, err = appendToFlowList(lines, knownVal, entry); err != nil {
			return nil, err
		}
	case knownVal.Kind == yaml.SequenceNode && len(knownVal.Content) > 0:
		out = appendToBlockList(lines, knownVal, entry)
	default:
		return nil, fmt.Errorf("config: labels.known: initialize it as [] before registering labels")
	}

	res := strings.Join(out, "\n")
	cfg, err := Parse([]byte(res))
	if err != nil {
		return nil, err
	}
	if !slices.Contains(cfg.Labels.Known, name) {
		return nil, fmt.Errorf("config: labels.known: appended entry did not round-trip")
	}
	return []byte(res), nil
}

func appendToFlowList(lines []string, val *yaml.Node, entry string) ([]string, error) {
	i := val.Line - 1
	open := strings.IndexByte(lines[i], '[')
	closing := strings.LastIndexByte(lines[i], ']')
	if open < 0 || closing < open || maxLine(val) != val.Line {
		return nil, fmt.Errorf("config: labels.known: cannot append to a multi-line flow list; reformat it as a block list")
	}
	sep := ", "
	if strings.TrimSpace(lines[i][open+1:closing]) == "" {
		sep = ""
	}
	return replaceLine(lines, i, lines[i][:closing]+sep+entry+lines[i][closing:]), nil
}
