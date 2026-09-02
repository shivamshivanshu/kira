package webui

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

func (s *server) manage(cfg *datamodel.Config, action string, f url.Values) (string, string, error) {
	key := f.Get("key")
	switch action {
	case "board-create":
		res, err := s.store.BoardCreate(cfg, key, f.Get("name"), f.Get("description"))
		if err != nil {
			return "", "", err
		}
		return "Created board " + res.Board.Key, "", nil
	case "board-rename":
		res, err := s.store.BoardRename(cfg, key, f.Get("name"))
		if err != nil {
			return "", "", err
		}
		return "Renamed board " + res.Board.Key + " to " + res.Board.Name, "", nil
	case "board-archive":
		archived, err := formBool(f, "archived")
		if err != nil {
			return "", "", err
		}
		var res *datamodel.BoardUpdateResult
		verb := "Unarchived"
		if archived {
			verb = "Archived"
			res, err = s.store.BoardArchive(cfg, key)
		} else {
			res, err = s.store.BoardUnarchive(cfg, key)
		}
		if err != nil {
			return "", "", err
		}
		return verb + " board " + res.Board.Key, "", nil
	case "sprint-create":
		res, err := s.store.SprintCreate(cfg, datamodel.Sprint{Key: key, Name: f.Get("name"), Start: f.Get("start"), End: f.Get("end")})
		if err != nil {
			return "", "", err
		}
		return "Created sprint " + res.Sprint.Key, "", nil
	case "sprint-activate":
		res, err := s.store.SprintActivate(cfg, key)
		if err != nil {
			return "", "", err
		}
		return "Activated sprint " + res.Activated, "", nil
	case "sprint-close":
		res, err := s.store.SprintClose(cfg, key, f.Get("move_to"))
		if err != nil {
			return "", "", err
		}
		message := "Closed sprint " + res.Closed
		if res.MovedTo != "" {
			message += fmt.Sprintf("; moved %d unfinished items to %s", len(res.Unfinished), res.MovedTo)
		} else if len(res.Unfinished) > 0 {
			message += "; unfinished items remain assigned: " + strings.Join(res.Unfinished, ", ")
		}
		return joinMessages(message, warningText(res.Warnings)), "", nil
	case "config-set":
		res, err := s.store.ConfigSet(cfg, key, f.Get("value"))
		if err != nil {
			return "", "", err
		}
		return "Set " + res.Key + " = " + res.Value, "", nil
	case "commit":
		res, err := s.store.CommitKira(cfg)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("Committed %d files: %s\n%s", res.Files, res.Subject, res.SHA), "", nil
	case "workon":
		noMove, err := formBool(f, "no_move")
		if err != nil {
			return "", "", err
		}
		if f.Get("worktree") != "" && f.Get("worktree") != "false" {
			return "", "", errx.User("the dashboard can only switch branches in the current checkout")
		}
		res, err := s.store.Workon(cfg, f.Get("id"), core.WorkonOpts{NoMove: noMove, Worktree: false})
		if err != nil {
			return "", f.Get("id"), err
		}
		return joinMessages("Working on "+res.Number+" in branch "+res.Branch, warningText(res.Warnings)), res.Number, nil
	case "sync":
		return s.sync(cfg, f)
	default:
		return "", "", errx.User("unknown action %q", action)
	}
}

func (s *server) sync(cfg *datamodel.Config, f url.Values) (string, string, error) {
	push, err := formBool(f, "push")
	if err != nil {
		return "", "", err
	}
	dirty := syncx.DirtyPolicy(f.Get("dirty"))
	if dirty == syncx.DirtyAuto {
		dirty = syncx.DirtyFail
	}
	if dirty != syncx.DirtyFail && dirty != syncx.DirtyCommit && dirty != syncx.DirtyStash {
		return "", "", errx.User("dirty policy must be fail, commit, or stash")
	}
	requestConfig := *cfg
	requestConfig.Sync.Push = false
	report, err := s.store.Sync(&requestConfig, core.SyncOpts{Push: push, Dirty: dirty})
	var lines []string
	if report != nil {
		for _, step := range report.Steps {
			lines = append(lines, fmt.Sprintf("%s: %s %s", step.Name, step.Status, step.Detail))
			if err == nil && step.Status == syncx.StepFailed {
				err = errx.User("sync finished with failed steps; review the step results")
			}
		}
	}
	return strings.Join(lines, "\n"), "", err
}

func formBool(f url.Values, key string) (bool, error) {
	switch f.Get(key) {
	case "true", "on", "1":
		return true, nil
	case "", "false", "0":
		return false, nil
	default:
		return false, errx.User("%s must be true or false", key)
	}
}
