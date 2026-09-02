package doctor_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/id"
)

// builtinTypes mirrors the workflows a default config declares, which is what
// makes a type exist.
func builtinTypes() map[string]bool {
	return map[string]bool{datamodel.TypeTicket: true, datamodel.TypeEpic: true}
}

func builtinSchemas(t *testing.T) map[string]entityschema.Schema {
	t.Helper()
	s, err := entityschema.Load("")
	if err != nil {
		t.Fatalf("loading built-in schemas: %v", err)
	}
	return s
}

const (
	ulidA = "01HAAAAAAAAAAAAAAAAAAAAAAA"
	ulidB = "01HBBBBBBBBBBBBBBBBBBBBBBB"
	ulidC = "01HCCCCCCCCCCCCCCCCCCCCCCC"
)

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
	t.Parallel()
	it, parsed, findings := doctor.Lint("no fence here\n", builtinSchemas(t))
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
	t.Parallel()
	content := "---\nid: " + ulidA + "\nnumber: KIRA-1\naliases: []\ntype: ticket\ntitle: t\n" +
		"state: TODO\nlabels: []\nblocked_by: []\nepic: null\nbogus_key: 1\n" +
		"created: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n## Description\n"
	_, parsed, findings := doctor.Lint(content, builtinSchemas(t))
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

func TestLintDeclaredExtraDoesNotWarn(t *testing.T) {
	t.Parallel()
	schemas := builtinSchemas(t)
	ticket := schemas[datamodel.TypeTicket]
	ticket.Fields = append(ticket.Fields, entityschema.FieldDef{Name: "severity", Type: entityschema.FieldString})
	schemas[datamodel.TypeTicket] = ticket

	content := "---\nid: " + ulidA + "\nnumber: KIRA-1\naliases: []\ntype: ticket\ntitle: t\n" +
		"state: TODO\nlabels: []\nblocked_by: []\nepic: null\nseverity: high\nbogus_key: 1\n" +
		"created: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n## Description\n"
	_, parsed, findings := doctor.Lint(content, schemas)
	if !parsed {
		t.Fatalf("expected a well-formed item to parse, got findings %+v", findings)
	}
	var sawSeverity, sawBogus bool
	for _, f := range findings {
		if f.Class != doctor.ClassSchema {
			continue
		}
		switch f.Field {
		case "severity":
			sawSeverity = true
		case "bogus_key":
			sawBogus = true
		}
	}
	if sawSeverity {
		t.Errorf("a schema-declared extra must not warn, got %+v", findings)
	}
	if !sawBogus {
		t.Errorf("a truly undeclared extra must still warn, got %+v", findings)
	}
}

func TestDoctorFlagsInvalidDeclaredExtra(t *testing.T) {
	t.Parallel()
	schemas := builtinSchemas(t)
	ticket := schemas[datamodel.TypeTicket]
	ticket.Fields = append(slices.Clone(ticket.Fields), entityschema.FieldDef{
		Name: "story_points", Type: entityschema.FieldInt, Placement: entityschema.PlacementFrontmatter,
	})
	schemas[datamodel.TypeTicket] = ticket
	itemWith := func(pointsLine string) []doctor.File {
		content := "---\nid: " + ulidA + "\nnumber: KIRA-1\naliases: []\ntype: ticket\ntitle: t\n" +
			"state: TODO\nlabels: []\nblocked_by: []\nepic: null\n" + pointsLine +
			"created: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n## Description\n"
		return []doctor.File{{Path: ulidA + ".md", Content: content}}
	}
	violation := func(report *doctor.Report) bool {
		for _, f := range report.Findings {
			if f.Class == doctor.ClassSchema && f.Severity == doctor.SeverityError && f.Field == "story_points" {
				return true
			}
		}
		return false
	}

	bad := doctor.Run(config.Default(), itemWith("story_points: abc\n"), nil, doctor.Env{GitInstalled: true}, schemas)
	if !violation(bad) {
		t.Fatalf("expected a schema error for the mistyped declared extra, got %+v", bad.Findings)
	}
	good := doctor.Run(config.Default(), itemWith("story_points: 3\n"), nil, doctor.Env{GitInstalled: true}, schemas)
	if violation(good) || !good.OK {
		t.Fatalf("a conforming declared extra must not be flagged, got %+v", good.Findings)
	}
}

