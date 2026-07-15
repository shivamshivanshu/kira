package core

import (
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

func (s *Store) LabelCreate(cfg *datamodel.Config, names []string) (*datamodel.LabelCreateResult, error) {
	for _, n := range names {
		if strings.TrimSpace(n) == "" {
			return nil, errx.User("label name cannot be empty")
		}
	}
	res := &datamodel.LabelCreateResult{Created: []string{}, AlreadyKnown: []string{}}

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
	current, err := config.Parse(data)
	if err != nil {
		return nil, errx.User("%v", err)
	}
	known := make(map[string]bool, len(current.Labels.Known))
	for _, n := range current.Labels.Known {
		known[n] = true
	}
	var toAdd []string
	for _, n := range names {
		if known[n] {
			res.AlreadyKnown = append(res.AlreadyKnown, n)
			continue
		}
		known[n] = true
		toAdd = append(toAdd, n)
	}
	if len(toAdd) == 0 {
		return res, nil
	}

	out, err := config.AppendKnownLabels(data, toAdd)
	if err != nil {
		return nil, errx.User("%v", err)
	}
	if err := os.WriteFile(fs.ConfigPath(), out, filePerm); err != nil {
		return nil, errx.Env("writing config: %v", err)
	}
	subject := cfg.Commit.SubjectPrefix + "label create " + strings.Join(toAdd, ",")
	if _, err := s.finalize(cfg.Commit.Mode, commitSpec{trailerKey: cfg.Commit.Trailer, subject: subject}, fs.RelToRoot(fs.ConfigPath())); err != nil {
		return nil, err
	}
	res.Created = toAdd
	return res, nil
}

func (s *Store) LabelList(cfg *datamodel.Config) (*datamodel.LabelListResult, error) {
	ld, err := s.load(cfg)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(cfg.Labels.Known))
	for _, n := range cfg.Labels.Known {
		counts[n] = 0
	}
	for _, it := range ld.items {
		seen := make(map[string]bool, len(it.Labels))
		for _, l := range it.Labels {
			if seen[l] {
				continue
			}
			seen[l] = true
			counts[l]++
		}
	}
	rows := make([]datamodel.LabelCount, 0, len(counts))
	for _, name := range slices.Sorted(maps.Keys(counts)) {
		rows = append(rows, datamodel.LabelCount{Name: name, Count: counts[name]})
	}
	return &datamodel.LabelListResult{Labels: rows}, nil
}

func (s *Store) LabelSet(cfg *datamodel.Config, ref, label string, add, force bool) (*datamodel.MutationResult, error) {
	apply := func(it *datamodel.Item, _ *id.Resolver, _ []*datamodel.Item) (hard, warns []error) {
		if add {
			if !slices.Contains(it.Labels, label) {
				it.Labels = append(it.Labels, label)
			}
			return nil, nil
		}
		it.Labels = slices.DeleteFunc(it.Labels, func(l string) bool { return l == label })
		return nil, nil
	}
	verb := "add"
	if !add {
		verb = "rm"
	}
	subjectOf := func(orig *datamodel.Item) string {
		return cfg.Commit.SubjectPrefix + orig.Number + " label " + verb + " " + label
	}
	updated, changed, err := s.mutate(cfg, ref, force, apply, subjectOf, datamodel.SourceCLI)
	if err != nil {
		return nil, err
	}
	return &datamodel.MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed}, nil
}
