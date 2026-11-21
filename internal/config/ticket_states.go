package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/setx"
)

func validateTicketStates(workflows map[string][]datamodel.State) error {
	for _, subtype := range slices.Sorted(maps.Keys(workflows)) {
		if !slices.Contains(datamodel.Subtypes, subtype) {
			return errx.User("config: ticket_states.%s: unknown ticket subtype", subtype)
		}
		if err := validateTicketWorkflow(subtype, workflows[subtype]); err != nil {
			return err
		}
	}
	return nil
}

func validateTicketWorkflow(subtype string, states []datamodel.State) error {
	path := "ticket_states." + subtype
	if len(states) == 0 {
		return errx.User("config: %s: at least one state is required", path)
	}
	names := setx.NewDeduper[string]()
	for i, state := range states {
		where := fmt.Sprintf("%s[%d]", path, i)
		if state.Key == "" || state.Key != strings.TrimSpace(state.Key) || strings.ContainsAny(state.Key, "\r\n\x00") {
			return errx.User("config: %s.name: expected a non-empty single-line name without surrounding whitespace", where)
		}
		if !names.Add(state.Key) {
			return errx.User("config: %s.name: duplicate state %q", where, state.Key)
		}
		if err := validateEnum(where+".category", state.Category, datamodel.Categories...); err != nil {
			return err
		}
	}
	if states[len(states)-1].Category != datamodel.CategoryDone {
		return errx.User("config: %s: the last state must have category done", path)
	}
	return nil
}