func TestLintSurfacesDuplicateKeys(t *testing.T) {
	t.Parallel()
	content := "---\nid: " + ulidA + "\nnumber: KIRA-1\naliases: []\ntype: ticket\ntitle: t\n" +
		"state: TODO\nstate: DONE\nlabels: []\nblocked_by: []\nepic: null\n" +
		"created: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n## Description\n"
	_, parsed, findings := doctor.Lint(content, builtinSchemas(t))
	if parsed {
		t.Fatal("duplicate keys must fail parsing")
	}
	for _, f := range findings {
		if f.Class == doctor.ClassSchema && f.Severity == doctor.SeverityError && strings.Contains(f.Message, "duplicate key") {
			return
		}
	}
	t.Fatalf("expected a duplicate-key schema finding, got %+v", findings)
}

func TestLintMalformedComment(t *testing.T) {
	t.Parallel()
	body := "## Comments\n<!-- kira:comment id=X author=y ts=bogus -->\nunterminated body\n"
	_, _, findings := doctor.Lint("---\nid: "+ulidA+"\nnumber: KIRA-1\naliases: []\ntype: ticket\n"+
		"title: t\nstate: TODO\nlabels: []\nblocked_by: []\nepic: null\n"+
		"created: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n"+body, builtinSchemas(t))
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
	t.Parallel()
	it := valid(ulidA, "KIRA-1")
	if f := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t)); len(f) != 0 {
		t.Fatalf("expected no findings for a clean item, got %+v", f)
	}
}

func TestCheckUnknownBoardPrefixWarns(t *testing.T) {
	it := valid(ulidA, "ZZZ-9")
	findings := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t))
	if classes(findings)[doctor.ClassBoard] != doctor.SeverityWarning {
		t.Fatalf("expected a board warning for the unconfigured prefix ZZZ, got %+v", findings)
	}
}

func TestCheckUnknownBoardPrefixWarnsForHashNumber(t *testing.T) {
	it := valid(ulidA, "ZZZ-a1b2c3")
	findings := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t))
	if classes(findings)[doctor.ClassBoard] != doctor.SeverityWarning {
		t.Fatalf("expected a board warning for the unconfigured hash-style prefix ZZZ, got %+v", findings)
	}
}

func TestCheckUnknownStateAndSprintAndDue(t *testing.T) {
	t.Parallel()
	it := valid(ulidA, "KIRA-1")
	it.State = "NOPE"
	it.Sprint = new("2099-S1")
	it.Due = new("not-a-date")
	findings := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t))
	cl := classes(findings)
	if cl[doctor.ClassState] != doctor.SeverityError {
		t.Errorf("expected state error, got %+v", findings)
	}
	if !hasClass(findings, doctor.ClassSchema) {
		t.Errorf("expected schema findings for sprint/due, got %+v", findings)
	}
}

func TestCheckVocabStrictVsWarn(t *testing.T) {
	t.Parallel()
	it := valid(ulidA, "KIRA-1")
	it.Labels = []string{"not-a-label"}

	warn := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t))
	if classes(warn)[doctor.ClassVocab] != doctor.SeverityWarning {
		t.Errorf("expected vocab warning under non-strict labels, got %+v", warn)
	}

	strict := config.Default()
	strict.Labels.Strict = true
	got := doctor.Check(strict, resolver(it), it, builtinSchemas(t))
	if classes(got)[doctor.ClassVocab] != doctor.SeverityError {
		t.Errorf("expected vocab error under strict labels, got %+v", got)
	}
}

func TestCheckExemptsSystemCapturedLabel(t *testing.T) {
	t.Parallel()
	it := valid(ulidA, "KIRA-1")
	it.Labels = []string{datamodel.CapturedLabel}

	strict := config.Default()
	strict.Labels.Strict = true
	if got := doctor.Check(strict, resolver(it), it, builtinSchemas(t)); hasClass(got, doctor.ClassVocab) {
		t.Errorf("system %q label must be exempt from vocab checks, got %+v", datamodel.CapturedLabel, got)
	}
}

