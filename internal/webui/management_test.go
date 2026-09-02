package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func requireAction(t *testing.T, d *dashboardTest, action string, fields url.Values) *httptest.ResponseRecorder {
	t.Helper()
	res := d.post(action, fields)
	if res.Code != http.StatusSeeOther {
		t.Fatalf("%s: %d %s", action, res.Code, res.Body.String())
	}
	return res
}

func followAction(d *dashboardTest, response *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	return d.get(response.Header().Get("Location"))
}

func TestManagementBoardsSprintsAndConfig(t *testing.T) {
	d := newDashboardTest(t, Options{})
	for _, step := range []struct {
		action string
		fields url.Values
	}{
		{"board-create", url.Values{"key": {"ALT"}, "name": {"Alternate"}, "description": {"Another board"}}},
		{"board-rename", url.Values{"key": {"ALT"}, "name": {"Renamed board"}}},
		{"board-archive", url.Values{"key": {"ALT"}, "archived": {"true"}}},
		{"board-archive", url.Values{"key": {"ALT"}, "archived": {"false"}}},
		{"sprint-create", url.Values{"key": {"s1"}, "name": {"First sprint"}, "start": {"2026-08-01"}, "end": {"2026-08-14"}}},
		{"sprint-create", url.Values{"key": {"s2"}, "name": {"Second sprint"}, "start": {"2026-08-15"}, "end": {"2026-08-28"}}},
		{"sprint-activate", url.Values{"key": {"s1"}}},
		{"config-set", url.Values{"key": {"project.name"}, "value": {"Managed in browser"}}},
	} {
		requireAction(t, d, step.action, step.fields)
	}
	cfg, err := d.store.Config()
	if err != nil {
		t.Fatal(err)
	}
	board, ok := cfg.BoardByKey("ALT")
	if !ok || board.Name != "Renamed board" || board.Archived || len(cfg.Sprints) != 2 || d.store.ActiveSprintKey() != "s1" || cfg.Project.Name != "Managed in browser" {
		t.Fatalf("management changes not persisted: board=%+v sprints=%+v project=%+v", board, cfg.Sprints, cfg.Project)
	}
	d.cfg = cfg
	it := d.create(t, core.CreateOpts{Title: "Sprint work", Sprint: "s1"})
	requireAction(t, d, "sprint-close", url.Values{"key": {"s1"}, "move_to": {"s2"}})
	shown, err := d.store.Show(cfg, it.ID, "")
	if err != nil || shown.Sprint == nil || *shown.Sprint != "s2" || d.store.ActiveSprintKey() != "" {
		t.Fatalf("closing sprint did not transfer work: %+v %v", shown, err)
	}
	res := d.get("/?view=manage&q=unknown%3Dquery&board=missing&type=missing&item=missing")
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "Managed in browser") || !strings.Contains(res.Body.String(), "Renamed board") {
		t.Fatalf("management must ignore item filters: %d %s", res.Code, res.Body.String())
	}
	for _, step := range []struct {
		action string
		fields url.Values
	}{
		{"config-set", url.Values{"key": {"ui.editor"}, "value": {"arbitrary"}}},
		{"config-set", url.Values{"key": {"commit.mode"}, "value": {"invalid"}}},
		{"board-create", url.Values{"key": {"../OUTSIDE"}}},
		{"sprint-create", url.Values{"key": {"invalid"}, "start": {"tomorrow"}, "end": {"yesterday"}}},
		{"sprint-activate", url.Values{"key": {"missing"}}},
	} {
		if res := d.submit(step.action, step.fields); res.Code != http.StatusBadRequest {
			t.Errorf("invalid %s: %d", step.action, res.Code)
		}
	}
}

func TestManagementCommitAndWorkonStayInCheckout(t *testing.T) {
	d := newDashboardTest(t, Options{})
	it := d.create(t, core.CreateOpts{Title: "Start this work"})
	res := requireAction(t, d, "commit", url.Values{})
	page := followAction(d, res)
	if !strings.Contains(page.Body.String(), "Committed") || strings.Contains(page.Body.String(), "commit staged changes") {
		t.Fatalf("commit result misleading: %s", page.Body.String())
	}
	if res := d.submit("workon", url.Values{"id": {it.ID}, "worktree": {"true"}}); res.Code != http.StatusBadRequest {
		t.Fatalf("worktree request must be rejected: %d", res.Code)
	}
	d.cfg.Workon.Worktree = true
	s := &server{store: d.store}
	if _, _, err := s.manage(d.cfg, "workon", url.Values{"id": {it.ID}, "no_move": {"on"}}); err != nil {
		t.Fatal(err)
	}
	now, err := d.store.Now(d.cfg)
	if err != nil || now.ID != it.ID || now.State != "TODO" || now.Branch == "main" {
		t.Fatalf("workon without transition: %+v %v", now, err)
	}
	repo := gitx.Repo{Dir: d.store.Root()}
	worktrees, err := repo.Output("worktree", "list", "--porcelain")
	if err != nil || strings.Count(worktrees, "worktree ") != 1 {
		t.Fatalf("workon must not create another worktree: %s %v", worktrees, err)
	}
	requireAction(t, d, "workon", url.Values{"id": {it.ID}})
	now, err = d.store.Now(d.cfg)
	if err != nil || now.State != "IN_PROGRESS" {
		t.Fatalf("workon did not use core transition: %+v %v", now, err)
	}
}

