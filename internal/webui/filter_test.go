package webui

import (
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

func TestDashboardDropdownFiltersCombineWithQueries(t *testing.T) {
	d := newDashboardTest(t, Options{})
	want := d.create(t, core.CreateOpts{Title: "Target item", Owner: "tester", Priority: "P1", Labels: []string{"keep"}})
	d.create(t, core.CreateOpts{Title: "Target other owner", Owner: "someone", Priority: "P1", Labels: []string{"keep"}})
	d.create(t, core.CreateOpts{Title: "Target other priority", Owner: "tester", Priority: "P2", Labels: []string{"keep"}})
	otherLabel := d.create(t, core.CreateOpts{Title: "Target other label", Owner: "tester", Priority: "P1", Labels: []string{"skip"}})
	d.create(t, core.CreateOpts{Title: "Different title", Owner: "tester", Priority: "P1", Labels: []string{"keep"}})
	active := d.create(t, core.CreateOpts{Title: "Target active", Owner: "tester", Priority: "P1", Labels: []string{"keep"}})
	if _, err := d.store.Move(d.cfg, active.ID, "IN_PROGRESS", core.MoveOpts{}); err != nil {
		t.Fatal(err)
	}
	rowID := regexp.MustCompile(`data-nav-item[^>]*data-item="([^"]+)"`)
	for _, view := range []string{"tree", "board"} {
		for _, tc := range []struct {
			name, field, value string
			want               []string
		}{
			{name: "combined", want: []string{want.ID}},
			{name: "me", field: "owner", value: "@me", want: []string{want.ID}},
			{name: "other label", field: "label", value: "skip", want: []string{otherLabel.ID}},
			{name: "saved filter contradiction", field: "state", value: "IN_PROGRESS"},
			{name: "unknown state", field: "state", value: "NOT_A_STATE"},
		} {
			t.Run(view+"/"+tc.name, func(t *testing.T) {
				q := url.Values{
					"view": {view}, "state": {"TODO"}, "owner": {"tester"},
					"priority": {"P1"}, "label": {"keep"}, "filter": {"next"}, "q": {`Target OR "Never matches"`},
				}
				if tc.field != "" {
					q.Set(tc.field, tc.value)
				}
				response := d.get("/?" + q.Encode())
				if response.Code != http.StatusOK {
					t.Fatalf("filtered dashboard: %d\n%s", response.Code, response.Body.String())
				}
				var got []string
				for _, match := range rowID.FindAllStringSubmatch(response.Body.String(), -1) {
					got = append(got, match[1])
				}
				if !slices.Equal(got, tc.want) {
					t.Fatalf("visible items = %v, want %v", got, tc.want)
				}
			})
		}
	}
}

func TestDropdownOptionsIncludeConfiguredAndExistingValues(t *testing.T) {
	p := pageData{
		Type: "ticket", State: "CUSTOM", Owner: "Other", Label: "custom",
		Config: &datamodel.Config{
			Workflows: map[string]datamodel.Workflow{
				"ticket": {States: []datamodel.State{{Key: "TODO"}, {Key: "IN_PROGRESS"}}},
				"epic":   {States: []datamodel.State{{Key: "PLANNED"}, {Key: "TODO"}}},
			},
			People: datamodel.People{Known: []datamodel.Person{{Name: "Ada"}, {Name: "Bob"}}},
			Labels: datamodel.Vocab{Known: []string{"bug", "feature"}},
		},
		AllItems: []datamodel.ListItem{
			{Owner: ptr.To("Ada"), Labels: []string{"bug", "ops"}},
			{Owner: ptr.To("Zoe"), Labels: []string{"ops"}},
			{},
		},
	}
	if got, want := p.StateOptions(), []string{"CUSTOM", "IN_PROGRESS", "TODO"}; !slices.Equal(got, want) {
		t.Errorf("state options = %v, want %v", got, want)
	}
	p.Type = ""
	if got, want := p.StateOptions(), []string{"CUSTOM", "IN_PROGRESS", "PLANNED", "TODO"}; !slices.Equal(got, want) {
		t.Errorf("all state options = %v, want %v", got, want)
	}
	if got, want := p.OwnerOptions(), []string{"Ada", "Bob", "Other", "Zoe"}; !slices.Equal(got, want) {
		t.Errorf("owner options = %v, want %v", got, want)
	}
	p.Owner = "@me"
	if slices.Contains(p.OwnerOptions(), "@me") {
		t.Error("owner options duplicate the dedicated @me option")
	}
	if got, want := p.LabelOptions(), []string{"bug", "custom", "feature", "ops"}; !slices.Equal(got, want) {
		t.Errorf("label options = %v, want %v", got, want)
	}
}
