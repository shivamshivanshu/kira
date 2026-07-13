package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

type BoardOpts struct {
	Type string
	Epic string
	At   string
}

func (s *Store) Board(cfg *datamodel.Config, opts BoardOpts) (*datamodel.BoardResult, error) {
	wfCfg := cfg
	if opts.At != "" {
		ld, err := s.read(cfg, loadOpts{at: opts.At})
		if err != nil {
			return nil, err
		}
		wfCfg = ld.cfg
	}
	typ := opts.Type
	if typ == "" {
		typ = datamodel.TypeTicket
	}
	wf, ok := wfCfg.Workflows[typ]
	if !ok {
		return nil, errx.User("no workflow configured for type %q", typ)
	}

	global, err := s.List(cfg, ListOpts{Type: typ, At: opts.At})
	if err != nil {
		return nil, err
	}
	shown := global
	if opts.Epic != "" {
		if shown, err = s.List(cfg, ListOpts{Type: typ, Epic: opts.Epic, At: opts.At}); err != nil {
			return nil, err
		}
	}

	counts := make(map[string]int, len(global.Items))
	for _, it := range global.Items {
		counts[it.State]++
	}
	byState := map[string][]datamodel.ListItem{}
	for _, it := range shown.Items {
		byState[it.State] = append(byState[it.State], it)
	}

	cols := make([]datamodel.BoardColumn, len(wf.States))
	for i, st := range wf.States {
		items := byState[st.Key]
		if items == nil {
			items = []datamodel.ListItem{}
		}
		cols[i] = datamodel.BoardColumn{
			State:    st.Key,
			Category: string(st.Category),
			Wip:      st.Wip,
			Count:    counts[st.Key],
			Items:    items,
		}
	}
	return &datamodel.BoardResult{Type: typ, Columns: cols}, nil
}

func AdjacentAllowed(cfg *datamodel.Config, typ, from, to string) bool {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return false
	}
	return !wf.EnforceTransitions || transitionAllowed(wf, from, to)
}
