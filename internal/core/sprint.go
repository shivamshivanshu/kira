package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

const activeSprintFile = "active-sprint"

func (s *Store) ActiveSprintKey() string {
	b, err := os.ReadFile(filepath.Join(s.cacheDir(), activeSprintFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

type SprintJSON struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Start string `json:"start"`
	End   string `json:"end"`
}

func sprintJSON(sp config.Sprint) SprintJSON {
	return SprintJSON{Key: sp.Key, Name: sp.Name, Start: sp.Start, End: sp.End}
}

func (s *Store) ResolveSprintKey(cfg *config.Config, key string) (string, error) {
	if key == "active" {
		active := s.ActiveSprintKey()
		if active == "" {
			return "", userErr("no active sprint is set (run `kira sprint activate <key>`)")
		}
		key = active
	}
	if !cfg.HasSprint(key) {
		return "", userErr("%q is not a key in the configured sprints", key)
	}
	return key, nil
}

func inSprint(it *item.Item, key string) bool {
	return it.Sprint != nil && *it.Sprint == key
}

type SprintCreateResult struct {
	Created bool       `json:"created"`
	Sprint  SprintJSON `json:"sprint"`
}

func (s *Store) SprintCreate(cfg *config.Config, sp config.Sprint) (*SprintCreateResult, error) {
	release, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer release()

	data, err := os.ReadFile(s.configPath())
	if err != nil {
		return nil, userErr("reading config: %v", err)
	}
	out, err := config.AppendSprint(data, sp)
	if err != nil {
		return nil, userErr("%v", err)
	}
	if err := os.WriteFile(s.configPath(), out, 0o644); err != nil {
		return nil, userErr("writing config: %v", err)
	}
	subject := "kira: sprint create " + sp.Key
	if err := s.finalize(cfg.Commit.Mode, cfg.Commit.Trailer, subject, "", s.relToRoot(s.configPath())); err != nil {
		return nil, err
	}
	return &SprintCreateResult{Created: true, Sprint: sprintJSON(sp)}, nil
}

type SprintItemCounts struct {
	Total int `json:"total"`
	Done  int `json:"done"`
}

type SprintListRow struct {
	SprintJSON
	Active bool             `json:"active"`
	Items  SprintItemCounts `json:"items"`
}

type SprintListResult struct {
	Sprints []SprintListRow `json:"sprints"`
}

func (s *Store) SprintList(cfg *config.Config) (*SprintListResult, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	active := s.ActiveSprintKey()
	rows := make([]SprintListRow, 0, len(cfg.Sprints))
	for _, sp := range cfg.Sprints {
		counts := SprintItemCounts{}
		for _, it := range items {
			if !inSprint(it, sp.Key) {
				continue
			}
			counts.Total++
			if isDoneState(cfg, it.Type, it.State) {
				counts.Done++
			}
		}
		rows = append(rows, SprintListRow{SprintJSON: sprintJSON(sp), Active: sp.Key == active, Items: counts})
	}
	return &SprintListResult{Sprints: rows}, nil
}

type SprintActivateResult struct {
	Activated string `json:"activated"`
	Previous  string `json:"previous,omitempty"`
}

func (s *Store) SprintActivate(cfg *config.Config, key string) (*SprintActivateResult, error) {
	if !cfg.HasSprint(key) {
		return nil, userErr("%q is not a key in the configured sprints", key)
	}
	prev := s.ActiveSprintKey()
	if err := os.MkdirAll(s.cacheDir(), 0o755); err != nil {
		return nil, userErr("creating cache dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.cacheDir(), activeSprintFile), []byte(key+"\n"), 0o644); err != nil {
		return nil, userErr("writing active-sprint pointer: %v", err)
	}
	return &SprintActivateResult{Activated: key, Previous: prev}, nil
}

type SprintCloseResult struct {
	Closed     string   `json:"closed"`
	WasActive  bool     `json:"was_active"`
	Unfinished []string `json:"unfinished"`
	MovedTo    string   `json:"moved_to,omitempty"`
}

func (s *Store) SprintClose(cfg *config.Config, key, moveTo string) (*SprintCloseResult, error) {
	if !cfg.HasSprint(key) {
		return nil, userErr("%q is not a key in the configured sprints", key)
	}
	if moveTo != "" {
		if moveTo == key {
			return nil, userErr("--move-to target %q is the sprint being closed", moveTo)
		}
		if !cfg.HasSprint(moveTo) {
			return nil, userErr("--move-to: %q is not a key in the configured sprints", moveTo)
		}
	}
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	var unfinished []*item.Item
	res := &SprintCloseResult{Closed: key, Unfinished: []string{}}
	for _, it := range items {
		if inSprint(it, key) && !isDoneState(cfg, it.Type, it.State) {
			unfinished = append(unfinished, it)
			res.Unfinished = append(res.Unfinished, it.Number)
		}
	}

	if moveTo != "" {
		apply := func(u *item.Item, _ *id.Resolver, _ []*item.Item) (hard, warns []error) {
			u.Sprint = &moveTo
			return nil, nil
		}
		subjectOf := func(orig *item.Item) string {
			return fmt.Sprintf("kira: %s sprint %s -> %s", orig.Number, key, moveTo)
		}
		for _, it := range unfinished {
			if _, _, err := s.mutate(cfg, it.ID, false, apply, subjectOf); err != nil {
				return nil, err
			}
		}
		res.MovedTo = moveTo
	}

	if s.ActiveSprintKey() == key {
		res.WasActive = true
		if err := os.Remove(filepath.Join(s.cacheDir(), activeSprintFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, userErr("clearing active-sprint pointer: %v", err)
		}
	}
	return res, nil
}