func TestCheckEnumVocabFollowsStrict(t *testing.T) {
	t.Parallel()
	it := valid(ulidA, "KIRA-1")
	it.Priority = new("P9")

	lenient := config.Default()
	lenient.Labels.Strict = false
	if classes(doctor.Check(lenient, resolver(it), it, builtinSchemas(t)))[doctor.ClassVocab] != doctor.SeverityWarning {
		t.Fatalf("expected enum vocab warning under non-strict labels, matching create")
	}

	strict := config.Default()
	strict.Labels.Strict = true
	if classes(doctor.Check(strict, resolver(it), it, builtinSchemas(t)))[doctor.ClassVocab] != doctor.SeverityError {
		t.Fatalf("expected enum vocab error under strict labels, matching create")
	}
}

func TestCheckDanglingAndSelfRef(t *testing.T) {
	t.Parallel()
	it := valid(ulidA, "KIRA-1")
	it.Parent = new(ulidB)
	it.BlockedBy = []string{ulidA}
	findings := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t))
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

func TestSequentialOutliers(t *testing.T) {
	t.Parallel()
	items := []*datamodel.Item{
		valid(ulidA, "KIRA-1"),
		valid(ulidB, "KIRA-42"),
		valid(ulidC, "KIRA-971436"),
	}
	findings := doctor.SequentialOutliers(items)
	if len(findings) != 1 {
		t.Fatalf("expected exactly one outlier, got %+v", findings)
	}
	f := findings[0]
	if f.Class != doctor.ClassNumberOutlier || f.Severity != doctor.SeverityWarning {
		t.Fatalf("expected a %s warning, got %+v", doctor.ClassNumberOutlier, f)
	}
	if f.ItemID != ulidC || f.Number != "KIRA-971436" {
		t.Fatalf("outlier misattributed: %+v", f)
	}
	if !strings.Contains(f.Message, "board move") {
		t.Fatalf("message must name the renumber repair, got %q", f.Message)
	}
}

func TestSequentialOutliersNeedBaselineAndGap(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		numbers []string
	}{
		{"no small baseline", []string{"KIRA-971436", "KIRA-812345"}},
		{"continuous six-digit board", []string{"KIRA-99999", "KIRA-100001"}},
		{"hash suffix with letters", []string{"KIRA-1", "KIRA-9X4MV3"}},
		{"foreign board unaffected", []string{"KIRA-971436", "XYZ-1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ulids := []string{ulidA, ulidB, ulidC}
			var items []*datamodel.Item
			for i, n := range tc.numbers {
				items = append(items, valid(ulids[i], n))
			}
			if findings := doctor.SequentialOutliers(items); len(findings) != 0 {
				t.Fatalf("expected no outliers, got %+v", findings)
			}
		})
	}
}

func TestCollisionsLiveLive(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	a := valid(ulidA, "KIRA-2")
	a.Aliases = []string{"KIRA-1"}
	b := valid(ulidB, "KIRA-3")
	if f := doctor.Collisions([]*datamodel.Item{a, b}); len(f) != 0 {
		t.Fatalf("expected no collision, got %+v", f)
	}
}

func TestCollisionsAliasAlias(t *testing.T) {
	t.Parallel()
	a := valid(ulidA, "KIRA-8")
	a.Aliases = []string{"KIRA-1"}
	b := valid(ulidB, "KIRA-9")
	b.Aliases = []string{"KIRA-1"}
	findings := doctor.Collisions([]*datamodel.Item{a, b})
	if len(findings) != 1 || findings[0].Collision.Kind != doctor.CollisionAliasAlias {
		t.Fatalf("expected one alias-alias collision, got %+v", findings)
	}
	c := findings[0].Collision
	if c.Keep != ulidA {
		t.Errorf("expected earlier ULID %s to be the keeper, got %s", ulidA, c.Keep)
	}
	if len(c.LiveIDs) != 0 || !slices.Equal(c.AliasIDs, []string{ulidA, ulidB}) {
		t.Errorf("expected no live holders and both alias holders, got live=%v alias=%v", c.LiveIDs, c.AliasIDs)
	}
}

func TestCollisionsCaseInsensitive(t *testing.T) {
	t.Parallel()
	a := valid(ulidA, "KIRA-5")
	b := valid(ulidB, "kira-5")
	findings := doctor.Collisions([]*datamodel.Item{a, b})
	if len(findings) != 1 || findings[0].Collision.Kind != doctor.CollisionLiveLive {
		t.Fatalf("expected one live-live collision across case variants, got %+v", findings)
	}
	c := findings[0].Collision
	if c.Value != "KIRA-5" || findings[0].Number != "KIRA-5" {
		t.Errorf("report should carry the first-seen original form KIRA-5, got value=%q number=%q", c.Value, findings[0].Number)
	}
	if !slices.Equal(c.LiveIDs, []string{ulidA, ulidB}) {
		t.Errorf("expected both live holders, got %v", c.LiveIDs)
	}
}

