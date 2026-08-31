package webui

import (
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"sync"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/errx"
)

type Options struct {
	RunCommand func([]string) (string, error)
}

type server struct {
	store *core.Store
	opts  Options
	token string
	mu    sync.Mutex
	flash map[string]flashMessage
}

type flashMessage struct {
	Message string
	Err     error
	Bulk    []bulkOutcome
	Form    url.Values
	Action  string
}

func New(store *core.Store, opts Options) (http.Handler, error) {
	if store == nil {
		return nil, errors.New("webui requires a store")
	}
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	s := &server{store: store, opts: opts, token: token, flash: make(map[string]flashMessage)}
	mux := http.NewServeMux()
	mux.Handle("GET "+assetURL(""), staticHandler(static))
	mux.HandleFunc("GET /{$}", s.dashboard)
	for _, action := range []string{
		"create", "edit", "move", "comment", "assign", "link", "command", "bulk",
		"board-create", "board-rename", "board-archive", "sprint-create", "sprint-activate", "sprint-close",
		"config-set", "commit", "workon", "sync",
	} {
		mux.HandleFunc("POST /actions/"+action, s.action)
	}
	return s.secure(mux), nil
}

func (s *server) dashboard(w http.ResponseWriter, r *http.Request) {
	p, err := s.dashboardPage(r)
	status := http.StatusOK
	if err != nil {
		p.Error = userError(err)
		status = errorStatus(err)
	}
	writePage(w, p, status)
}

func (s *server) dashboardPage(r *http.Request) (pageData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.loadPage(r.Context(), r.URL)
	if token := r.URL.Query().Get("_notice"); token != "" {
		flash := s.flash[token]
		p.Message = joinMessages(p.Message, flash.Message)
		p.Bulk, p.Form, p.Action = flash.Bulk, flash.Form, flash.Action
		if flash.Err != nil {
			err = flash.Err
		}
		if p.Action == "edit" {
			p.Revision = p.Form.Get("revision")
		}
		delete(s.flash, token)
	}
	return p, err
}

func errorStatus(err error) int {
	if e, ok := errors.AsType[*errx.Error](err); ok {
		switch e.Code {
		case errx.ExitConflict:
			return http.StatusConflict
		case errx.ExitEnv, errx.ExitCrash:
			return http.StatusInternalServerError
		}
	}
	return http.StatusBadRequest
}

func userError(err error) string {
	if e, ok := errors.AsType[*errx.Error](err); ok && e.Hint != "" {
		return err.Error() + "\n" + e.Hint
	}
	return err.Error()
}
