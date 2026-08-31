package webui

import (
	"bytes"
	"html/template"
	"net/url"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/showfmt"
)

func TestMarkdownOmitsExecutableContent(t *testing.T) {
	body, err := markdown("# Safe heading\n\n<script>alert('bad')</script>\n\n<img src=x onerror=alert(1)>\n\n[unsafe](javascript:alert(1))\n\n**Useful formatting**")
	if err != nil {
		t.Fatal(err)
	}
	for _, unsafe := range []string{"<script", "<img src=x", "onerror=", `href="javascript:`} {
		if strings.Contains(string(body), unsafe) {
			t.Errorf("Markdown rendered executable content %q: %s", unsafe, body)
		}
	}
	for _, safe := range []string{"<h1>Safe heading</h1>", "<strong>Useful formatting</strong>"} {
		if !strings.Contains(string(body), safe) {
			t.Errorf("Markdown missing %q: %s", safe, body)
		}
	}
}

func TestDetailKeepsCommentsOutOfDescription(t *testing.T) {
	comment := datamodel.Comment{ID: "comment-id", Author: "tester", Ts: "2026-08-01T12:00:00Z", Body: "Discussion-only text"}
	p := pageData{
		View: "tree", URL: "/?item=WEB-1", Token: "test-token", Revision: "test-revision",
		Detail: &datamodel.ShowResult{
			ID: "item-id", Number: "WEB-1", Title: "Editable item", Type: "ticket",
			Body:     codec.JoinComments("Description text", []datamodel.Comment{comment}),
			Comments: []datamodel.CommentView{{ID: comment.ID, Author: comment.Author, Ts: comment.Ts, Text: comment.Body}},
		},
	}
	var out bytes.Buffer
	if err := renderPage(&out, p); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), comment.Body) != 1 || strings.Contains(out.String(), "kira:comment") {
		t.Fatal("comment content was duplicated into the description or editor")
	}
	if !strings.Contains(out.String(), `name="revision" value="test-revision"`) {
		t.Fatal("edit form lacks its revision")
	}
	p.Action = "edit"
	p.Form = url.Values{"title": {"Unsaved <title>"}, "body": {"Unsaved <description>"}, "owner": {""}}
	out.Reset()
	if err := renderPage(&out, p); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`value="Unsaved &lt;title&gt;"`, "Unsaved &lt;description&gt;"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("failed form lost escaped input %q", want)
		}
	}
}

func TestInsightsDistinguishesMissingSamples(t *testing.T) {
	p := pageData{View: "stats", URL: "/?view=stats", Stats: &datamodel.StatsResult{
		Completion: &datamodel.Completion{Done: 1, Total: 4, Pct: .25},
		CycleTime:  &datamodel.Percentiles{},
		LeadTime:   &datamodel.Percentiles{},
	}}
	var out bytes.Buffer
	if err := renderPage(&out, p); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `25<span>%</span>`) {
		t.Fatal("completion fraction was not rendered as a percentage")
	}
	if strings.Contains(out.String(), "0.0<span>days</span>") || !strings.Contains(out.String(), "Complete work to measure cycle time.") {
		t.Fatal("missing samples were displayed as zero days")
	}
}

func TestNavigationClearsCommitAndOpensManagedItems(t *testing.T) {
	p := pageData{View: "tree", URL: "/?view=tree&item=WEB-1&commit=old-sha&q=owner%3DAda"}
	for _, pairs := range [][]string{{"item", "WEB-2"}, {"view", "board"}} {
		u, err := url.Parse(p.Link(pairs...))
		if err != nil {
			t.Fatal(err)
		}
		if u.Query().Get("commit") != "" || u.Query().Get("q") != "owner=Ada" {
			t.Errorf("navigation retained a stale commit or lost filters: %s", u)
		}
	}
	p.View, p.URL = "manage", "/?view=manage"
	u, err := url.Parse(p.Link("item", "WEB-2"))
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("view") != "tree" || u.Query().Get("item") != "WEB-2" {
		t.Fatalf("managed item link did not open the tree: %s", u)
	}
}

