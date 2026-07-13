package seed

import (
	"fmt"
	"time"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type Opts struct {
	Size int
	Seed int64
}

type Summary struct {
	Epics    int
	Tickets  int
	Comments int
}

func (s Summary) Items() int { return s.Epics + s.Tickets }

var seedEpoch = time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)

type materializer func(i int, sp Spec, parentNumber string) (number string, err error)

func Seed(root string, cfg *datamodel.Config, opts Opts) (Summary, error) {
	st := storage.New(root)
	existing, err := st.LoadAll()
	if err != nil {
		return Summary{}, err
	}
	baseN := id.Allocate(storage.Snapshot(cfg.Project.Key, existing)).N

	specs := Recipe(opts.Size, opts.Seed)
	var sum Summary
	if err := walk(specs, rawSink(st, cfg, baseN, &sum)); err != nil {
		return sum, err
	}

	repo := gitx.Repo{Dir: root}
	if err := repo.Stage(".kira"); err != nil {
		return sum, err
	}
	if err := repo.Commit(fmt.Sprintf("kira: seed %d fixture items", len(specs)), "", ""); err != nil {
		return sum, err
	}
	return sum, nil
}

func walk(specs []Spec, m materializer) error {
	numbers := make([]string, len(specs))
	for i, sp := range specs {
		parent := ""
		if sp.Parent >= 0 {
			parent = numbers[sp.Parent]
		}
		n, err := m(i, sp, parent)
		if err != nil {
			return err
		}
		numbers[i] = n
	}
	return nil
}

func rawSink(st *storage.Store, cfg *datamodel.Config, baseN int, sum *Summary) materializer {
	hashStyle := cfg.ID.Style == datamodel.IDStyleHash
	return func(i int, sp Spec, parent string) (string, error) {
		u := id.Mint()
		number := id.AllocFor(hashStyle, cfg.Project.Key, baseN+i, u)
		ts := seedEpoch.Add(time.Duration(i) * time.Hour)
		it := buildItem(cfg, sp, u.String(), number, ts)
		if parent != "" {
			it.Epic = ptr(parent)
		}
		content := codec.Serialize(it)
		for j, body := range sp.Comments {
			content = codec.AppendComment(content, datamodel.Comment{
				ID:     id.Mint().String(),
				Author: sp.Owner,
				Ts:     ts.Add(time.Duration(j+1) * time.Minute).Format(time.RFC3339),
				Body:   body,
			})
			sum.Comments++
		}
		if _, err := st.WriteItemRaw(it.ID, content); err != nil {
			return "", err
		}
		if sp.Type == datamodel.TypeEpic {
			sum.Epics++
		} else {
			sum.Tickets++
		}
		return number, nil
	}
}

func buildItem(cfg *datamodel.Config, sp Spec, ulid, number string, ts time.Time) *datamodel.Item {
	state := resolveState(cfg, sp.Type, sp.Category)
	stamp := ts.Format(time.RFC3339)
	it := &datamodel.Item{
		ID:        ulid,
		Number:    number,
		Aliases:   []string{},
		Type:      sp.Type,
		Title:     sp.Title,
		State:     state,
		Labels:    append([]string{}, sp.Labels...),
		BlockedBy: []string{},
		Created:   stamp,
		Updated:   stamp,
		Body:      sp.Body,
	}
	if sp.Subtype != "" {
		it.Subtype = ptr(sp.Subtype)
	}
	if sp.Priority != "" {
		it.Priority = ptr(sp.Priority)
	}
	if sp.Owner != "" {
		it.Owner = ptr(sp.Owner)
	}
	if r := doneResolution(cfg, sp.Type, state); r != "" {
		it.Resolution = ptr(r)
	}
	return it
}

func resolveState(cfg *datamodel.Config, typ string, cat datamodel.Category) string {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return ""
	}
	for _, s := range wf.States {
		if s.Category == cat {
			return s.Key
		}
	}
	return wf.Initial
}

func doneResolution(cfg *datamodel.Config, typ, state string) string {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return ""
	}
	for _, s := range wf.States {
		if s.Key == state && s.Category == datamodel.CategoryDone {
			return s.Resolution
		}
	}
	return ""
}

func ptr[T any](v T) *T { return &v }
