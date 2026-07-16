package core

import (
	"errors"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

func fieldHint(key string) string {
	if n := errx.Nearest(key, datamodel.EditableFields); n != "" {
		return "did you mean `--field " + n + "=...`?"
	}
	return "editable fields: " + strings.Join(datamodel.EditableFields, ", ")
}

func sprintKeys(cfg *datamodel.Config) []string {
	keys := make([]string, len(cfg.Sprints))
	for i, s := range cfg.Sprints {
		keys[i] = s.Key
	}
	return keys
}

func sprintHint(cfg *datamodel.Config, key string) string {
	keys := sprintKeys(cfg)
	if len(keys) == 0 {
		return "no sprints are configured; add one with `kira sprint create`"
	}
	if n := errx.Nearest(key, keys); n != "" {
		return "did you mean `" + n + "`? (configured: " + strings.Join(keys, ", ") + ")"
	}
	return "configured sprints: " + strings.Join(keys, ", ")
}

func transitionHint(wf datamodel.Workflow, from string) string {
	targets := allowedTargets(wf, from)
	if len(targets) == 0 {
		return "no transitions out of " + from + "; use `--force` to override"
	}
	return "allowed from " + from + ": " + strings.Join(targets, ", ") + " (or `--force` to override)"
}

func stateHint(wf datamodel.Workflow, state string) string {
	keys := stateKeys(wf)
	if n := errx.Nearest(state, keys); n != "" {
		return "did you mean `" + n + "`? (states: " + strings.Join(keys, ", ") + ")"
	}
	return "valid states: " + strings.Join(keys, ", ")
}

func resolveID(r *id.Resolver, ref string) (string, error) {
	ulid, err := r.Resolve(ref)
	if err == nil {
		return ulid, nil
	}
	var nf *id.NotFoundError
	if errors.As(err, &nf) {
		e := errx.User("%v", nf)
		if nf.Suggestion != "" {
			return "", e.WithHint("did you mean `%s`?", nf.Suggestion)
		}
		return "", e.WithHint("run `kira list` to see valid ids")
	}
	return "", errx.User("%v", err)
}
