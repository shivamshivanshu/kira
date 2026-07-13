package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func listFixture(t *testing.T) (*core.Store, *datamodel.Config) {
	t.Helper()
	root := testutil.InitGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := core.Discover(root)
	cfg, _ := s.Config()
	cfg.Filters = map[string]string{
		"mine":    "owner=shivam",
		"groomed": "rank IS NOT EMPTY ORDER BY rank",
	}
	cfg.Sprints = []datamodel.Sprint{{Key: "2026-S14", Name: "Sprint 14", Start: "2026-07-13", End: "2026-07-26"}}
	for _, c := range []core.CreateOpts{
		{Type: datamodel.TypeTicket, Title: "alpha", Owner: "shivam", Priority: "P2", NoEdit: true},
		{Type: datamodel.TypeTicket, Title: "beta", Owner: "alice", Priority: "P0", Sprint: "2026-S14", NoEdit: true},
		{Type: datamodel.TypeTicket, Title: "gamma", Owner: "shivam", Rank: "mm", Due: "2026-07-20", NoEdit: true},
	} {
		if _, err := s.Create(cfg, c); err != nil {
			t.Fatalf("create %s: %v", c.Title, err)
		}
	}
	return s, cfg
}

func listNumbers(res *datamodel.ListResult) string {
	nums := make([]string, len(res.Items))
	for i, it := range res.Items {
		nums[i] = it.Number
	}
	return strings.Join(nums, ",")
}

func TestListDefaultPrecedenceAndOrderBy(t *testing.T) {
	s, cfg := listFixture(t)

	res, err := s.List(cfg, core.ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got, want := listNumbers(res), "KIRA-3,KIRA-2,KIRA-1"; got != want {
		t.Errorf("default order = %s, want %s", got, want)
	}

	res, err = s.List(cfg, core.ListOpts{Query: "type=ticket ORDER BY priority"})
	if err != nil {
		t.Fatalf("List order by priority: %v", err)
	}
	if got, want := listNumbers(res), "KIRA-2,KIRA-1,KIRA-3"; got != want {
		t.Errorf("ORDER BY priority = %s, want %s", got, want)
	}
	res, err = s.List(cfg, core.ListOpts{Query: "type=ticket ORDER BY due desc"})
	if err != nil {
		t.Fatalf("List order by due desc: %v", err)
	}
	if got, want := listNumbers(res), "KIRA-3,KIRA-2,KIRA-1"; got != want {
		t.Errorf("ORDER BY due desc = %s, want %s", got, want)
	}
}

func TestListFlagFilters(t *testing.T) {
	s, cfg := listFixture(t)

	res, err := s.List(cfg, core.ListOpts{Priority: "P0"})
	if err != nil {
		t.Fatalf("List --priority: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-2" {
		t.Errorf("--priority P0 = %s, want KIRA-2", got)
	}

	res, err = s.List(cfg, core.ListOpts{Sprint: "2026-S14"})
	if err != nil {
		t.Fatalf("List --sprint: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-2" {
		t.Errorf("--sprint 2026-S14 = %s, want KIRA-2", got)
	}

	res, err = s.List(cfg, core.ListOpts{Sprint: "active"})
	if err != nil {
		t.Fatalf("List --sprint active: %v", err)
	}
	if res.Count != 0 {
		t.Errorf("--sprint active with no active sprint matched %d, want 0", res.Count)
	}
	if len(res.StderrNotes) != 1 {
		t.Errorf("notes = %v, want the no-active-sprint note", res.StderrNotes)
	}
}

func TestListSavedFilters(t *testing.T) {
	s, cfg := listFixture(t)

	res, err := s.List(cfg, core.ListOpts{Filter: "mine"})
	if err != nil {
		t.Fatalf("List --filter mine: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-3,KIRA-1" {
		t.Errorf("--filter mine = %s, want KIRA-3,KIRA-1", got)
	}

	res, err = s.List(cfg, core.ListOpts{Filter: "mine", Priority: "P2"})
	if err != nil {
		t.Fatalf("List --filter mine --priority: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-1" {
		t.Errorf("--filter mine --priority P2 = %s, want KIRA-1", got)
	}

	res, err = s.List(cfg, core.ListOpts{Filter: "groomed"})
	if err != nil {
		t.Fatalf("List --filter groomed: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-3" {
		t.Errorf("--filter groomed = %s, want KIRA-3", got)
	}

	if _, err := s.List(cfg, core.ListOpts{Filter: "groomed", Query: "a ORDER BY due"}); err == nil {
		t.Error("filter+query each with ORDER BY should be rejected")
	}

	_, err = s.List(cfg, core.ListOpts{Filter: "nope"})
	var ce *errx.Error
	if !errors.As(err, &ce) || ce.Code != errx.ExitUser {
		t.Fatalf("unknown filter err = %v, want exit-1 user error", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "groomed") || !strings.Contains(msg, "mine") {
		t.Errorf("unknown filter message %q should list configured names", msg)
	}
}

func TestFiltersView(t *testing.T) {
	cfg := config.Default()
	cfg.Filters = map[string]string{"b": "owner=x", "a": "label=y"}
	res := core.Filters(cfg)
	if len(res.Filters) != 2 || res.Filters[0].Name != "a" || res.Filters[1].Name != "b" {
		t.Errorf("Filters = %+v, want name-sorted [a b]", res.Filters)
	}
}
