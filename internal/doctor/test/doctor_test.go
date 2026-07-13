package doctor_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/id"
)

const (
	ulidA = "01HAAAAAAAAAAAAAAAAAAAAAAA"
	ulidB = "01HBBBBBBBBBBBBBBBBBBBBBBB"
	ulidC = "01HCCCCCCCCCCCCCCCCCCCCCCC"
)

func strp(s string) *string { return &s }

func resolver(items ...*datamodel.Item) *id.Resolver {
	snap := id.Snapshot{Key: "KIRA"}
	for _, it := range items {
		snap.Items = append(snap.Items, id.Item{ULID: it.ID, Number: it.Number, Aliases: it.Aliases})
	}
	return id.NewResolver(snap)
}

func valid(ulid, number string) *datamodel.Item {
	return &datamodel.Item{
		ID: ulid, Number: number, Aliases: []string{}, Type: datamodel.TypeTicket,
		Title: "t", State: "TODO", Labels: []string{}, BlockedBy: []string{},
		Created: "2026-07-01T00:00:00Z", Updated: "2026-07-01T00:00:00Z",
	}
}

func classes(findings []doctor.Finding) map[doctor.Class]doctor.Severity {
	out := map[doctor.Class]doctor.Severity{}
	for _, f := range findings {
		out[f.Class] = f.Severity
	}
	return out
}

func hasClass(findings []doctor.Finding, class doctor.Class) bool {
	for _, f := range findings {
		if f.Class == class {
			return true
		}
	}
	return false
}

func TestLintMalformedFrontmatter(t *testing.T) {
	it, parsed, findings := doctor.Lint("no fence here\n")
	if parsed {
		t.Fatal("expected parsed=false for content without a frontmatter fence")
	}
	if it != nil {
		t.Fatalf("expected nil item, got %+v", it)
	}
	if !hasClass(findings, doctor.ClassSchema) {
		t.Fatalf("expected a schema finding, got %+v", findings)
	}
}

func TestLintUnknownFieldWarns(t *testing.T) {
	content := "---\nid: " + ulidA + "\nnumber: KIRA-1\naliases: []\ntype: ticket\ntitle: t\n" +
		"state: TODO\nlabels: []\nblocked_by: []\nepic: null\nbogus_key: 1\n" +
		"created: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n## Description\n"
	_, parsed, findings := doctor.Lint(content)
	if !parsed {
		t.Fatalf("expected a well-formed item to parse, got findings %+v", findings)
	}
	for _, f := range findings {
		if f.Class == doctor.ClassSchema && f.Field == "bogus_key" && f.Severity == doctor.SeverityWarning {
			return
		}
	}
	t.Fatalf("expected an unknown-field warning for bogus_key, got %+v", findings)
}

func TestLintMalformedComment(t *testing.T) {
	body := "## Comments\n<!-- kira:comment id=X author=y ts=bogus -->\nunterminated body\n"
	_, _, findings := doctor.Lint("---\nid: " + ulidA + "\nnumber: KIRA-1\naliases: []\ntype: ticket\n" +
		"title: t\nstate: TODO\nlabels: []\nblocked_by: []\nepic: null\n" +
		"created: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n" + body)
	var commentFindings int
	for _, f := range findings {
		if f.Class == doctor.ClassComment {
			commentFindings++
		}
	}
	if commentFindings == 0 {
		t.Fatalf("expected comment findings for bad ts + unterminated block")
	}
}

func TestCheckCleanItemHasNoFindings(t *testing.T) {
	it := valid(ulidA, "KIRA-1")
	if f := doctor.Check(config.Default(), resolver(it), it); len(f) != 0 {
		t.Fatalf("expected no findings for a clean item, got %+v", f)
	}
}

func TestCheckUnknownStateAndSprintAndDue(t *testing.T) {
	it := valid(ulidA, "KIRA-1")
	it.State = "NOPE"
	it.Sprint = strp("2099-S1")
	it.Due = strp("not-a-date")
	findings := doctor.Check(config.Default(), resolver(it), it)
	cl := classes(findings)
	if cl[doctor.ClassState] != doctor.SeverityError {
		t.Errorf("expected state error, got %+v", findings)
	}
	if !hasClass(findings, doctor.ClassSchema) {
		t.Errorf("expected schema findings for sprint/due, got %+v", findings)
	}
}

