package core

import (
	"os"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/query"
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
	return datamodel.BoardView(b)
}

// BoardOpts filters and scopes a Board query.
type BoardOpts struct {
	Type     string
	Parent   string
	State    string
	Owner    string
	Priority string
	Label    string
	Board    string
	Sprint   string
	Query    string
	Filter   string
	At       string
}

// Board returns the workflow columns for opts.Type, populated with the
// matching items (optionally scoped to opts.Parent) grouped by state.
func (s *Store) Board(cfg *datamodel.Config, opts BoardOpts) (*datamodel.BoardResult, error) {
	ld, err := s.read(cfg, loadOpts{at: opts.At, useIndex: true})
	if err != nil {
		return nil, err
	}
	return boardResult(ld, opts, s.queryOptions(ld))
}

func boardResult(ld *loaded, opts BoardOpts, qopts query.Options) (*datamodel.BoardResult, error) {
	typ := opts.Type
	if typ == "" {
		typ = datamodel.TypeTicket
	}
	wf, ok := datamodel.WorkflowOf(typ)
	if !ok {
		return nil, UnknownTypeError(typ)
	}

	base := ListOpts{
		Type: typ, State: opts.State, Owner: opts.Owner, Priority: opts.Priority,
		Label: opts.Label, Board: opts.Board, Sprint: opts.Sprint,
		Query: opts.Query, Filter: opts.Filter, At: opts.At,
	}
	global, err := matchSorted(ld, base, qopts)
	if err != nil {
		return nil, err
	}
	shown := global
	if opts.Parent != "" {
		scoped := base
		scoped.Parent = opts.Parent
		if shown, err = matchSorted(ld, scoped, qopts); err != nil {
			return nil, err
		}
	}

	counts := make(map[string]int, len(global))
	for _, it := range global {
		counts[it.State]++
	}
	parentNumbers := parentNumberMap(ld.items)
	byState := map[string][]datamodel.ListItem{}
	for _, it := range shown {
		byState[it.State] = append(byState[it.State], listItemOf(it, parentNumbers))
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
	noop    bool
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
	if e.noop {
		return nil
	}
	if err := os.WriteFile(fs.ConfigPath(), e.data, filePerm); err != nil {
		return errx.Env("writing config: %v", err)
	}
	_, err = s.finalize(e.commit.Mode, commitSpec{subject: e.subject}, fs.RelToRoot(fs.ConfigPath()))
	return err
}

// BoardCreate adds a new board to the config.
func (s *Store) BoardCreate(_ *datamodel.Config, key, name, description string) (*datamodel.BoardCreateResult, error) {
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
		out, err := config.AddBoard(data, board)
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
	return s.updateBoard(cfg, key, "rename", false, func(b datamodel.Board) datamodel.Board {
		b.Name = name
		return b
	})
}

// BoardArchive archives the board keyed by key, refusing to archive the last
// active board.
func (s *Store) BoardArchive(cfg *datamodel.Config, key string) (*datamodel.BoardUpdateResult, error) {
	return s.updateBoard(cfg, key, "archive", true, func(b datamodel.Board) datamodel.Board {
		b.Archived = true
		return b
	})
}

func (s *Store) BoardUnarchive(cfg *datamodel.Config, key string) (*datamodel.BoardUpdateResult, error) {
	return s.updateBoard(cfg, key, "unarchive", false, func(b datamodel.Board) datamodel.Board {
		b.Archived = false
		return b
	})
}

func (s *Store) updateBoard(_ *datamodel.Config, key, verb string, guardLastActive bool, mutate func(datamodel.Board) datamodel.Board) (*datamodel.BoardUpdateResult, error) {
	var view datamodel.BoardView
	err := s.mutateConfig(func(data []byte, locked *datamodel.Config) (configEdit, error) {
		target, ok := locked.BoardByKey(key)
		if !ok {
			return configEdit{}, errx.User("no such board %q", key).
				WithHint("boards: %s", strings.Join(activeBoardKeys(locked.ActiveBoards()), ", "))
		}
		if guardLastActive {
			active := locked.ActiveBoards()
			if len(active) == 1 && strings.EqualFold(active[0].Key, key) {
				return configEdit{}, errx.User("cannot archive %s: it is the last active board", key).
					WithHint("create or unarchive another board first")
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

// BoardList returns every configured board, archived ones included.
func (s *Store) BoardList(cfg *datamodel.Config) (*datamodel.BoardListResult, error) {
	boards := cfg.Boards
	views := make([]datamodel.BoardView, len(boards))
	for i, b := range boards {
		views[i] = boardView(b)
	}
	return &datamodel.BoardListResult{Boards: views}, nil
}

// StateDeclared reports whether typ's workflow declares state. Every declared
// state is reachable from every other, so a board column naming an undeclared
// state is the only thing left to reject.
func StateDeclared(typ, state string) bool {
	wf, ok := datamodel.WorkflowOf(typ)
	if !ok {
		return false
	}
	_, defined := wf.StateByKey(state)
	return defined
}
