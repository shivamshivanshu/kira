package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

type dashboardTest struct {
	store   *core.Store
	cfg     *datamodel.Config
	handler http.Handler
	token   string
}

func newDashboardTest(t *testing.T, opts Options) *dashboardTest {
	t.Helper()
	dir := testutil.InitGitRepo(t)
	if _, err := core.Init(dir, "WEB", false); err != nil {
		t.Fatal(err)
	}
	store, err := core.Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Config()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConfigSet(cfg, "commit.mode", "manual"); err != nil {
		t.Fatal(err)
	}
	cfg, err = store.Config()
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(store, opts)
	if err != nil {
		t.Fatal(err)
	}
	d := &dashboardTest{store: store, cfg: cfg, handler: handler}
	res := d.get("/")
	if res.Code != http.StatusOK {
		t.Fatalf("initial dashboard: %d %s", res.Code, res.Body.String())
	}
	match := regexp.MustCompile(`name="csrf" value="([a-f0-9]+)"`).FindStringSubmatch(res.Body.String())
	if len(match) != 2 {
		t.Fatalf("dashboard missing CSRF form field: %s", res.Body.String())
	}
	d.token = match[1]
	return d
}

func (d *dashboardTest) create(t *testing.T, opts core.CreateOpts) *datamodel.CreateResult {
	t.Helper()
	if opts.Type == "" {
		opts.Type = datamodel.TypeTicket
	}
	opts.NoEdit = true
	res, err := d.store.Create(d.cfg, opts)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func (d *dashboardTest) get(path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "http://localhost:7432"+path, nil)
	w := httptest.NewRecorder()
	d.handler.ServeHTTP(w, r)
	return w
}

func (d *dashboardTest) post(action string, fields url.Values) *httptest.ResponseRecorder {
	fields.Set("csrf", d.token)
	if action == "edit" && !fields.Has("revision") {
		_, raw, err := d.store.ResolveItemFile(d.cfg, fields.Get("id"))
		if err == nil {
			fields.Set("revision", core.ItemRevision(raw))
		}
	}
	r := httptest.NewRequest(http.MethodPost, "http://localhost:7432/actions/"+action, strings.NewReader(fields.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "http://localhost:7432")
	w := httptest.NewRecorder()
	d.handler.ServeHTTP(w, r)
	return w
}

func (d *dashboardTest) submit(action string, fields url.Values) *httptest.ResponseRecorder {
	res := d.post(action, fields)
	if res.Code == http.StatusSeeOther {
		return followAction(d, res)
	}
	return res
}

func TestDashboardViewsAndScopedQueries(t *testing.T) {
	d := newDashboardTest(t, Options{})
	epic := d.create(t, core.CreateOpts{Type: datamodel.TypeEpic, Title: "Delivery epic"})
	ticket := d.create(t, core.CreateOpts{Title: "Filtered child", Parent: epic.Number})
	d.create(t, core.CreateOpts{Title: "Unrelated task"})
	for _, view := range []string{"tree", "board", "stats"} {
		res := d.get("/?view=" + view)
		if res.Code != http.StatusOK || !strings.HasPrefix(res.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("%s: %d %s", view, res.Code, res.Body.String())
		}
	}
	res := d.get("/?q=" + url.QueryEscape(`"Filtered child"`))
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `<span class="item-title">Delivery epic</span>`) || !strings.Contains(res.Body.String(), `<span class="item-title">Filtered child</span>`) || strings.Contains(res.Body.String(), `<span class="item-title">Unrelated task</span>`) {
		t.Fatalf("filtered hierarchy did not preserve parent context: %d %s", res.Code, res.Body.String())
	}
	if _, err := d.store.BoardCreate(d.cfg, "ALT", "Other board", ""); err != nil {
		t.Fatal(err)
	}
	var err error
	d.cfg, err = d.store.Config()
	if err != nil {
		t.Fatal(err)
	}
	d.create(t, core.CreateOpts{Title: "Elsewhere", Board: "ALT"})
	query := url.Values{"view": {"board"}, "board": {"WEB"}, "q": {"type=ticket OR type=epic ORDER BY created DESC"}}
	res = d.get("/?" + query.Encode())
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "<h3>"+ticket.Title+"</h3>") || strings.Contains(res.Body.String(), "<h3>Elsewhere</h3>") {
		t.Fatalf("scoped ordered board: %d %s", res.Code, res.Body.String())
	}
	res = d.get("/?view=stats&q=invalid%3Dquery&board=missing")
	if res.Code != http.StatusOK {
		t.Fatalf("stats should use its supported scopes: %d %s", res.Code, res.Body.String())
	}
	for _, path := range []string{"/?q=invalid%3Dquery", "/?filter=missing", "/?board=missing", "/?item=WEB-99999", "/?view=unknown", "/?board=WEB&q=ORDER+BY+created"} {
		if res := d.get(path); res.Code != http.StatusBadRequest {
			t.Errorf("%s returned %d", path, res.Code)
		}
	}
}