func TestManagementSyncUsesExplicitPolicyAndPush(t *testing.T) {
	d := newDashboardTest(t, Options{})
	requireAction(t, d, "config-set", url.Values{"key": {"sync.push"}, "value": {"true"}})
	requireAction(t, d, "commit", url.Values{})
	remote := gitx.Repo{Dir: t.TempDir()}
	if _, err := remote.Output("init", "--bare", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	repo := gitx.Repo{Dir: d.store.Root()}
	if _, err := repo.Output("remote", "add", "origin", remote.Dir); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Output("push", "--set-upstream", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	before, err := remote.Output("rev-parse", "main")
	if err != nil {
		t.Fatal(err)
	}
	d.create(t, core.CreateOpts{Title: "Local change"})
	res := d.submit("sync", url.Values{"dirty": {"fail"}})
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "uncommitted kira changes") {
		t.Fatalf("sync must refuse dirty tree by default: %d %s", res.Code, res.Body.String())
	}
	requireAction(t, d, "sync", url.Values{"dirty": {"commit"}})
	after, err := remote.Output("rev-parse", "main")
	if err != nil || before != after {
		t.Fatalf("unchecked push changed remote: before=%s after=%s err=%v", before, after, err)
	}
	head, err := repo.Output("rev-parse", "HEAD")
	if err != nil || head == before {
		t.Fatalf("commit dirty policy did not commit local changes: head=%s err=%v", head, err)
	}
	res = requireAction(t, d, "sync", url.Values{"dirty": {"fail"}, "push": {"on"}})
	after, err = remote.Output("rev-parse", "main")
	if err != nil || after != head {
		t.Fatalf("checked push did not publish: head=%s remote=%s err=%v", head, after, err)
	}
	if page := followAction(d, res); !strings.Contains(page.Body.String(), "push: done") {
		t.Fatalf("sync report missing: %s", page.Body.String())
	}
	if res := d.submit("sync", url.Values{"dirty": {"unexpected"}}); res.Code != http.StatusBadRequest {
		t.Errorf("invalid dirty policy returned %d", res.Code)
	}
}

func TestStatsScope(t *testing.T) {
	d := newDashboardTest(t, Options{})
	epic := d.create(t, core.CreateOpts{Type: "epic", Title: "Scoped epic"})
	d.create(t, core.CreateOpts{Title: "Included", Parent: epic.ID})
	d.create(t, core.CreateOpts{Title: "Outside epic"})
	s := &server{store: d.store, token: d.token}
	u, _ := url.Parse("/?view=stats&epic=" + epic.ID + "&weeks=4&since=2020-01-01")
	p, err := s.loadPage(t.Context(), u)
	if err != nil || p.Stats.Scope.Parent != epic.ID || p.Stats.Scope.Weeks != 4 || p.Stats.Scope.Since != "2020-01-01" || p.Stats.Completion.Total != 1 {
		t.Fatalf("stats scope not forwarded: %+v %v", p.Stats, err)
	}
	for _, path := range []string{"/?view=stats&weeks=invalid", "/?view=stats&weeks=999999", "/?view=stats&since=yesterday"} {
		if res := d.get(path); res.Code != http.StatusBadRequest {
			t.Errorf("invalid scope %s: %d", path, res.Code)
		}
	}
}

func TestMoveResolutionAndCreateRedirect(t *testing.T) {
	d := newDashboardTest(t, Options{})
	it := d.create(t, core.CreateOpts{Title: "Finish normally"})
	for _, state := range []string{"IN_PROGRESS", "REVIEW"} {
		requireAction(t, d, "move", url.Values{"id": {it.ID}, "state": {state}})
	}
	requireAction(t, d, "move", url.Values{"id": {it.ID}, "state": {"DONE"}, "resolution": {"done"}})
	shown, err := d.store.Show(d.cfg, it.ID, "")
	if err != nil || shown.State != "DONE" || shown.Resolution == nil || *shown.Resolution != "done" {
		t.Fatalf("move resolution not persisted: %+v %v", shown, err)
	}
	res := requireAction(t, d, "create", url.Values{"title": {"Created from settings"}, "return_to": {"/?view=manage&item=" + it.ID + "&commit=old"}})
	location, err := url.Parse(res.Header().Get("Location"))
	if err != nil || location.Query().Get("view") != "tree" || location.Query().Get("commit") != "" || location.Query().Get("item") == it.Number {
		t.Fatalf("create redirect retained old detail state: %s %v", location, err)
	}
}
