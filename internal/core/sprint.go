package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

const activeSprintFile = "active-sprint"

func (s *Store) ActiveSprintKey() string {
	b, err := os.ReadFile(filepath.Join(s.fs().CacheDir(), activeSprintFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func sprintView(sp datamodel.Sprint) datamodel.SprintView {
	return datamodel.SprintView{Key: sp.Key, Name: sp.Name, Start: sp.Start, End: sp.End}
}

func (s *Store) ResolveSprintKey(cfg *datamodel.Config, key string) (string, error) {
	if key == "active" {
		active := s.ActiveSprintKey()
		if active == "" {
			return "", errx.User("no active sprint is set (run `kira sprint activate <key>`)")
		}
		key = active
	}
	if !cfg.HasSprint(key) {
		return "", errx.User("%q is not a key in the configured sprints", key).WithHint("%s", sprintHint(cfg, key))
	}
	return key, nil
}

func inSprint(it *datamodel.Item, key string) bool {
	return it.Sprint != nil && *it.Sprint == key
}

func (s *Store) SprintCreate(cfg *datamodel.Config, sp datamodel.Sprint) (*datamodel.SprintCreateResult, error) {
	fs := s.fs()
	release, err := fs.Lock()
	if err != nil {
		return nil, err
	}
	defer release()

	data, err := os.ReadFile(fs.ConfigPath())
	if err != nil {
		return nil, errx.User("reading config: %v", err)
	}
	out, err := config.AppendSprint(data, sp)
	if err != nil {
		return nil, errx.User("%v", err)
	}
	if err := os.WriteFile(fs.ConfigPath(), out, 0o644); err != nil {
		return nil, errx.User("writing config: %v", err)
	}
	subject := subjectPrefix + "sprint create " + sp.Key
	if _, err := s.finalize(cfg.Commit.Mode, commitSpec{trailerKey: cfg.Commit.Trailer, subject: subject}, fs.RelToRoot(fs.ConfigPath())); err != nil {
		return nil, err
	}
	return &datamodel.SprintCreateResult{Created: true, Sprint: sprintView(sp)}, nil
}

func (s *Store) SprintList(cfg *datamodel.Config) (*datamodel.SprintListResult, error) {
	ld, err := s.read(cfg, loadOpts{})
	if err != nil {
		return nil, err
	}
	active := s.ActiveSprintKey()
	rows := make([]datamodel.SprintListRow, 0, len(cfg.Sprints))
	for _, sp := range cfg.Sprints {
		counts := datamodel.SprintItemCounts{}
		for _, it := range ld.items {
			if !inSprint(it, sp.Key) {
				continue
			}
			counts.Total++
			if isDoneState(cfg, it.Type, it.State) {
				counts.Done++
			}
		}
		rows = append(rows, datamodel.SprintListRow{SprintView: sprintView(sp), Active: sp.Key == active, Items: counts})
	}
	return &datamodel.SprintListResult{Sprints: rows}, nil
}

func (s *Store) SprintActivate(cfg *datamodel.Config, key string) (*datamodel.SprintActivateResult, error) {
	if !cfg.HasSprint(key) {
		return nil, errx.User("%q is not a key in the configured sprints", key)
	}
	prev := s.ActiveSprintKey()
	fs := s.fs()
	if err := os.MkdirAll(fs.CacheDir(), 0o755); err != nil {
		return nil, errx.User("creating cache dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fs.CacheDir(), activeSprintFile), []byte(key+"\n"), 0o644); err != nil {
		return nil, errx.User("writing active-sprint pointer: %v", err)
	}
	return &datamodel.SprintActivateResult{Activated: key, Previous: prev}, nil
}

func (s *Store) SprintClose(cfg *datamodel.Config, key, moveTo string) (*datamodel.SprintCloseResult, error) {
	if !cfg.HasSprint(key) {
		return nil, errx.User("%q is not a key in the configured sprints", key)
	}
	if moveTo != "" {
		if moveTo == key {
			return nil, errx.User("--move-to target %q is the sprint being closed", moveTo)
		}
		if !cfg.HasSprint(moveTo) {
			return nil, errx.User("--move-to: %q is not a key in the configured sprints", moveTo)
		}
	}
	ld, err := s.read(cfg, loadOpts{})
	if err != nil {
		return nil, err
	}
	var unfinished []*datamodel.Item
	res := &datamodel.SprintCloseResult{Closed: key, Unfinished: []string{}}
	for _, it := range ld.items {
		if inSprint(it, key) && !isDoneState(cfg, it.Type, it.State) {
			unfinished = append(unfinished, it)
			res.Unfinished = append(res.Unfinished, it.Number)
		}
	}

	if moveTo != "" {
		apply := func(u *datamodel.Item, _ *id.Resolver, _ []*datamodel.Item) (hard, warns []error) {
			u.Sprint = &moveTo
			return nil, nil
		}
		subjectOf := func(orig *datamodel.Item) string {
			return fmt.Sprintf(subjectPrefix+"%s sprint %s -> %s", orig.Number, key, moveTo)
		}
		for _, it := range unfinished {
			if _, _, err := s.mutate(cfg, it.ID, false, apply, subjectOf, datamodel.SourceCLI); err != nil {
				return nil, err
			}
		}
		res.MovedTo = moveTo
	}

	if s.ActiveSprintKey() == key {
		res.WasActive = true
		if err := os.Remove(filepath.Join(s.fs().CacheDir(), activeSprintFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, errx.User("clearing active-sprint pointer: %v", err)
		}
	}
	return res, nil
}
