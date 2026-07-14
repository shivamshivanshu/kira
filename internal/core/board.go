package core

import (
	"os"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

var reservedBoardKeys = map[string]bool{
	"create":    true,
	"list":      true,
	"rename":    true,
	"archive":   true,
	"unarchive": true,
	"move":      true,
}

func boardView(b datamodel.Board) datamodel.BoardView {
	return datamodel.BoardView{Key: b.Key, Name: b.Name, Description: b.Description, Default: b.Default, Archived: b.Archived}
}

type BoardOpts struct {
	Type   string
	Epic   string
	Owner  string
	Label  string
	Query  string
	Filter string
	At     string
}

func (s *Store) Board(cfg *datamodel.Config, opts BoardOpts) (*datamodel.BoardResult, error) {
	ld, err := s.read(cfg, loadOpts{at: opts.At, useIndex: true})
	if err != nil {
		return nil, err
	}
	cfg = ld.cfg
	typ := opts.Type
	if typ == "" {
		typ = datamodel.TypeTicket
	}
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return nil, errx.User("no workflow configured for type %q", typ)
	}

	base := ListOpts{Type: typ, Owner: opts.Owner, Label: opts.Label, Query: opts.Query, Filter: opts.Filter, At: opts.At}
	global, err := s.matchSorted(cfg, ld, base)
	if err != nil {
		return nil, err
	}
	shown := global
	if opts.Epic != "" {
		scoped := base
		scoped.Epic = opts.Epic
		if shown, err = s.matchSorted(cfg, ld, scoped); err != nil {
			return nil, err
		}
	}

	counts := make(map[string]int, len(global))
	for _, it := range global {
		counts[it.State]++
	}
	epicNumbers := epicNumberMap(ld.items)
	byState := map[string][]datamodel.ListItem{}
	for _, it := range shown {
		byState[it.State] = append(byState[it.State], listItemOf(cfg, it, epicNumbers))
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

type configEdit struct {
	data    []byte
	commit  datamodel.Commit
	subject string
}

func (s *Store) mutateConfig(edit func(data []byte, locked *datamodel.Config) (configEdit, error)) error {
	fs := s.fs()
	release, err := fs.Lock()
	if err != nil {
		return err
	}
	defer release()

	data, err := os.ReadFile(fs.ConfigPath())
	if err != nil {
		return errx.User("reading config: %v", err)
	}
	locked, err := config.Parse(data)
	if err != nil {
		return errx.User("%v", err)
	}
	e, err := edit(data, locked)
	if err != nil {
		return err
	}
	if err := os.WriteFile(fs.ConfigPath(), e.data, filePerm); err != nil {
		return errx.User("writing config: %v", err)
	}
	_, err = s.finalize(e.commit.Mode, commitSpec{trailerKey: e.commit.Trailer, subject: e.commit.SubjectPrefix + e.subject}, fs.RelToRoot(fs.ConfigPath()))
	return err
}

func adoptImplicitBoard(locked *datamodel.Config) (*datamodel.Board, error) {
	if !config.ValidBoardKey(locked.Project.Key) {
		return nil, errx.User("project.key %q is not a valid board key (must match %s); boards cannot be adopted until the project key conforms", locked.Project.Key, config.BoardKeyPattern)
	}
	name := locked.Project.Name
	if name == "" {
		name = locked.Project.Key
	}
	return &datamodel.Board{Key: locked.Project.Key, Name: name, Default: true}, nil
}

func (s *Store) BoardCreate(cfg *datamodel.Config, key, name, description string) (*datamodel.BoardCreateResult, error) {
	if !config.ValidBoardKey(key) {
		return nil, errx.User("board key %q must match %s", key, config.BoardKeyPattern)
	}
	if reservedBoardKeys[strings.ToLower(key)] {
		return nil, errx.User("board key %q is a reserved subcommand name", key)
	}
	if name == "" {
		name = key
	}
	board := datamodel.Board{Key: key, Name: name, Description: description}
	err := s.mutateConfig(func(data []byte, locked *datamodel.Config) (configEdit, error) {
		for _, b := range locked.Boards {
			if strings.EqualFold(b.Key, key) {
				return configEdit{}, errx.User("board %q already exists", b.Key)
			}
		}
		var implicit *datamodel.Board
		if locked.Boards == nil {
			if strings.EqualFold(key, locked.Project.Key) || locked.Project.Key == "" {
				board.Default = true
			} else {
				adopted, err := adoptImplicitBoard(locked)
				if err != nil {
					return configEdit{}, err
				}
				implicit = adopted
			}
		}
		out, err := config.AddBoard(data, board, implicit)
		if err != nil {
			return configEdit{}, errx.User("%v", err)
		}
		return configEdit{data: out, commit: locked.Commit, subject: "board create " + key}, nil
	})
	if err != nil {
		return nil, err
	}
	return &datamodel.BoardCreateResult{Created: true, Board: boardView(board)}, nil
}

func (s *Store) BoardRename(cfg *datamodel.Config, key, name string) (*datamodel.BoardUpdateResult, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errx.User("board name must not be empty")
	}
	return s.updateBoard(cfg, key, "rename", func(b datamodel.Board) datamodel.Board {
		b.Name = name
		return b
	})
}

func (s *Store) BoardArchive(cfg *datamodel.Config, key string) (*datamodel.BoardUpdateResult, error) {
	active := cfg.ActiveBoards()
	if len(active) == 1 && strings.EqualFold(active[0].Key, key) {
		return nil, errx.User("cannot archive %s: it is the last active board", key).
			WithHint("create or unarchive another board first")
	}
	return s.updateBoard(cfg, key, "archive", func(b datamodel.Board) datamodel.Board {
		b.Archived = true
		return b
	})
}

func (s *Store) BoardUnarchive(cfg *datamodel.Config, key string) (*datamodel.BoardUpdateResult, error) {
	return s.updateBoard(cfg, key, "unarchive", func(b datamodel.Board) datamodel.Board {
		b.Archived = false
		return b
	})
}

func (s *Store) updateBoard(cfg *datamodel.Config, key, verb string, mutate func(datamodel.Board) datamodel.Board) (*datamodel.BoardUpdateResult, error) {
	var view datamodel.BoardView
	err := s.mutateConfig(func(data []byte, locked *datamodel.Config) (configEdit, error) {
		target, ok := locked.BoardByKey(key)
		if !ok {
			return configEdit{}, errx.User("no such board %q", key).
				WithHint("boards: %s", strings.Join(activeBoardKeys(locked.ActiveBoards()), ", "))
		}
		if locked.Boards == nil {
			adopted, err := adoptImplicitBoard(locked)
			if err != nil {
				return configEdit{}, err
			}
			if data, err = config.AddBoard(data, *adopted, nil); err != nil {
				return configEdit{}, errx.User("%v", err)
			}
		}
		out, err := config.UpdateBoard(data, target.Key, mutate)
		if err != nil {
			return configEdit{}, errx.User("%v", err)
		}
		reread, err := config.Parse(out)
		if err != nil {
			return configEdit{}, errx.User("%v", err)
		}
		nb, _ := reread.BoardByKey(target.Key)
		view = boardView(nb)
		return configEdit{data: out, commit: locked.Commit, subject: "board " + verb + " " + target.Key}, nil
	})
	if err != nil {
		return nil, err
	}
	return &datamodel.BoardUpdateResult{Board: view}, nil
}

func (s *Store) BoardList(cfg *datamodel.Config) (*datamodel.BoardListResult, error) {
	boards := cfg.EffectiveBoards()
	views := make([]datamodel.BoardView, len(boards))
	for i, b := range boards {
		views[i] = boardView(b)
	}
	return &datamodel.BoardListResult{Boards: views}, nil
}

func AdjacentAllowed(cfg *datamodel.Config, typ, from, to string) bool {
	wf, ok := cfg.Workflows[typ]
	if !ok {
		return false
	}
	return !wf.EnforceTransitions || transitionAllowed(wf, from, to)
}