func TestDashboardScopesPreserveCoreQueryGrammar(t *testing.T) {
	d := newDashboardTest(t, Options{})
	for _, step := range []struct {
		action string
		fields url.Values
	}{
		{"board-create", url.Values{"key": {"ALT"}}},
		{"sprint-create", url.Values{"key": {"s1"}, "name": {"First"}, "start": {"2026-08-01"}, "end": {"2026-08-14"}}},
		{"sprint-create", url.Values{"key": {"s2"}, "name": {"Second"}, "start": {"2026-08-15"}, "end": {"2026-08-28"}}},
	} {
		if page := d.submit(step.action, step.fields); page.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", step.action, page.Code, page.Body.String())
		}
	}
	var err error
	d.cfg, err = d.store.Config()
	if err != nil {
		t.Fatal(err)
	}
	want := d.create(t, core.CreateOpts{Title: "BY created target", Owner: "ORDER", Sprint: "s1"})
	d.create(t, core.CreateOpts{Title: "BY created other sprint", Owner: "ORDER", Sprint: "s2"})
	d.create(t, core.CreateOpts{Title: "BY created other board", Owner: "ORDER", Sprint: "s1", Board: "ALT"})
	s := &server{store: d.store}
	for _, view := range []string{"tree", "board"} {
		u := &url.URL{Path: "/", RawQuery: url.Values{
			"view": {view}, "board": {"web"}, "sprint": {"s1"}, "q": {"owner=ORDER BY created"},
		}.Encode()}
		p, err := s.loadPage(t.Context(), u)
		if err != nil {
			t.Fatalf("%s changed core query grammar: %v", view, err)
		}
		if len(p.Items) != 1 || p.Items[0].ID != want.ID || p.BoardKey != "WEB" {
			t.Fatalf("%s scope mismatch: board=%s items=%+v", view, p.BoardKey, p.Items)
		}
		if view == "board" {
			var ids []string
			for _, column := range p.Board.Columns {
				for _, item := range column.Items {
					ids = append(ids, item.ID)
				}
			}
			if !reflect.DeepEqual(ids, []string{want.ID}) {
				t.Fatalf("board and list scope differ: %v", ids)
			}
		}
	}
}

