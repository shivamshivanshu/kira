package query_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/query"
)

func evalFixture() (items []*datamodel.Item, opts query.Options, cfg *datamodel.Config) {
	items, cfg = fixture()
	snap := id.Snapshot{Key: "KIRA"}
	for _, it := range items {
		snap.Items = append(snap.Items, id.Item{ULID: it.ID, Number: it.Number, Aliases: it.Aliases})
	}
	opts = query.Options{Resolver: id.NewResolver(snap), Priorities: cfg.Priorities.Values}
	return items, opts, cfg
}

func matchNums(t *testing.T, expr string, items []*datamodel.Item, opts query.Options, cfg *datamodel.Config) []string {
	t.Helper()
	c, err := query.Compile(expr, opts)
	if err != nil {
		t.Fatalf("Compile(%q): %v", expr, err)
	}
	var out []string
	for _, it := range items {
		if c.Pred(it, cfg) {
			out = append(out, it.Number)
		}
	}
	sort.Strings(out)
	return out
}

func TestEval(t *testing.T) {
	items, opts, cfg := evalFixture()
	tests := []struct {
		expr string
		want []string
	}{
		{"state=IN_PROGRESS", []string{"KIRA-1"}},
		{"state!=IN_PROGRESS", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"owner=shivam", []string{"KIRA-1", "KIRA-3"}},
		{"owner!=shivam", []string{"KIRA-100", "KIRA-2"}},
		{"reporter=shivam", []string{"KIRA-2"}},
		{"label=bug", []string{"KIRA-1", "KIRA-3"}},
		{"label=perf", []string{"KIRA-2", "KIRA-3"}},
		{"label=bug AND label=perf", []string{"KIRA-3"}},
		{"label!=bug", []string{"KIRA-100", "KIRA-2"}},
		{"category=doing", []string{"KIRA-1", "KIRA-100"}},
		{"category=done", []string{"KIRA-3"}},
		{"type=epic", []string{"KIRA-100"}},
		{"type=ticket", []string{"KIRA-1", "KIRA-2", "KIRA-3"}},
		{"subtype=bug", []string{"KIRA-1"}},
		{"resolution=done", []string{"KIRA-3"}},
		{"priority=P1", []string{"KIRA-1"}},
		{"rank=aam", []string{"KIRA-1"}},
		{"board=KIRA", []string{"KIRA-1", "KIRA-100", "KIRA-2", "KIRA-3"}},
		{"board=kira", []string{"KIRA-1", "KIRA-100", "KIRA-2", "KIRA-3"}},
		{"board!=KIRA", nil},
		{"sprint=2026-S14", []string{"KIRA-1"}},
		{"epic=KIRA-100", []string{"KIRA-1"}},
		{"NOT epic=KIRA-100", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"created>2026-06-15", []string{"KIRA-1", "KIRA-100", "KIRA-2"}},
		{"created<2026-07-01", []string{"KIRA-3"}},
		{"created>=2026-07-01", []string{"KIRA-1", "KIRA-100", "KIRA-2"}},
		{"updated<=2026-07-01", []string{"KIRA-100", "KIRA-3"}},
		{"due<2026-07-10", []string{"KIRA-2"}},
		{"due>=2026-07-20", []string{"KIRA-1"}},
		{"due=2026-07-01", []string{"KIRA-2"}},
		{"due!=2026-07-01", []string{"KIRA-1"}},
		{"estimate>3", []string{"KIRA-2"}},
		{"estimate<=3", []string{"KIRA-1"}},
		{"estimate=5", []string{"KIRA-2"}},
		{"priority<=P1", []string{"KIRA-1", "KIRA-2"}},
		{"priority<P1", []string{"KIRA-2"}},
		{"priority>=P1", []string{"KIRA-1"}},
		{"priority>P0", []string{"KIRA-1"}},
		{"priority>P1", nil},
		{"priority IN (P0,P1)", []string{"KIRA-1", "KIRA-2"}},
		{"owner IN (alice, bob)", []string{"KIRA-2"}},
		{"label IN (perf, missing)", []string{"KIRA-2", "KIRA-3"}},
		{"epic IN (KIRA-100)", []string{"KIRA-1"}},
		{"NOT priority IN (P0,P1)", []string{"KIRA-100", "KIRA-3"}},
		{"owner IS EMPTY", []string{"KIRA-100"}},
		{"owner IS NOT EMPTY", []string{"KIRA-1", "KIRA-2", "KIRA-3"}},
		{"rank IS EMPTY", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"rank IS NOT EMPTY", []string{"KIRA-1"}},
		{"resolution IS NOT EMPTY", []string{"KIRA-3"}},
		{"sprint IS EMPTY", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"due IS EMPTY", []string{"KIRA-100", "KIRA-3"}},
		{"estimate IS EMPTY", []string{"KIRA-100", "KIRA-3"}},
		{"epic IS EMPTY", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"label IS EMPTY", []string{"KIRA-100"}},
		{"blocked_by IS NOT EMPTY", []string{"KIRA-1"}},
		{"links IS EMPTY", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"links IS NOT EMPTY", []string{"KIRA-1"}},
		{"blocked_by=KIRA-2", []string{"KIRA-1"}},
		{"links=KIRA-3", []string{"KIRA-1"}},
		{"race", []string{"KIRA-1"}},
		{"RACE", []string{"KIRA-1"}},
		{`"fix race"`, []string{"KIRA-1"}},
		{"owner=shivam AND label=perf", []string{"KIRA-3"}},
		{"owner=alice OR label=bug", []string{"KIRA-1", "KIRA-2", "KIRA-3"}},
		{"category=doing AND NOT owner=alice", []string{"KIRA-1", "KIRA-100"}},
		{"epic=KIRA-100 AND created>2026-07-01", []string{"KIRA-1"}},
		{"priority IN (P0,P1) AND label=bug", []string{"KIRA-1"}},
	}
	for _, tc := range tests {
		got := matchNums(t, tc.expr, items, opts, cfg)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%q matched %v, want %v", tc.expr, got, tc.want)
		}
	}
}