func TestCheckEmptyRankIsSchemaError(t *testing.T) {
	t.Parallel()
	it := valid(ulidA, "KIRA-1")
	it.Rank = new("")
	findings := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t))
	for _, f := range findings {
		if f.Class == doctor.ClassSchema && f.Field == datamodel.KeyRank && f.Severity == doctor.SeverityError {
			return
		}
	}
	t.Fatalf("expected a schema error for an empty rank, got %+v", findings)
}

func TestRefTypeSkipsMissingParent(t *testing.T) {
	t.Parallel()
	child := valid(ulidA, "KIRA-1")
	child.Parent = new(ulidC)
	items := []*datamodel.Item{child}
	if f := doctor.RefTypeMismatches(builtinSchemas(t), items, resolver(items...), builtinTypes()); len(f) != 0 {
		t.Fatalf("a missing/unresolved parent must not raise a ref-type finding, got %+v", f)
	}
}

func TestEnvOptionalBinsAndHookFindings(t *testing.T) {
	t.Parallel()
	env := doctor.Env{
		GitInstalled:        true,
		InsideWorkTree:      true,
		MissingOptionalBins: []string{"rg"},
		TrackedHooks:        []string{"post-merge", "pre-commit"},
		InstalledHooks:      []string{"post-merge"},
	}
	report := doctor.Run(config.Default(), nil, nil, env, builtinSchemas(t))
	var optionalBin, missingHook, mergeDriver, ticketAttr bool
	for _, f := range report.Findings {
		switch {
		case f.Class == doctor.ClassEnv && strings.Contains(f.Message, "rg not found"):
			optionalBin = true
		case f.Class == doctor.ClassHooks && strings.Contains(f.Message, "pre-commit is not installed"):
			missingHook = true
		case f.Class == doctor.ClassHooks && strings.Contains(f.Message, "merge driver"):
			mergeDriver = true
		case f.Class == doctor.ClassHooks && strings.Contains(f.Message, "ticket merge attribute"):
			ticketAttr = true
		}
	}
	if !optionalBin {
		t.Error("expected an optional-bin info finding for rg")
	}
	if !missingHook {
		t.Error("expected an uninstalled-hook finding for pre-commit")
	}
	if !mergeDriver || !ticketAttr {
		t.Errorf("expected merge-driver and ticket-attr findings, got %+v", report.Findings)
	}
}

func TestParentCycle(t *testing.T) {
	t.Parallel()
	a := valid(ulidA, "KIRA-1")
	b := valid(ulidB, "KIRA-2")
	a.Parent = new(ulidB)
	b.Parent = new(ulidA)
	items := []*datamodel.Item{a, b}
	findings := doctor.RefCycles(items, resolver(items...))
	if len(findings) != 2 {
		t.Fatalf("expected a finding per cycle member, got %+v", findings)
	}
	for _, f := range findings {
		if f.Class != doctor.ClassRefCycle || f.Field != datamodel.KeyParent || f.Severity != doctor.SeverityError {
			t.Fatalf("unexpected finding %+v", f)
		}
	}
}

func TestParentChainAcyclic(t *testing.T) {
	t.Parallel()
	child := valid(ulidA, "KIRA-1")
	parent := valid(ulidB, "KIRA-2")
	child.Parent = new(ulidB)
	items := []*datamodel.Item{child, parent}
	if f := doctor.RefCycles(items, resolver(items...)); len(f) != 0 {
		t.Fatalf("expected no cycle in a linear chain, got %+v", f)
	}
}

func TestRefTypeEpicParentDetected(t *testing.T) {
	t.Parallel()
	child := valid(ulidA, "KIRA-1")
	parent := valid(ulidB, "KIRA-2")
	child.Parent = new(ulidB)
	items := []*datamodel.Item{child, parent}
	f := doctor.RefTypeMismatches(builtinSchemas(t), items, resolver(items...), builtinTypes())
	if len(f) != 1 || f[0].Class != doctor.ClassRefType || f[0].ItemID != ulidA || f[0].Field != datamodel.KeyParent {
		t.Fatalf("expected one ref-type finding on the child's epic, got %+v", f)
	}
}

