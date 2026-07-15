package config

import (
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func AddBoard(data []byte, b datamodel.Board, materializeImplicit *datamodel.Board) ([]byte, error) {
	original, err := Parse(data)
	if err != nil {
		return nil, err
	}
	expected := append([]datamodel.Board{}, original.Boards...)
	if materializeImplicit != nil {
		expected = append(expected, *materializeImplicit)
	}
	expected = append(expected, b)

	out := data
	if materializeImplicit != nil {
		if out, err = spliceBoard(out, *materializeImplicit); err != nil {
			return nil, err
		}
	}
	if out, err = spliceBoard(out, b); err != nil {
		return nil, err
	}
	if out, err = bumpVersionToBoards(out); err != nil {
		return nil, err
	}
	cfg, err := Parse(out)
	if err != nil {
		return nil, err
	}
	if !slices.Equal(cfg.Boards, expected) {
		return nil, errx.User("config: boards: entry did not round-trip cleanly (a value may need reformatting)")
	}
	return out, nil
}

func spliceBoard(data []byte, b datamodel.Board) ([]byte, error) {
	entry, err := inlineBoardEntry(b)
	if err != nil {
		return nil, err
	}
	return appendToTopLevelList(data, "boards", entry)
}

func inlineBoardEntry(b datamodel.Board) (string, error) {
	key, err := flowScalar("boards", b.Key)
	if err != nil {
		return "", err
	}
	name, err := flowScalar("boards", b.Name)
	if err != nil {
		return "", err
	}
	entry := fmt.Sprintf("{ key: %s, name: %s", key, name)
	if b.Description != "" {
		desc, err := flowScalar("boards", b.Description)
		if err != nil {
			return "", err
		}
		entry += ", description: " + desc
	}
	if b.Default {
		entry += ", default: true"
	}
	if b.Archived {
		entry += ", archived: true"
	}
	return entry + " }", nil
}

func UpdateBoard(data []byte, key string, mutate func(datamodel.Board) datamodel.Board) ([]byte, error) {
	cfg, err := Parse(data)
	if err != nil {
		return nil, err
	}
	current, ok := cfg.BoardByKey(key)
	if !ok {
		return nil, errx.User("config: boards: no board with key %q", key)
	}
	updated := mutate(current)
	entry, err := inlineBoardEntry(updated)
	if err != nil {
		return nil, err
	}
	expected := make([]datamodel.Board, len(cfg.Boards))
	for i, b := range cfg.Boards {
		if strings.EqualFold(b.Key, key) {
			expected[i] = updated
		} else {
			expected[i] = b
		}
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, errx.User("config: %w", err)
	}
	_, val := findTopLevel(&doc, "boards")
	if val == nil || val.Kind != yaml.SequenceNode {
		return nil, errx.User("config: boards: expected a list")
	}
	lines := strings.Split(string(data), "\n")
	for _, node := range val.Content {
		_, kv := mapEntry(node, "key")
		if kv == nil || !strings.EqualFold(kv.Value, key) {
			continue
		}
		if maxLine(node) != node.Line {
			return nil, errx.User("config: boards: cannot rewrite a multi-line entry for %q; reformat it inline", key)
		}
		i := node.Line - 1
		open := node.Column - 1
		closing := -1
		if open >= 0 && open < len(lines[i]) && lines[i][open] == '{' {
			closing = flowCloseIndex(lines[i], open)
		}
		if closing < 0 {
			return nil, errx.User("config: boards: malformed entry for %q", key)
		}
		out := replaceLine(lines, i, lines[i][:open]+entry+lines[i][closing+1:])
		res := []byte(strings.Join(out, "\n"))
		reread, err := Parse(res)
		if err != nil {
			return nil, err
		}
		if !slices.Equal(reread.Boards, expected) {
			return nil, errx.User("config: boards: update did not round-trip cleanly for %q", key)
		}
		return res, nil
	}
	return nil, errx.User("config: boards: no board with key %q", key)
}

func bumpVersionToBoards(data []byte) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, errx.User("config: %w", err)
	}
	_, val := findTopLevel(&doc, "version")
	lines := strings.Split(string(data), "\n")
	token := fmt.Sprint(datamodel.BoardsSchemaVersion)
	switch {
	case val == nil:
		lines = append([]string{"version: " + token}, lines...)
	case val.Value == fmt.Sprint(datamodel.InitialSchemaVersion):
		var err error
		if lines, err = replaceScalarLine(lines, val, token); err != nil {
			return nil, err
		}
	default:
		return data, nil
	}
	return []byte(strings.Join(lines, "\n")), nil
}