func TestEvalMe(t *testing.T) {
	items, opts, cfg := evalFixture()
	opts.Me = "shivam"
	tests := []struct {
		expr string
		want string
	}{
		{"owner=@me", "KIRA-1,KIRA-3"},
		{"owner!=@me", "KIRA-100,KIRA-2"},
		{"reporter=@me", "KIRA-2"},
		{"owner IN (@me)", "KIRA-1,KIRA-3"},
	}
	for _, tc := range tests {
		if got := matchNums(t, tc.expr, items, opts, cfg); strings.Join(got, ",") != tc.want {
			t.Errorf("%q matched %v, want %s", tc.expr, got, tc.want)
		}
	}
}

func TestEvalSprintActive(t *testing.T) {
	items, opts, cfg := evalFixture()

	opts.ActiveSprint = "2026-S14"
	for _, expr := range []string{"sprint=active", "sprint IN (active, 2026-S99)"} {
		if got := matchNums(t, expr, items, opts, cfg); strings.Join(got, ",") != "KIRA-1" {
			t.Errorf("%q with active sprint matched %v, want [KIRA-1]", expr, got)
		}
	}
	c, err := query.Compile("sprint=active", opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Notes) != 0 {
		t.Errorf("notes with an active sprint = %v, want none", c.Notes)
	}

	opts.ActiveSprint = ""
	if got := matchNums(t, "sprint=active", items, opts, cfg); got != nil {
		t.Errorf("sprint=active with no active sprint matched %v, want none", got)
	}
	if got := matchNums(t, "sprint!=active", items, opts, cfg); len(got) != len(items) {
		t.Errorf("sprint!=active with no active sprint matched %v, want all", got)
	}
	c, err = query.Compile("sprint=active", opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Notes) != 1 || c.Notes[0] != datamodel.WarnNoActiveSprint {
		t.Errorf("notes = %v, want the no-active-sprint note once", c.Notes)
	}
}

func TestEvalInEstimate(t *testing.T) {
	items, opts, cfg := evalFixture()
	zero := &datamodel.Item{
		ID: id.Mint().String(), Number: "KIRA-0", Type: datamodel.TypeTicket, Title: "Zero estimate",
		State: "TODO", Estimate: f64p(0), Created: "2026-07-05T00:00:00Z", Updated: "2026-07-05T00:00:00Z",
	}
	items = append(items, zero)

	if got := matchNums(t, "estimate IN (3)", items, opts, cfg); strings.Join(got, ",") != "KIRA-1" {
		t.Errorf("estimate IN (3) matched %v, want [KIRA-1] (only estimate==3, never estimate==0)", got)
	}
}

func TestEvalInCreatedDate(t *testing.T) {
	items, opts, cfg := evalFixture()
	if got := matchNums(t, "created IN (2026-07-01)", items, opts, cfg); strings.Join(got, ",") != "KIRA-100" {
		t.Errorf("created IN (2026-07-01) matched %v, want [KIRA-100]", got)
	}
}

func TestCompileErrors(t *testing.T) {
	_, opts, _ := evalFixture()
	noPrio := opts
	noPrio.Priorities = nil
	tests := []struct {
		name string
		expr string
		opts query.Options
		pos  int
	}{
		{"unresolved epic", "epic=KIRA-999", opts, 5},
		{"unresolved blocked_by", "blocked_by=KIRA-999", opts, 11},
		{"unresolved links in IN", "links IN (KIRA-999)", opts, 10},
		{"ranked compare needs priorities", "priority<=P1", noPrio, 8},
		{"unknown priority literal", "priority<=P9", opts, 10},
		{"order by priority needs priorities", "a ORDER BY priority", noPrio, 11},
		{"owner @me without identity", "owner=@me", opts, 6},
		{"reporter @me without identity", "reporter=@me", opts, 9},
	}
	for _, tc := range tests {
		_, err := query.Compile(tc.expr, tc.opts)
		qe, ok := err.(*query.Error)
		if !ok {
			t.Fatalf("%s: Compile(%q) err = %v, want *Error", tc.name, tc.expr, err)
		}
		if qe.Pos != tc.pos {
			t.Errorf("%s: pos = %d, want %d (%s)", tc.name, qe.Pos, tc.pos, qe.Msg)
		}
	}
	if _, err := query.Compile("priority=P1", noPrio); err != nil {
		t.Errorf("priority=P1 must stay legal without a priority vocabulary: %v", err)
	}
}