func TestRefTypeEpicParentAccepted(t *testing.T) {
	t.Parallel()
	child := valid(ulidA, "KIRA-1")
	parent := valid(ulidB, "KIRA-2")
	parent.Type = datamodel.TypeEpic
	child.Parent = new(ulidB)
	items := []*datamodel.Item{child, parent}
	if f := doctor.RefTypeMismatches(builtinSchemas(t), items, resolver(items...), builtinTypes()); len(f) != 0 {
		t.Fatalf("an epic parent must be accepted, got %+v", f)
	}
}

func TestRefTypeBlockedByMismatch(t *testing.T) {
	t.Parallel()
	child := valid(ulidA, "KIRA-1")
	blocker := valid(ulidB, "KIRA-2")
	blocker.Type = datamodel.TypeEpic
	child.BlockedBy = []string{ulidB}
	items := []*datamodel.Item{child, blocker}
	f := doctor.RefTypeMismatches(builtinSchemas(t), items, resolver(items...), builtinTypes())
	if len(f) != 1 || f[0].Class != doctor.ClassRefType || f[0].Field != datamodel.KeyBlockedBy {
		t.Fatalf("expected one ref-type finding on blocked_by, got %+v", f)
	}
}

func TestRefTypeBlockedByAccepted(t *testing.T) {
	t.Parallel()
	child := valid(ulidA, "KIRA-1")
	blocker := valid(ulidB, "KIRA-2")
	child.BlockedBy = []string{ulidB}
	items := []*datamodel.Item{child, blocker}
	if f := doctor.RefTypeMismatches(builtinSchemas(t), items, resolver(items...), builtinTypes()); len(f) != 0 {
		t.Fatalf("a ticket blocked by a ticket must be accepted, got %+v", f)
	}
}

func TestRefTypeExtraRefFieldMismatch(t *testing.T) {
	t.Parallel()
	schemas := builtinSchemas(t)
	ticketSchema := schemas[datamodel.TypeTicket]
	ticketSchema.Fields = append(slices.Clone(ticketSchema.Fields), entityschema.FieldDef{
		Name: "reviewer_of", Type: entityschema.FieldRef, Target: datamodel.TypeEpic,
	})
	schemas[datamodel.TypeTicket] = ticketSchema

	child := valid(ulidA, "KIRA-1")
	child.ExtraScalars = map[string]datamodel.Rendered{"reviewer_of": ulidB}
	other := valid(ulidB, "KIRA-2")
	items := []*datamodel.Item{child, other}

	f := doctor.RefTypeMismatches(schemas, items, resolver(items...), builtinTypes())
	if len(f) != 1 || f[0].Class != doctor.ClassRefType || f[0].ItemID != ulidA || f[0].Field != "reviewer_of" {
		t.Fatalf("expected one ref-type finding on the declared ref extra, got %+v", f)
	}
}

func TestCheckDanglingExtraRefField(t *testing.T) {
	t.Parallel()
	schemas := builtinSchemas(t)
	ticketSchema := schemas[datamodel.TypeTicket]
	ticketSchema.Fields = append(slices.Clone(ticketSchema.Fields), entityschema.FieldDef{
		Name: "reviewer_of", Type: entityschema.FieldRef, Target: datamodel.TypeTicket,
	})
	schemas[datamodel.TypeTicket] = ticketSchema

	it := valid(ulidA, "KIRA-1")
	it.ExtraScalars = map[string]datamodel.Rendered{"reviewer_of": ulidB}

	f := doctor.Check(config.Default(), resolver(it), it, schemas)
	if len(f) != 1 || f[0].Class != doctor.ClassRef || f[0].Field != "reviewer_of" {
		t.Fatalf("a custom ref pointing at nothing must be flagged dangling, got %+v", f)
	}
}

func TestCheckReportsBuiltinDanglingRefOnce(t *testing.T) {
	t.Parallel()
	it := valid(ulidA, "KIRA-1")
	it.Parent = new(ulidB)

	f := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t))
	if len(f) != 1 || f[0].Field != datamodel.KeyParent {
		t.Fatalf("epic is both a built-in and schema-declared; it must be reported once, got %+v", f)
	}
}