func TestCopyPickerUsesSharedFormats(t *testing.T) {
	it := datamodel.ListItem{ID: "immutable-id", Number: "WEB-7", Title: `A <script> & a "title"`}
	cfg := config.Default()
	options := pageData{Config: cfg}.CopyForms(it.ID, it.Number, it.Title)
	if len(options) != len(showfmt.Forms) {
		t.Fatalf("got %d copy formats, want %d", len(options), len(showfmt.Forms))
	}
	for i, form := range showfmt.Forms {
		want, err := showfmt.Format(form, showfmt.Item{ID: it.ID, Number: it.Number, Title: it.Title}, cfg.Workon)
		if err != nil {
			t.Fatal(err)
		}
		if options[i].Name != string(form) || options[i].Value != want || options[i].Label != form.Label() {
			t.Errorf("%s copy option differs from CLI: %+v", form, options[i])
		}
	}
	for view, p := range map[string]pageData{
		"tree": {Config: cfg, Tree: []treeRow{{Item: it}}},
		"board": {Config: cfg, Board: &datamodel.BoardResult{Columns: []datamodel.BoardColumn{
			{State: "todo", Count: 1, Items: []datamodel.ListItem{it}},
		}}},
		"detail": {
			Config:   cfg,
			Detail:   &datamodel.ShowResult{ID: it.ID, Number: it.Number, Title: it.Title},
			Items:    []datamodel.ListItem{{ID: "other-id", Number: "WEB-1", Title: "Inside the filter"}},
			AllItems: []datamodel.ListItem{{ID: "other-id", Number: "WEB-1", Title: "Inside the filter"}},
		},
	} {
		t.Run(view, func(t *testing.T) {
			p.View, p.URL = view, "/?q=inside&item=WEB-7"
			var out bytes.Buffer
			if err := pageTemplate.ExecuteTemplate(&out, view, p); err != nil {
				t.Fatal(err)
			}
			body := out.String()
			if strings.Count(body, `<template class="item-copy">`) != 1 {
				t.Fatal("expected one item copy picker")
			}
			for _, option := range options {
				want := `data-copy="` + template.HTMLEscapeString(option.Value) + `" data-copy-format="` + option.Name + `"`
				if !strings.Contains(body, want) {
					t.Errorf("missing escaped copy format %q", want)
				}
			}
			if strings.Contains(body, "A <script>") {
				t.Fatal("item text was rendered as executable HTML")
			}
			if view != "detail" {
				want := `href="/?item=immutable-id&amp;q=inside"`
				if !strings.Contains(body, want) {
					t.Errorf("item link lost its target or filter: want %q", want)
				}
			}
		})
	}
}

func TestDetailCommitCardsPreserveLinksAndEscapeMetadata(t *testing.T) {
	linked := datamodel.CommitLink{SHA: strings.Repeat("a", 40), Subject: `Linked <script> & "subject"`, Author: "<linked>", Ts: "2026-08-01T12:00:00Z"}
	referenced := datamodel.CommitLink{SHA: strings.Repeat("b", 40), Subject: `Referenced <script> & "subject"`, Author: "<referenced>", Ts: "2026-08-02T12:00:00Z"}
	p := pageData{
		View: "tree", URL: "/?view=tree&item=WEB-7&commit=stale&q=owner%3DAda&_notice=old",
		Detail: &datamodel.ShowResult{
			ID: "immutable-id", Number: "WEB-7", Title: "Outside the filter",
			LinkedCommits: []datamodel.CommitLink{linked}, ReferencedBy: []datamodel.CommitLink{referenced},
		},
	}
	var out bytes.Buffer
	if err := pageTemplate.ExecuteTemplate(&out, "detail", p); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	if strings.Count(body, `<article class="commit">`) != 2 {
		t.Fatal("expected linked and referenced commit cards")
	}
	for _, commit := range []datamodel.CommitLink{linked, referenced} {
		query := url.Values{"view": {"tree"}, "item": {p.Detail.ID}, "commit": {commit.SHA}, "q": {"owner=Ada"}}
		for _, want := range []string{
			`href="` + template.HTMLEscapeString("/?"+query.Encode()) + `"`,
			`<code>` + commit.SHA[:8] + `</code> View patch`,
			`data-copy="` + commit.SHA + `"`,
			`<p>` + template.HTMLEscapeString(commit.Subject) + `</p>`,
			`<small>` + template.HTMLEscapeString(commit.Author) + ` · ` + displayTime(commit.Ts) + `</small>`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("commit card missing %q", want)
			}
		}
	}
	if strings.Contains(body, "<script>") || strings.Contains(body, "<linked>") || strings.Contains(body, "<referenced>") {
		t.Fatal("commit metadata was rendered as executable HTML")
	}
}