func TestDashboardActionsUseCore(t *testing.T) {
	d := newDashboardTest(t, Options{})
	res := d.post("create", url.Values{"title": {"From browser"}, "type": {"ticket"}, "body": {"Original description"}})
	if res.Code != http.StatusSeeOther {
		t.Fatalf("create: %d %s", res.Code, res.Body.String())
	}
	redirect, err := url.Parse(res.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	ref := redirect.Query().Get("item")
	if ref == "" {
		t.Fatal("create did not select new item")
	}
	for _, step := range []struct {
		action string
		fields url.Values
	}{
		{"assign", url.Values{"owner": {"tester"}}},
		{"comment", url.Values{"text": {"Keep this comment"}}},
		{"edit", url.Values{"title": {"Edited in browser"}, "body": {"Updated description"}, "labels": {"web,testing"}}},
		{"move", url.Values{"state": {"IN_PROGRESS"}}},
	} {
		step.fields.Set("id", ref)
		res = d.post(step.action, step.fields)
		if res.Code != http.StatusSeeOther {
			t.Fatalf("%s: %d %s", step.action, res.Code, res.Body.String())
		}
	}
	detail, err := d.store.Show(d.cfg, ref, "")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Title != "Edited in browser" || detail.State != "IN_PROGRESS" || detail.Owner == nil || *detail.Owner != "tester" || !strings.Contains(detail.Body, "Updated description") || len(detail.Comments) != 1 || detail.Comments[0].Text != "Keep this comment" {
		t.Fatalf("persisted item: %+v", detail)
	}
	blocker := d.create(t, core.CreateOpts{Title: "Blocker"})
	for _, remove := range []string{"", "true"} {
		res = d.post("link", url.Values{"id": {ref}, "kind": {"blocked-by"}, "target": {blocker.Number}, "remove": {remove}})
		if res.Code != http.StatusSeeOther {
			t.Fatalf("link remove=%s: %d %s", remove, res.Code, res.Body.String())
		}
		detail, err = d.store.Show(d.cfg, ref, "")
		if err != nil {
			t.Fatal(err)
		}
		if (len(detail.BlockedBy) == 0) != (remove != "") {
			t.Fatalf("link remove=%s: %+v", remove, detail.BlockedBy)
		}
	}
	for _, step := range []struct {
		action string
		fields url.Values
	}{
		{"move", url.Values{"state": {"MISSING"}}},
		{"comment", url.Values{"text": {""}}},
		{"link", url.Values{"kind": {"blocked-by"}, "target": {ref}}},
		{"edit", url.Values{"title": {""}}},
	} {
		step.fields.Set("id", ref)
		res = d.submit(step.action, step.fields)
		if res.Code != http.StatusBadRequest {
			t.Errorf("invalid %s: %d %s", step.action, res.Code, res.Body.String())
		}
	}
}

func TestDashboardStaleEditAndFlash(t *testing.T) {
	d := newDashboardTest(t, Options{})
	it := d.create(t, core.CreateOpts{Title: "Existing"})
	_, raw, err := d.store.ResolveItemFile(d.cfg, it.Number)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.store.Comment(d.cfg, it.Number, core.CommentOpts{Message: "Concurrent comment", HasMessage: true}); err != nil {
		t.Fatal(err)
	}
	res := d.submit("edit", url.Values{"id": {it.Number}, "title": {"Unsaved browser title"}, "revision": {core.ItemRevision(raw)}})
	if res.Code != http.StatusConflict || !strings.Contains(res.Body.String(), "Unsaved browser title") {
		t.Fatalf("stale edit should retain submitted values: %d %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `name="revision" value="`+core.ItemRevision(raw)+`"`) {
		t.Fatal("stale edit retry must retain its original revision")
	}
	res = d.submit("edit", url.Values{"id": {it.Number}, "title": {"Missing revision"}, "revision": {""}})
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "missing item revision") {
		t.Fatalf("missing revision: %d %s", res.Code, res.Body.String())
	}
	res = d.post("assign", url.Values{"id": {it.Number}, "owner": {"tester"}, "return_to": {"https://example.org/"}})
	if res.Code != http.StatusSeeOther || strings.HasPrefix(res.Header().Get("Location"), "http") {
		t.Fatalf("unsafe redirect: %d %s", res.Code, res.Header().Get("Location"))
	}
	w := followAction(d, res)
	if !strings.Contains(w.Body.String(), "Assigned "+it.Number) {
		t.Fatalf("missing action result: %s", w.Body.String())
	}
}

func TestDashboardSecurity(t *testing.T) {
	d := newDashboardTest(t, Options{})
	for _, tc := range []struct {
		name, host, origin, site, token string
	}{
		{name: "missing token", host: "localhost:7432"},
		{name: "wrong token", host: "localhost:7432", token: "wrong"},
		{name: "remote host", host: "attacker.example", token: d.token},
		{name: "cross origin", host: "localhost:7432", origin: "https://attacker.example", token: d.token},
		{name: "other port", host: "localhost:7432", origin: "http://localhost:9999", token: d.token},
		{name: "same site", host: "localhost:7432", site: "same-site", token: d.token},
	} {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{"csrf": {tc.token}, "title": {"Unauthorized"}}
			r := httptest.NewRequest(http.MethodPost, "http://localhost:7432/actions/create", strings.NewReader(form.Encode()))
			r.Host = tc.host
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Sec-Fetch-Site", tc.site)
			w := httptest.NewRecorder()
			d.handler.ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
	if res := d.get("/actions/create?title=Unauthorized&csrf=" + d.token); res.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET mutation returned %d", res.Code)
	}
	if res := d.post("create", url.Values{"title": {strings.Repeat("x", 1<<20)}}); res.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body returned %d", res.Code)
	}
	items, err := d.store.List(d.cfg, core.ListOpts{})
	if err != nil || len(items.Items) != 0 {
		t.Fatalf("security checks allowed mutation: %+v %v", items, err)
	}
	it := d.create(t, core.CreateOpts{Title: `<script>alert("unsafe")</script>`})
	res := d.get("/?item=" + it.Number)
	if strings.Contains(res.Body.String(), `<script>alert("unsafe")</script>`) || !strings.Contains(res.Body.String(), "&lt;script&gt;") {
		t.Fatal("item title was not HTML escaped")
	}
	if !strings.Contains(res.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") || res.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing security headers")
	}
}