// A type exists because config declares a workflow for it, with or without a
// schema file, so a ref targeting one must still be checked.
func TestRefChecksReachConfigOnlyTargetType(t *testing.T) {
	t.Parallel()
	schemas := builtinSchemas(t)
	ticketSchema := schemas[datamodel.TypeTicket]
	ticketSchema.Fields = append(slices.Clone(ticketSchema.Fields), entityschema.FieldDef{
		Name: "saga_ref", Type: entityschema.FieldRef, Target: "saga",
	})
	schemas[datamodel.TypeTicket] = ticketSchema
	types := map[string]bool{datamodel.TypeTicket: true, datamodel.TypeEpic: true, "saga": true}

	dangling := valid(ulidA, "KIRA-1")
	dangling.ExtraScalars = map[string]datamodel.Rendered{"saga_ref": ulidC}
	cfg := config.Default()
	cfg.Workflows["saga"] = cfg.Workflows[datamodel.TypeTicket]

	f := doctor.Check(cfg, resolver(dangling), dangling, schemas)
	if len(f) != 1 || f[0].Class != doctor.ClassRef || f[0].Field != "saga_ref" {
		t.Fatalf("a ref to a config-only type must be dangling-checked, got %+v", f)
	}

	wrongType := valid(ulidA, "KIRA-1")
	wrongType.ExtraScalars = map[string]datamodel.Rendered{"saga_ref": ulidB}
	items := []*datamodel.Item{wrongType, valid(ulidB, "KIRA-2")}
	m := doctor.RefTypeMismatches(schemas, items, resolver(items...), types)
	if len(m) != 1 || m[0].Field != "saga_ref" {
		t.Fatalf("a ref to a config-only type must be type-checked, got %+v", m)
	}
}

func TestOwnerAndSprintValuesAreNotItemRefs(t *testing.T) {
	t.Parallel()
	other := valid(ulidB, "KIRA-2")
	it := valid(ulidA, "KIRA-1")
	it.Owner = new(ulidB)
	it.Reporter = new("KIRA-2")
	it.Sprint = new(ulidB)
	items := []*datamodel.Item{it, other}
	if f := doctor.RefTypeMismatches(builtinSchemas(t), items, resolver(items...), builtinTypes()); len(f) != 0 {
		t.Fatalf("source-backed ref fields must not be type-checked against items, got %+v", f)
	}

	named := valid(ulidA, "KIRA-1")
	named.Owner = new("alice")
	named.Reporter = new("bob")
	named.Sprint = new("2026-S1")
	for _, f := range doctor.Check(config.Default(), resolver(named), named, builtinSchemas(t)) {
		if f.Class == doctor.ClassRef {
			t.Fatalf("source-backed ref fields must not be dangling-checked against items, got %+v", f)
		}
	}
}

func TestDanglingRefNotDoubleReported(t *testing.T) {
	t.Parallel()
	content := "---\nid: " + ulidA + "\nnumber: KIRA-1\naliases: []\ntype: ticket\ntitle: t\n" +
		"state: TODO\nlabels: []\nblocked_by: [" + ulidC + "]\nepic: null\n" +
		"created: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n## Description\n"
	files := []doctor.File{{Path: ulidA + ".md", Content: content}}
	report := doctor.Run(config.Default(), files, nil, doctor.Env{GitInstalled: true}, builtinSchemas(t))
	var dangling int
	for _, f := range report.Findings {
		switch f.Class {
		case doctor.ClassRef:
			dangling++
		case doctor.ClassRefType:
			t.Fatalf("a dangling ref must not also be reported as wrong-type: %+v", f)
		}
	}
	if dangling != 1 {
		t.Fatalf("expected exactly one dangling-ref finding, got %+v", report.Findings)
	}
}

func TestRefCycleDetected(t *testing.T) {
	t.Parallel()
	a := valid(ulidA, "KIRA-1")
	b := valid(ulidB, "KIRA-2")
	a.BlockedBy = []string{ulidB}
	b.BlockedBy = []string{ulidA}
	items := []*datamodel.Item{a, b}
	f := doctor.RefCycles(items, resolver(items...))
	if len(f) != 2 {
		t.Fatalf("expected a finding per cycle member, got %+v", f)
	}
	for _, x := range f {
		if x.Class != doctor.ClassRefCycle || x.Field != datamodel.KeyBlockedBy {
			t.Fatalf("unexpected finding %+v", x)
		}
	}
}