func TestCommitPatchEscapesSourceAndClassifiesChanges(t *testing.T) {
	p := pageData{View: "tree", URL: "/?item=WEB-1&commit=abc", Commit: &commitView{SHA: "abc", Text: "diff --git a/file b/file\n-old\n+<script>alert(1)</script>\n context"}}
	var out bytes.Buffer
	if err := renderPage(&out, p); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`class="diff-header"`, `class="diff-remove">-old`, `class="diff-add">&#43;&lt;script&gt;alert(1)&lt;/script&gt;`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("commit patch missing safely rendered line %q", want)
		}
	}
	if strings.Contains(out.String(), "+<script>") {
		t.Fatal("commit source was rendered as executable HTML")
	}
}

func TestFailedNativeFormsRetainOptionsAndIdentifyTarget(t *testing.T) {
	p := pageData{
		View: "tree", URL: "/?item=WEB-2", Selected: "WEB-2", Action: "workon", Error: "branch switch failed",
		Form:   url.Values{"id": {"WEB-2"}, "no_move": {"true"}},
		Detail: &datamodel.ShowResult{ID: "item-b", Number: "WEB-2", Title: "Outside the filter"},
		Items:  []datamodel.ListItem{{ID: "item-a", Number: "WEB-1", Title: "Visible item"}},
	}
	var out bytes.Buffer
	if err := renderPage(&out, p); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-failed-action="workon" data-failed-id="item-b"`, `name="no_move" value="true" checked`, `name="selected" value="item-b"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("failed native form missing %q", want)
		}
	}
	p.View, p.URL, p.Action, p.Error = "manage", "/?view=manage", "sprint-close", "rollover failed"
	p.Detail = nil
	p.Form = url.Values{"key": {"current"}, "move_to": {"next"}}
	p.Sprints = []datamodel.SprintListRow{
		{SprintView: datamodel.SprintView{Key: "current", Name: "Current"}},
		{SprintView: datamodel.SprintView{Key: "next", Name: "Next"}},
	}
	out.Reset()
	if err := renderPage(&out, p); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `<option value="next" selected>Move to Next</option>`) {
		t.Fatal("failed sprint close lost its rollover target")
	}
}

func TestSuccessfulBulkDoesNotSetFailedAction(t *testing.T) {
	p := pageData{View: "tree", URL: "/", Action: "bulk", Message: "Updated items"}
	var out bytes.Buffer
	if err := renderPage(&out, p); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `data-failed-action=""`) {
		t.Fatal("successful bulk action was marked as a failed form")
	}
}

func TestCommandErrorPreservesRedirectedTarget(t *testing.T) {
	p := pageData{
		View: "tree", URL: "/?item=WEB-2", Selected: "WEB-2", Action: "command", Error: "comment failed",
		Form:   url.Values{"selected": {"item-b"}, "line": {`comment . -m "Retry text"`}},
		Detail: &datamodel.ShowResult{ID: "item-b", Number: "WEB-2", Title: "Command target"},
		Items:  []datamodel.ListItem{{ID: "item-a", Number: "WEB-1", Title: "Previously open"}},
	}
	var out bytes.Buffer
	if err := renderPage(&out, p); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-selected="WEB-2"`, `name="selected" value="item-b"`, `name="line" value="comment . -m &#34;Retry text&#34;"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("command retry lost redirected target or input: missing %q", want)
		}
	}
}