func TestEvalBlocked(t *testing.T) {
	items, opts, cfg := evalFixture()
	waitsOnDone := &datamodel.Item{
		ID: id.Mint().String(), Number: "KIRA-4", Type: datamodel.TypeTicket, Title: "waits on done",
		State: "TODO", BlockedBy: []string{items[3].ID},
		Created: "2026-07-05T00:00:00Z", Updated: "2026-07-05T00:00:00Z",
	}
	waitsOnGhost := &datamodel.Item{
		ID: id.Mint().String(), Number: "KIRA-5", Type: datamodel.TypeTicket, Title: "waits on ghost",
		State: "TODO", BlockedBy: []string{"01ZZZZZZZZZZZZZZZZZZZZZZZZZ"},
		Created: "2026-07-05T00:00:00Z", Updated: "2026-07-05T00:00:00Z",
	}
	items = append(items, waitsOnDone, waitsOnGhost)
	opts.Items = items

	notBlocked := []string{"KIRA-100", "KIRA-2", "KIRA-3", "KIRA-4", "KIRA-5"}
	tests := []struct {
		expr string
		want []string
	}{
		{"blocked", []string{"KIRA-1"}},
		{"blocked = true", []string{"KIRA-1"}},
		{"blocked != false", []string{"KIRA-1"}},
		{"NOT blocked", notBlocked},
		{"blocked = false", notBlocked},
		{"blocked AND state=IN_PROGRESS", []string{"KIRA-1"}},
	}
	for _, tc := range tests {
		got := matchNums(t, tc.expr, items, opts, cfg)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%q matched %v, want %v", tc.expr, got, tc.want)
		}
	}
}

func TestEvalActivity(t *testing.T) {
	items, opts, cfg := evalFixture()
	items[0].Activity = items[0].Updated
	items[1].Activity = "2026-07-15T00:00:00Z"
	items[2].Activity = items[2].Updated
	items[3].Activity = items[3].Updated

	tests := []struct {
		expr string
		want []string
	}{
		{"activity>2026-07-12", []string{"KIRA-1"}},
		{"activity<2026-07-05", []string{"KIRA-100", "KIRA-3"}},
		{"activity=2026-07-10", []string{"KIRA-2"}},
	}
	for _, tc := range tests {
		got := matchNums(t, tc.expr, items, opts, cfg)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%q matched %v, want %v", tc.expr, got, tc.want)
		}
	}

	c, err := query.Compile("state=TODO ORDER BY activity", opts)
	if err != nil {
		t.Fatalf("ORDER BY activity: %v", err)
	}
	if c.Order == nil || c.Order.Field != "activity" {
		t.Errorf("ORDER BY activity did not parse into an activity order: %+v", c.Order)
	}
}

func TestBlockedActivitySyntaxErrors(t *testing.T) {
	_, opts, _ := evalFixture()
	for _, expr := range []string{"state=TODO ORDER BY blocked", "blocked IS EMPTY", "blocked = maybe", "blocked IN (true)"} {
		if _, err := query.Compile(expr, opts); err == nil {
			t.Errorf("Compile(%q) should error", expr)
		}
	}
}

func TestMatch(t *testing.T) {
	items, opts, cfg := evalFixture()
	pred, err := query.Match("owner", "shivam", opts)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	var n int
	for _, it := range items {
		if pred(it, cfg) {
			n++
		}
	}
	if n != 2 {
		t.Errorf("Match owner=shivam matched %d, want 2", n)
	}
	if _, err := query.Match("epic", "KIRA-100", opts); err != nil {
		t.Errorf("Match epic=KIRA-100: %v", err)
	}
	active := opts
	active.ActiveSprint = "2026-S14"
	pred, err = query.Match("sprint", "active", active)
	if err != nil {
		t.Fatalf("Match sprint=active: %v", err)
	}
	n = 0
	for _, it := range items {
		if pred(it, cfg) {
			n++
		}
	}
	if n != 1 {
		t.Errorf("Match sprint=active matched %d, want 1", n)
	}
	if _, err := query.Match("created", "2026-01-01", opts); err == nil {
		t.Errorf("Match on a date field should be rejected")
	}
	if _, err := query.Match("bogus", "x", opts); err == nil {
		t.Errorf("Match on an unknown field should be rejected")
	}
}
