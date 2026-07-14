package config

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func AppendSprint(data []byte, sp datamodel.Sprint) ([]byte, error) {
	entry, err := inlineSprintEntry(sp)
	if err != nil {
		return nil, err
	}
	out, err := appendToTopLevelList(data, "sprints", entry)
	if err != nil {
		return nil, err
	}
	cfg, err := Parse(out)
	if err != nil {
		return nil, err
	}
	if n := len(cfg.Sprints); n == 0 || cfg.Sprints[n-1] != sp {
		return nil, fmt.Errorf("config: sprints: appended entry did not round-trip")
	}
	return out, nil
}

func inlineSprintEntry(sp datamodel.Sprint) (string, error) {
	fields := [4]string{}
	for i, v := range []string{sp.Key, sp.Name, sp.Start, sp.End} {
		s, err := flowScalar("sprints", v)
		if err != nil {
			return "", err
		}
		fields[i] = s
	}
	return fmt.Sprintf("{ key: %s, name: %s, start: %s, end: %s }",
		fields[0], fields[1], fields[2], fields[3]), nil
}