func TestCommandBridge(t *testing.T) {
	var got []string
	d := newDashboardTest(t, Options{RunCommand: func(args []string) (string, error) {
		got = args
		return "Existing CLI output", nil
	}})
	res := d.post("command", url.Values{"line": {`kira comment . -m "A quoted message"`}, "selected": {"WEB-7"}})
	if res.Code != http.StatusSeeOther || !reflect.DeepEqual(got, []string{"comment", "WEB-7", "-m", "A quoted message"}) {
		t.Fatalf("command: %d args=%q %s", res.Code, got, res.Body.String())
	}
}

func TestDashboardConfigFailure(t *testing.T) {
	d := newDashboardTest(t, Options{})
	if err := os.WriteFile(filepath.Join(d.store.Root(), ".kira", "config.yaml"), []byte("invalid: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	res := d.get("/")
	if res.Code != http.StatusBadRequest || !strings.HasPrefix(res.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("invalid config: %d %s", res.Code, res.Body.String())
	}
}

func TestCommandArgs(t *testing.T) {
	for _, tc := range []struct {
		line, selected string
		want           []string
	}{
		{`comment . -m 'hello world'`, "WEB-1", []string{"comment", "WEB-1", "-m", "hello world"}},
		{`edit . --field owner=""`, "WEB-1", []string{"edit", "WEB-1", "--field", "owner="}},
		{`comment WEB-1 -m "say \"hi\""`, "", []string{"comment", "WEB-1", "-m", `say "hi"`}},
		{`comment WEB-1 -m '$(touch /tmp/no)'`, "", []string{"comment", "WEB-1", "-m", "$(touch /tmp/no)"}},
	} {
		got, err := commandArgs(tc.line, tc.selected)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: %q, %v", tc.line, got, err)
		}
	}
	for _, line := range []string{"", "kira", "comment .", `comment WEB-1 -m "unfinished`, `show WEB-1\`} {
		if _, err := commandArgs(line, ""); err == nil {
			t.Errorf("%q should fail", line)
		}
	}
}

func TestLocalReturn(t *testing.T) {
	for _, raw := range []string{"https://example.com/", "//example.com/", `/\example.com`, "/actions/create", "/#fragment", "/%0d%0aLocation:evil"} {
		if got := localReturn(raw); got != "/" {
			t.Errorf("unsafe return %q accepted as %q", raw, got)
		}
	}
	if got := localReturn("/?view=board&item=WEB-1"); got != "/?view=board&item=WEB-1" {
		t.Errorf("local return lost: %s", got)
	}
}