func TestCheckVocabStrictVsWarn(t *testing.T) {
	it := valid(ulidA, "KIRA-1")
	it.Labels = []string{"not-a-label"}

	warn := doctor.Check(config.Default(), resolver(it), it)
	if classes(warn)[doctor.ClassVocab] != doctor.SeverityWarning {
		t.Errorf("expected vocab warning under non-strict labels, got %+v", warn)
	}

	strict := config.Default()
	strict.Labels.Strict = true
	got := doctor.Check(strict, resolver(it), it)
	if classes(got)[doctor.ClassVocab] != doctor.SeverityError {
		t.Errorf("expected vocab error under strict labels, got %+v", got)
	}
}

func TestCheckEnumVocabAlwaysErrors(t *testing.T) {
	it := valid(ulidA, "KIRA-1")
	it.Priority = strp("P9")
	cfg := config.Default()
	cfg.Labels.Strict = false
	findings := doctor.Check(cfg, resolver(it), it)
	if classes(findings)[doctor.ClassVocab] != doctor.SeverityError {
		t.Fatalf("priority/subtype/resolution are always-strict; expected an error even under non-strict labels, got %+v", findings)
	}
}

func TestCheckDanglingAndSelfRef(t *testing.T) {
	it := valid(ulidA, "KIRA-1")
	it.Epic = strp(ulidB)
	it.BlockedBy = []string{ulidA}
	findings := doctor.Check(config.Default(), resolver(it), it)
	var dangling, selfRef bool
	for _, f := range findings {
		if f.Class != doctor.ClassRef {
			continue
		}
		if strings.Contains(f.Message, "resolves to no item") {
			dangling = true
		}
		if strings.Contains(f.Message, "the item itself") {
			selfRef = true
		}
	}
	if !dangling || !selfRef {
		t.Fatalf("expected dangling epic + self blocked_by, got %+v", findings)
	}
}

func TestCollisionsLiveLive(t *testing.T) {
	a := valid(ulidA, "KIRA-1")
	b := valid(ulidB, "KIRA-1")
	findings := doctor.Collisions([]*datamodel.Item{a, b})
	if len(findings) != 1 {
		t.Fatalf("expected exactly one collision, got %+v", findings)
	}
	c := findings[0].Collision
	if c == nil || c.Kind != doctor.CollisionLiveLive {
		t.Fatalf("expected live-live collision payload, got %+v", findings[0])
	}
	if c.Keep != ulidA {
		t.Errorf("expected earlier ULID %s to be the keeper, got %s", ulidA, c.Keep)
	}
	if !slices.Equal(c.LiveIDs, []string{ulidA, ulidB}) || len(c.AliasIDs) != 0 {
		t.Errorf("expected both live holders and no alias holders, got live=%v alias=%v", c.LiveIDs, c.AliasIDs)
	}
}

func TestCollisionsLiveAlias(t *testing.T) {
	a := valid(ulidA, "KIRA-5")
	b := valid(ulidB, "KIRA-9")
	b.Aliases = []string{"KIRA-5"}
	findings := doctor.Collisions([]*datamodel.Item{a, b})
	if len(findings) != 1 || findings[0].Collision.Kind != doctor.CollisionLiveAlias {
		t.Fatalf("expected one live-alias collision, got %+v", findings)
	}
	c := findings[0].Collision
	if c.Keep != ulidA {
		t.Errorf("live holder %s should be the keeper, got %s", ulidA, c.Keep)
	}
	if !slices.Equal(c.LiveIDs, []string{ulidA}) || !slices.Equal(c.AliasIDs, []string{ulidB}) {
		t.Errorf("expected live=[%s] alias=[%s], got live=%v alias=%v", ulidA, ulidB, c.LiveIDs, c.AliasIDs)
	}
}

func TestCollisionsNoneWhenAliasIsOwnRetiredNumber(t *testing.T) {
	a := valid(ulidA, "KIRA-2")
	a.Aliases = []string{"KIRA-1"}
	b := valid(ulidB, "KIRA-3")
	if f := doctor.Collisions([]*datamodel.Item{a, b}); len(f) != 0 {
		t.Fatalf("expected no collision, got %+v", f)
	}
}

func TestEpicCycle(t *testing.T) {
	a := valid(ulidA, "KIRA-1")
	b := valid(ulidB, "KIRA-2")
	a.Epic = strp(ulidB)
	b.Epic = strp(ulidA)
	items := []*datamodel.Item{a, b}
	findings := doctor.EpicCycles(items, resolver(items...))
	if len(findings) != 2 {
		t.Fatalf("expected a finding per cycle member, got %+v", findings)
	}
	for _, f := range findings {
		if f.Class != doctor.ClassCycle || f.Severity != doctor.SeverityError {
			t.Fatalf("unexpected finding %+v", f)
		}
	}
}