func TestRefCycleIgnoresLinearAndSymmetric(t *testing.T) {
	t.Parallel()
	a := valid(ulidA, "KIRA-1")
	b := valid(ulidB, "KIRA-2")
	a.BlockedBy = []string{ulidB}
	a.Links = map[string][]string{string(datamodel.LinkRelates): {ulidB}}
	b.Links = map[string][]string{string(datamodel.LinkRelates): {ulidA}}
	items := []*datamodel.Item{a, b}
	if f := doctor.RefCycles(items, resolver(items...)); len(f) != 0 {
		t.Fatalf("linear blocked_by and symmetric relates are not cycles, got %+v", f)
	}
}

func TestRunAggregatesAndFlagsIdentity(t *testing.T) {
	t.Parallel()
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
	report := doctor.Run(config.Default(), files, nil, doctor.Env{GitInstalled: true}, builtinSchemas(t))
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

func TestRunReportsStrayFiles(t *testing.T) {
	t.Parallel()
	report := doctor.Run(config.Default(), nil, []string{"README.md"}, doctor.Env{GitInstalled: true}, builtinSchemas(t))
	if report.OK {
		t.Fatal("expected OK=false with a stray file present")
	}
	var found bool
	for _, f := range report.Findings {
		if f.Class == doctor.ClassStray && f.Path == "README.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a %s finding for README.md, got %+v", doctor.ClassStray, report.Findings)
	}
}

func TestRunFreshnessSeam(t *testing.T) {
	t.Parallel()
	env := doctor.Env{GitInstalled: true, InsideWorkTree: true}
	report := doctor.Run(config.Default(), nil, nil, env, builtinSchemas(t))
	if !report.OK {
		t.Fatalf("empty repo should be OK, got %+v", report)
	}
	if !hasClass(report.Findings, doctor.ClassFreshness) {
		t.Fatalf("expected an index-freshness info finding via the seam, got %+v", report.Findings)
	}

	fresh := env
	fresh.Freshness = &doctor.Freshness{Built: true, Fresh: false, Reason: "head-advanced"}
	stale := doctor.Run(config.Default(), nil, nil, fresh, builtinSchemas(t))
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
	t.Parallel()
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
	t.Parallel()
	store := doctor.File{Path: ulidA + ".md", Content: "---\nid: " + ulidA +
		"\nnumber: KIRA-1\naliases: []\ntype: ticket\ntitle: t\nstate: TODO\nlabels: []\n" +
		"blocked_by: []\nepic: null\ncreated: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n"}
	badTarget := doctor.File{Path: ulidB + ".md", Content: "---\nid: " + ulidB +
		"\nnumber: KIRA-2\naliases: []\ntype: ticket\ntitle: t\nstate: WAT\nlabels: []\n" +
		"blocked_by: []\nepic: null\ncreated: 2026-07-01T00:00:00Z\nupdated: 2026-07-01T00:00:00Z\n---\n"}

	report := doctor.Validate(config.Default(), []doctor.File{store}, []doctor.File{badTarget}, builtinSchemas(t))
	if report.OK {
		t.Fatal("expected the bad target state to fail validation")
	}
	for _, f := range report.Findings {
		if f.Path != badTarget.Path {
			t.Errorf("validate should only report the target file, got a finding on %q", f.Path)
		}
	}
}

func TestCheckStaleResolutionWarns(t *testing.T) {
	t.Parallel()
	it := valid(ulidA, "KIRA-1")
	it.Resolution = new("done")
	findings := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t))
	if classes(findings)[doctor.ClassState] != doctor.SeverityWarning {
		t.Fatalf("expected a stale-resolution warning on a non-done state, got %+v", findings)
	}
	it.State = "WONT_DO"
	if f := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t)); hasClass(f, doctor.ClassState) {
		t.Fatalf("resolution on a done state must be clean, got %+v", f)
	}
}

