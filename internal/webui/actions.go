package webui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/shivamshivanshu/kira/internal/commandline"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func (s *server) action(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, s.actionRedirect(r), http.StatusSeeOther)
}

func (s *server) actionRedirect(r *http.Request) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := strings.TrimPrefix(r.URL.Path, "/actions/")
	back, _ := url.Parse(localReturn(r.PostForm.Get("return_to")))
	cfg, err := s.store.Config()
	var message, selected string
	var bulk []bulkOutcome
	if err == nil {
		if name == "bulk" {
			bulk, err = s.bulk(cfg, r.PostForm)
			if err == nil {
				message = bulkMessage(bulk)
			}
		} else {
			message, selected, err = s.mutate(cfg, name, r.PostForm)
		}
		if err == nil && stagesChanges(name) && (cfg.Commit.Mode == datamodel.CommitManual || cfg.Commit.Mode == datamodel.CommitPrompt) {
			message = joinMessages(message, "Commit mode: "+string(cfg.Commit.Mode)+". Run kira commit in the command bar to commit staged changes.")
		}
	}
	if selected != "" {
		q := back.Query()
		if q.Get("item") != selected {
			q.Del("commit")
		}
		q.Set("item", selected)
		if q.Get("view") == "manage" && (name == "create" || name == "workon") {
			q.Set("view", "tree")
		}
		back.RawQuery = q.Encode()
	}
	flash := flashMessage{Message: message, Err: err, Bulk: bulk}
	if err != nil {
		flash.Form, flash.Action = r.PostForm, name
	} else if bulk != nil {
		flash.Form = bulkRetry(r.PostForm, bulk)
		flash.Action = "bulk"
	}
	if token := s.setFlash(flash); token != "" {
		q := back.Query()
		q.Set("_notice", token)
		back.RawQuery = q.Encode()
	}
	return back.RequestURI()
}

func stagesChanges(action string) bool {
	switch action {
	case "command", "commit", "sync", "sprint-activate":
		return false
	default:
		return true
	}
}

func (s *server) setFlash(flash flashMessage) string {
	if flash.Message == "" && flash.Err == nil {
		return ""
	}
	token, err := randomToken()
	if err != nil {
		return ""
	}
	if len(s.flash) >= 64 {
		clear(s.flash)
	}
	if len(flash.Message) > 64<<10 {
		flash.Message = flash.Message[:64<<10] + "\n(output truncated)"
	}
	s.flash[token] = flash
	return token
}

func (s *server) mutate(cfg *datamodel.Config, action string, f url.Values) (message, selected string, err error) {
	selected = f.Get("id")
	switch action {
	case "create":
		opts := core.CreateOpts{Type: f.Get("type"), Title: f.Get("title"), Board: f.Get("board"), Owner: f.Get("owner"), Priority: f.Get("priority"), Parent: f.Get("epic"), NoEdit: true}
		if opts.Type == "" {
			opts.Type = datamodel.TypeTicket
		}
		if strings.TrimSpace(f.Get("body")) != "" {
			body := f.Get("body")
			opts.Body = &body
		}
		res, err := s.store.Create(cfg, opts)
		if err != nil {
			return "", "", err
		}
		return joinMessages("Created "+res.Number, warningText(res.Warnings)), res.Number, nil
	case "edit":
		if f.Get("revision") == "" {
			return "", selected, errx.User("missing item revision; refresh the dashboard before editing")
		}
		opts := core.EditOpts{ExpectedRevision: f.Get("revision")}
		for _, key := range []string{"title", "owner", "priority", "sprint", "due", "labels", "subtype", "resolution", "estimate"} {
			if f.Has(key) {
				opts.Fields = append(opts.Fields, core.FieldEdit{Key: key, Value: f.Get(key)})
			}
		}
		if f.Has("body") {
			body := f.Get("body")
			opts.Description = &body
		}
		if len(opts.Fields) == 0 && opts.Description == nil {
			return "", selected, errx.User("no fields to edit")
		}
		res, err := s.store.Edit(cfg, selected, opts)
		return mutationMessage(res, err, "Updated", selected)
	case "move":
		res, err := s.store.Move(cfg, selected, f.Get("state"), core.MoveOpts{Resolution: f.Get("resolution")})
		if err != nil {
			return "", selected, err
		}
		return joinMessages("Moved "+res.Number+" to "+res.To, strings.Join(res.Warnings, "\n")), res.Number, nil
	case "comment":
		res, err := s.store.Comment(cfg, selected, core.CommentOpts{Message: f.Get("text"), HasMessage: true})
		if err != nil {
			return "", selected, err
		}
		return "Comment added to " + res.Number, res.Number, nil
	case "assign":
		res, err := s.store.Assign(cfg, selected, f.Get("owner"), core.AssignOpts{})
		return mutationMessage(res, err, "Assigned", selected)
	case "link":
		opts := core.LinkOpts{Ref: f.Get("target"), Remove: f.Get("remove") != "" && f.Get("remove") != "false"}
		switch f.Get("kind") {
		case "parent":
			opts.Target = core.LinkEpic
		case "blocked-by":
			opts.Target = core.LinkBlockedBy
		case "relates", "duplicate-of":
			opts.Target = core.LinkTyped
			opts.Type = strings.ReplaceAll(f.Get("kind"), "-", "_")
		default:
			return "", selected, errx.User("unknown relationship %q", f.Get("kind"))
		}
		res, err := s.store.Link(cfg, selected, opts)
		return mutationMessage(res, err, "Updated relationships for", selected)
	case "command":
		selected = f.Get("selected")
		argv, err := commandArgs(f.Get("line"), selected)
		if err != nil {
			return "", selected, err
		}
		if s.opts.RunCommand == nil {
			return "", selected, errx.User("command runner unavailable")
		}
		output, err := s.opts.RunCommand(argv)
		if err != nil && output != "" {
			return "", selected, fmt.Errorf("%w\n%s", err, output)
		}
		if output == "" && err == nil {
			output = "Command completed."
		}
		return output, selected, err
	default:
		return s.manage(cfg, action, f)
	}
}

func mutationMessage(res *datamodel.MutationResult, err error, verb, selected string) (string, string, error) {
	if err != nil {
		return "", selected, err
	}
	return joinMessages(verb+" "+res.Number, warningText(res.Warnings)), res.Number, nil
}

func commandArgs(line, selected string) ([]string, error) {
	args, err := commandline.Parse(line)
	if err != nil {
		return nil, err
	}
	if len(args) > 0 && args[0] == "kira" {
		args = args[1:]
	}
	if len(args) == 0 {
		return nil, errx.User("enter a kira command")
	}
	for i, arg := range args {
		if arg == "." {
			if selected == "" {
				return nil, errx.User("select an item before using . in a command")
			}
			args[i] = selected
		}
	}
	return args, nil
}