func TestEpicChainAcyclic(t *testing.T) {
	child := valid(ulidA, "KIRA-1")
	parent := valid(ulidB, "KIRA-2")
	child.Epic = strp(ulidB)
	items := []*datamodel.Item{child, parent}
	if f := doctor.EpicCycles(items, resolver(items...)); len(f) != 0 {
		t.Fatalf("expected no cycle in a linear chain, got %+v", f)
	}
}

func TestRunAggregatesAndFlagsIdentity(t *testing.T) {
	good := "---\nid: " + ulidA + "\nnumber: KIRA-1\naliases: []\ntype: ticket\ntitle: t\n" +
		"state: TODO\nlabels: []\nblocked_by: []\nepic: null\n" +
		"created: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n## Description\n"
	mismatch := "---\nid: " + ulidC + "\nnumber: KIRA-2\naliases: []\ntype: ticket\ntitle: t\n" +
		"state: TODO\nlabels: []\nblocked_by: []\nepic: null\n" +
		"created: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n## Description\n"
	files := []doctor.File{
		{Path: ulidA + ".md", Content: good},
		{Path: ulidB + ".md", Content: mismatch},
		{Path: ulidC + ".md", Content: "garbage without a fence\n"},
	}
	report := doctor.Run(config.Default(), files, doctor.Env{GitInstalled: true})
	if report.OK {
		t.Fatal("expected OK=false with a malformed file and an id mismatch present")
	}
	var idMismatch, malformed bool
	for _, f := range report.Findings {
		if f.Class == doctor.ClassSchema && f.Field == datamodel.KeyID {
			idMismatch = true
		}
		if f.Class == doctor.ClassSchema && strings.Contains(f.Message, "fence") {
			malformed = true
		}
	}
	if !idMismatch {
		t.Error("expected an id/filename mismatch finding")
	}
	if !malformed {
		t.Error("expected a malformed-frontmatter finding")
	}
	if report.Summary.Error == 0 {
		t.Error("expected a non-zero error count in the summary")
	}
}

func TestRunFreshnessSeam(t *testing.T) {
	env := doctor.Env{GitInstalled: true, InsideWorkTree: true}
	report := doctor.Run(config.Default(), nil, env)
	if !report.OK {
		t.Fatalf("empty repo should be OK, got %+v", report)
	}
	if !hasClass(report.Findings, doctor.ClassFreshness) {
		t.Fatalf("expected an index-freshness info finding via the seam, got %+v", report.Findings)
	}

	fresh := env
	fresh.Freshness = &doctor.Freshness{Built: true, Fresh: false, Reason: "head-advanced"}
	stale := doctor.Run(config.Default(), nil, fresh)
	for _, f := range stale.Findings {
		if f.Class == doctor.ClassFreshness && f.Severity == doctor.SeverityWarning {
			return
		}
	}
	t.Fatalf("expected a stale-index warning when the reporter says not fresh, got %+v", stale.Findings)
}

type fakeReporter struct {
	f   doctor.Freshness
	err error
}

func (r fakeReporter) Freshness() (doctor.Freshness, error) { return r.f, r.err }

func TestResolveFreshnessSeam(t *testing.T) {
	if doctor.ResolveFreshness(nil) != nil {
		t.Error("a nil reporter should resolve to nil (absent)")
	}
	got := doctor.ResolveFreshness(fakeReporter{f: doctor.Freshness{Fresh: true}})
	if got == nil || !got.Fresh {
		t.Errorf("expected a fresh result from the reporter, got %+v", got)
	}
	if doctor.ResolveFreshness(fakeReporter{err: errStub}) != nil {
		t.Error("a reporter error should degrade to nil, not panic")
	}
}

var errStub = stubErr("index unavailable")

type stubErr string

func (e stubErr) Error() string { return string(e) }

func TestValidateScopesToTargets(t *testing.T) {
	store := doctor.File{Path: ulidA + ".md", Content: "---\nid: " + ulidA +
		"\nnumber: KIRA-1\naliases: []\ntype: ticket\ntitle: t\nstate: TODO\nlabels: []\n" +
		"blocked_by: []\nepic: null\ncreated: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n"}
	badTarget := doctor.File{Path: ulidB + ".md", Content: "---\nid: " + ulidB +
		"\nnumber: KIRA-2\naliases: []\ntype: ticket\ntitle: t\nstate: WAT\nlabels: []\n" +
		"blocked_by: []\nepic: null\ncreated: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n"}

	report := doctor.Validate(config.Default(), []doctor.File{store}, []doctor.File{badTarget})
	if report.OK {
		t.Fatal("expected the bad target state to fail validation")
	}
	for _, f := range report.Findings {
		if f.Path != badTarget.Path {
			t.Errorf("validate should only report the target file, got a finding on %q", f.Path)
		}
	}
}