func TestCheckDoneStateWithNilResolutionWarns(t *testing.T) {
	t.Parallel()
	it := valid(ulidA, "KIRA-1")
	it.State = "WONT_DO"
	findings := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t))
	if classes(findings)[doctor.ClassState] != doctor.SeverityWarning {
		t.Fatalf("expected a missing-resolution warning on a done state with nil resolution, got %+v", findings)
	}
	it.Resolution = new("dropped")
	if f := doctor.Check(config.Default(), resolver(it), it, builtinSchemas(t)); hasClass(f, doctor.ClassState) {
		t.Fatalf("resolution set on a done state must be clean, got %+v", f)
	}
}

func TestStateDriftDetected(t *testing.T) {
	t.Parallel()
	drifted := entityschema.Schema{Name: datamodel.TypeTicket, Fields: []entityschema.FieldDef{{
		Name: datamodel.KeyState, Type: entityschema.FieldState,
		States: []entityschema.StateValue{
			{Key: "TODO", Category: datamodel.CategoryTodo},
			{Key: "DOING", Category: datamodel.CategoryDoing},
		},
	}}}
	f := doctor.StateDrift(config.Default(), map[string]entityschema.Schema{datamodel.TypeTicket: drifted})
	if len(f) != 1 || f[0].Class != doctor.ClassStateDrift || f[0].Severity != doctor.SeverityWarning {
		t.Fatalf("expected one state-drift warning, got %+v", f)
	}
	if !strings.Contains(f[0].Message, "TODO, DOING") || !strings.Contains(f[0].Message, "IN_PROGRESS") {
		t.Fatalf("message should show both state sets, got %q", f[0].Message)
	}
}

func TestStateDriftQuietWhenAligned(t *testing.T) {
	t.Parallel()
	if f := doctor.StateDrift(config.Default(), builtinSchemas(t)); len(f) != 0 {
		t.Fatalf("builtin schemas must agree with the default workflows, got %+v", f)
	}
}

func TestWipOverrunsReportsOverLimitState(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	wf := cfg.Workflows[datamodel.TypeTicket]
	for i, st := range wf.States {
		if st.Key == "IN_PROGRESS" {
			wf.States[i].Wip = 1
		}
	}
	cfg.Workflows[datamodel.TypeTicket] = wf
	items := []*datamodel.Item{
		{ID: ulidA, Type: datamodel.TypeTicket, State: "IN_PROGRESS"},
		{ID: ulidB, Type: datamodel.TypeTicket, State: "IN_PROGRESS"},
		{ID: ulidC, Type: datamodel.TypeEpic, State: "IN_PROGRESS"},
	}

	f := doctor.WipOverruns(cfg, items)

	if len(f) != 1 || f[0].Class != doctor.ClassWipLimit || f[0].Severity != doctor.SeverityWarning {
		t.Fatalf("expected one wip-limit warning, got %+v", f)
	}
	if !strings.Contains(f[0].Message, "(2 > 1)") || !strings.Contains(f[0].Message, "IN_PROGRESS") {
		t.Fatalf("message should show state and counts, got %q", f[0].Message)
	}
}

func TestWipOverrunsQuietAtOrUnderLimit(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	wf := cfg.Workflows[datamodel.TypeTicket]
	for i := range wf.States {
		wf.States[i].Wip = 1
	}
	cfg.Workflows[datamodel.TypeTicket] = wf
	items := []*datamodel.Item{
		{ID: ulidA, Type: datamodel.TypeTicket, State: "IN_PROGRESS"},
		{ID: ulidB, Type: datamodel.TypeTicket, State: "TODO"},
	}

	if f := doctor.WipOverruns(cfg, items); len(f) != 0 {
		t.Fatalf("expected no findings at the limit, got %+v", f)
	}
}

func TestSchemaIconsWarnsOnlyOnGlyphsTooWideToBeGlyphs(t *testing.T) {
	cases := []struct {
		icon string
		warn bool
	}{
		{"ticket", true},
		{"board", true},
		{"epic", true},
		{"", false},
		{"\U0001F680", false},
		{"[T]", false},
		{"", false},
	}
	for _, c := range cases {
		schemas := map[string]entityschema.Schema{
			"x": {Name: "x", Representation: entityschema.Representation{Icon: c.icon}},
		}

		findings := doctor.SchemaIcons(schemas)

		if got := len(findings) > 0; got != c.warn {
			t.Errorf("icon %q: warned = %v, want %v", c.icon, got, c.warn)
		}
	}
}
