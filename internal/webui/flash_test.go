package webui

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func TestInterleavedActionResultsStayWithTheirRedirect(t *testing.T) {
	d := newDashboardTest(t, Options{})
	first := d.create(t, core.CreateOpts{Title: "First item"})
	second := d.create(t, core.CreateOpts{Title: "Second item"})
	a := requireAction(t, d, "assign", url.Values{"id": {first.ID}, "owner": {"tester"}})
	b := requireAction(t, d, "assign", url.Values{"id": {second.ID}, "owner": {"tester"}})
	for _, tc := range []struct {
		path, want, unwanted string
	}{
		{b.Header().Get("Location"), "Assigned " + second.Number, "Assigned " + first.Number},
		{a.Header().Get("Location"), "Assigned " + first.Number, "Assigned " + second.Number},
	} {
		u, err := url.Parse(tc.path)
		if err != nil || u.Query().Get("_notice") == "" {
			t.Fatal("action redirect needs its own result token")
		}
		page := d.get(tc.path)
		if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), tc.want) || strings.Contains(page.Body.String(), tc.unwanted) {
			t.Fatal("action result was lost or delivered to the wrong redirect")
		}
		if strings.Contains(page.Body.String(), "_notice") {
			t.Fatal("result tokens must not propagate into page links or forms")
		}
		if page = d.get(tc.path); page.Code != http.StatusOK || strings.Contains(page.Body.String(), tc.want) {
			t.Fatal("action result must only be consumed once")
		}
	}
	if a.Header().Get("Set-Cookie") != "" || b.Header().Get("Set-Cookie") != "" {
		t.Fatal("per-redirect results must not use shared cookies")
	}
	page := d.get("/?_notice=unknown")
	if page.Code != http.StatusOK || strings.Contains(page.Body.String(), "_notice") {
		t.Fatal("unknown result token should be ignored and removed from page URLs")
	}
}

func TestFailedActionRedirectDoesNotReplayOnRefresh(t *testing.T) {
	calls := 0
	d := newDashboardTest(t, Options{RunCommand: func([]string) (string, error) {
		calls++
		return "Partial result", errx.User("Command failed after starting")
	}})
	res := d.post("command", url.Values{"line": {"sync"}, "return_to": {"/?view=manage"}})
	if res.Code != http.StatusSeeOther {
		t.Fatalf("failed action must redirect to a safe GET: %d", res.Code)
	}
	location := res.Header().Get("Location")
	page := d.get(location)
	if page.Code != http.StatusBadRequest {
		t.Fatalf("error page status = %d", page.Code)
	}
	for _, want := range []string{"Partial result", "Command failed after starting", `name="line" value="sync"`} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("error page lost %q", want)
		}
	}
	page = d.get(location)
	if page.Code != http.StatusOK || calls != 1 {
		t.Fatalf("refresh replayed a failed command: status=%d calls=%d", page.Code, calls)
	}
}

func TestCommandResultsKeepTheirSelectedItem(t *testing.T) {
	for _, fail := range []bool{false, true} {
		d := newDashboardTest(t, Options{RunCommand: func([]string) (string, error) {
			if fail {
				return "", errx.User("Retry this command")
			}
			return "Command finished", nil
		}})
		open := d.create(t, core.CreateOpts{Title: "Open detail"})
		selected := d.create(t, core.CreateOpts{Title: "Command target"})
		res := requireAction(t, d, "command", url.Values{
			"selected": {selected.ID}, "line": {"show ."}, "return_to": {"/?item=" + open.ID},
		})
		u, err := url.Parse(res.Header().Get("Location"))
		if err != nil || u.Query().Get("item") != selected.ID {
			t.Fatal("command redirect changed its selected item")
		}
		page := followAction(d, res)
		if !strings.Contains(page.Body.String(), `name="selected" value="`+selected.ID+`"`) {
			t.Fatal("command retry changed its target to the previously open detail")
		}
	}
}

func TestBlankCreateDescriptionUsesCoreTemplate(t *testing.T) {
	d := newDashboardTest(t, Options{})
	res := requireAction(t, d, "create", url.Values{"title": {"Use the template"}, "body": {""}})
	u, err := url.Parse(res.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	it, err := d.store.Show(d.cfg, u.Query().Get("item"), "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(it.Body, "## Description") || !strings.Contains(it.Body, "## Acceptance criteria") {
		t.Fatal("blank optional description discarded the existing core template")
	}
}
