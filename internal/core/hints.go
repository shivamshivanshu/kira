package core

import (
	"errors"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

func fieldHint(key string, schema entityschema.Schema) string {
	candidates := editableFieldNames(schema)
	if n := errx.Nearest(key, candidates); n != "" {
		return "did you mean `--field " + n + "=...`?"
	}
	return "editable fields: " + strings.Join(candidates, ", ")
}

// editableFieldNames is the union of native --field keys and the schema's
// declared, settable, frontmatter-placed extra fields.
func editableFieldNames(schema entityschema.Schema) []string {
	out := slices.Clone(datamodel.EditableFields)
	for _, f := range schema.Fields {
		if !fieldEditableViaFlag(f) {
			continue
		}
		if _, ok := datamodel.Field(f.Name); ok {
			continue
		}
		out = append(out, f.Name)
	}
	slices.Sort(out)
	return out
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
	if nf, ok := errors.AsType[*id.NotFoundError](err); ok {
		e := errx.User("%v", nf)
		if nf.Suggestion != "" {
			return "", e.WithHint("did you mean `%s`?", nf.Suggestion)
		}
		return "", e.WithHint("run `kira list` to see valid ids")
	}
	return "", errx.User("%v", err)
}
