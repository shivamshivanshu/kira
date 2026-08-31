package core_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const futureFieldDecl = `{"name": "future_field", "type": "string", "placement": "frontmatter"}`

func declareTicketExtras(t *testing.T, s *core.Store, fieldsJSON string) *core.Store {
	t.Helper()
	return declareTicketSchema(t, s, `{"name": "ticket", "fields": [`+fieldsJSON+`]}`)
}

func declareTicketSchema(t *testing.T, s *core.Store, schemaJSON string) *core.Store {
	t.Helper()
	path := filepath.Join(storage.New(s.Root()).SchemaDir(), "ticket.json")
	if err := os.WriteFile(path, []byte(schemaJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	fresh, err := core.Discover(s.Root())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	return fresh
}

func TestDeclaredExtraDoesNotBlockWrites(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Declared extra")
	overwriteItem(t, s, res.ID, withUnknownKey(mustReadItem(t, s, res.ID)))
	s = declareTicketExtras(t, s, futureFieldDecl)

	if _, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{}); err != nil {
		t.Fatalf("a schema-declared extra must not block mutation: %v", err)
	}
	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, unknownKeyLine) {
		t.Fatalf("mutation dropped the declared extra:\n%s", got)
	}
}

func TestUndeclaredExtraStillBlocksWrites(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Undeclared extra")
	content := strings.Replace(mustReadItem(t, s, res.ID), "---\n", "---\nother_field: x\n", 1)
	overwriteItem(t, s, res.ID, content)
	s = declareTicketExtras(t, s, futureFieldDecl)

	_, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{})
	assertUpgradeRefusal(t, err)
	if !strings.Contains(err.Error(), "other_field") {
		t.Fatalf("refusal must name the undeclared key: %v", err)
	}
}

func TestExtrasBlockWhenTypeHasNoSchema(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "No schema for type")
	content := strings.Replace(withUnknownKey(mustReadItem(t, s, res.ID)), "type: ticket\n", "type: gadget\n", 1)
	overwriteItem(t, s, res.ID, content)
	s = declareTicketExtras(t, s, futureFieldDecl)

	_, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{})
	assertUpgradeRefusal(t, err)
}

func TestDeclaredUncapturedExtraStillBlocks(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Declared mapping")
	content := strings.Replace(mustReadItem(t, s, res.ID), "---\n", "---\nfuture_field:\n  nested: 1\n", 1)
	overwriteItem(t, s, res.ID, content)
	s = declareTicketExtras(t, s, futureFieldDecl)

	_, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{})
	assertUpgradeRefusal(t, err)
}

func TestEditFieldSetsDeclaredExtras(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Typed extras")
	s = declareTicketExtras(t, s,
		`{"name": "points", "type": "int", "placement": "frontmatter"}, `+
			`{"name": "note", "type": "string", "placement": "frontmatter"}`)

	fields := []core.FieldEdit{{Key: "points", Value: "42"}, {Key: "note", Value: "a: b"}}
	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: fields}); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	got := mustReadItem(t, s, res.ID)
	if !strings.Contains(got, "points: 42\n") {
		t.Fatalf("int extra must serialize unquoted:\n%s", got)
	}
	if !strings.Contains(got, "note: 'a: b'\n") {
		t.Fatalf("quote-needing extra must serialize quoted:\n%s", got)
	}

	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "points", Value: "43"}}}); err != nil {
		t.Fatalf("re-edit after reparse: %v", err)
	}
	got = mustReadItem(t, s, res.ID)
	if !strings.Contains(got, "points: 43\n") || !strings.Contains(got, "note: 'a: b'\n") {
		t.Fatalf("stored extras did not round-trip through parse and serialize:\n%s", got)
	}
}

func TestEditFieldRejectsUndeclaredKey(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	mustCreate(t, s, cfg, "Reject unknown --field")
	s = declareTicketExtras(t, s, futureFieldDecl)

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "titl", Value: "x"}}})
	if err == nil {
		t.Fatal("expected an unknown-field error")
	}
	if !strings.Contains(err.Error(), `unknown or immutable field "titl"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	var e *errx.Error
	if !errors.As(err, &e) || !strings.Contains(e.Hint, "did you mean `--field title=...`") {
		t.Fatalf("expected the did-you-mean hint, got %v", err)
	}
}

func TestEditFieldHonorsImmutableExtra(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	mustCreate(t, s, cfg, "Immutable extra")
	s = declareTicketExtras(t, s, `{"name": "external_id", "type": "string", "immutable": true, "placement": "frontmatter"}`)

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "external_id", Value: "x"}}})
	if err == nil || !strings.Contains(err.Error(), `unknown or immutable field "external_id"`) {
		t.Fatalf("expected the immutable-field refusal, got %v", err)
	}
}

const typedTicketSchema = `{
  "name": "ticket",
  "enums": [{"name": "sev", "values": ["LOW", "HIGH"]}],
  "fields": [
    {"name": "severity", "type": "enum", "enum": "sev", "placement": "frontmatter"},
    {"name": "story_points", "type": "int", "placement": "frontmatter"}
  ]
}`

func TestEditFieldRejectsMistypedExtraValues(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Typed rejection")
	s = declareTicketSchema(t, s, typedTicketSchema)
	before := mustReadItem(t, s, res.ID)

	for key, tt := range map[string]struct{ value, want string }{
		"story_points": {"abc", "expected type int"},
		"severity":     {"BOGUS", `not in the "sev" vocabulary`},
	} {
		_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: key, Value: tt.value}}})
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("%s=%s: expected a violation containing %q, got %v", key, tt.value, tt.want, err)
		}
	}
	if got := mustReadItem(t, s, res.ID); got != before {
		t.Fatalf("a refused edit mutated the file:\n%s", got)
	}

	fields := []core.FieldEdit{{Key: "story_points", Value: "5"}, {Key: "severity", Value: "LOW"}}
	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: fields}); err != nil {
		t.Fatalf("conforming extras must pass: %v", err)
	}
	got := mustReadItem(t, s, res.ID)
	if !strings.Contains(got, "story_points: 5\n") || !strings.Contains(got, "severity: LOW\n") {
		t.Fatalf("conforming extras did not persist:\n%s", got)
	}
}

func TestMutationBlockedByInvalidOnDiskExtra(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Invalid on disk")
	content := strings.Replace(mustReadItem(t, s, res.ID), "---\n", "---\nseverity: BOGUS\n", 1)
	overwriteItem(t, s, res.ID, content)
	s = declareTicketSchema(t, s, typedTicketSchema)

	_, err := s.Assign(cfg, "KIRA-1", "bob", core.AssignOpts{})
	if err == nil || !strings.Contains(err.Error(), `not in the "sev" vocabulary`) {
		t.Fatalf("expected the on-disk violation to block mutation, got %v", err)
	}
}

func declareQueryableExtras(t *testing.T, s *core.Store) *core.Store {
	t.Helper()
	return declareTicketExtras(t, s,
		`{"name": "team", "type": "string", "placement": "frontmatter"}, `+
			`{"name": "watchers", "type": "string", "list": true, "placement": "frontmatter"}, `+
			`{"name": "points", "type": "int", "placement": "frontmatter"}`)
}

func setExtras(t *testing.T, s *core.Store, cfg *datamodel.Config, ref string) {
	t.Helper()
	fields := []core.FieldEdit{{Key: "team", Value: "infra"}, {Key: "watchers", Value: "alice,bob"}, {Key: "points", Value: "8"}}
	if _, err := s.Edit(cfg, ref, core.EditOpts{Fields: fields}); err != nil {
		t.Fatalf("Edit %s: %v", ref, err)
	}
}

func requireIndexServed(t *testing.T, res *datamodel.ListResult) {
	t.Helper()
	for _, n := range res.StderrNotes {
		if n.Code == datamodel.WarnIndexFallback {
			t.Fatalf("list fell back to the filesystem, so the query never exercised the cache: %v", n.Args)
		}
	}
}

func TestQueryOnExtrasServedByWarmIndex(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	s = declareQueryableExtras(t, s)
	mustCreate(t, s, cfg, "tagged")
	mustCreate(t, s, cfg, "plain")
	setExtras(t, s, cfg, "KIRA-1")

	warm, err := s.List(cfg, core.ListOpts{})
	if err != nil {
		t.Fatalf("warming List: %v", err)
	}
	requireIndexServed(t, warm)
	rep, err := index.Probe(storage.New(s.Root()), gitx.Repo{Dir: s.Root()})
	if err != nil || !rep.Built {
		t.Fatalf("cache not built after the warming List (rep=%+v err=%v)", rep, err)
	}

	cfg.Filters = map[string]string{"infra": "team = infra"}
	queries := map[string]core.ListOpts{
		"scalar equality":  {Query: "team = infra"},
		"list contains":    {Query: "watchers = bob"},
		"numeric ordered":  {Query: "points > 5"},
		"saved filter":     {Filter: "infra"},
		"combined builtin": {Query: "team = infra AND state = TODO"},
	}
	for name, opts := range queries {
		res, err := s.List(cfg, opts)
		if err != nil {
			t.Fatalf("%s: List: %v", name, err)
		}
		requireIndexServed(t, res)
		if got := listNumbers(res); got != "KIRA-1" {
			t.Errorf("%s: matched %q, want KIRA-1", name, got)
		}
	}

	if _, err := s.List(cfg, core.ListOpts{Query: "bogus = x"}); err == nil || !strings.Contains(err.Error(), "team") {
		t.Errorf("unknown field must error and suggest declared extras, got %v", err)
	}
}

func TestExtrasInJSONShapes(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	s = declareQueryableExtras(t, s)
	mustCreate(t, s, cfg, "tagged")
	mustCreate(t, s, cfg, "plain")
	setExtras(t, s, cfg, "KIRA-1")

	res, err := s.List(cfg, core.ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	rows := map[string]datamodel.ListItem{}
	for _, r := range res.Items {
		rows[r.Number] = r
	}
	want := map[string]datamodel.ExtraJSON{"team": "infra", "watchers": []string{"alice", "bob"}, "points": "8"}
	if !reflect.DeepEqual(rows["KIRA-1"].Extras, want) {
		t.Errorf("list extras = %#v, want %#v", rows["KIRA-1"].Extras, want)
	}

	tagged, err := json.Marshal(rows["KIRA-1"])
	if err != nil || !strings.Contains(string(tagged), `"extras":{"points":"8","team":"infra","watchers":["alice","bob"]}`) {
		t.Errorf("list --json must carry extras, got %s (err=%v)", tagged, err)
	}
	plain, err := json.Marshal(rows["KIRA-2"])
	if err != nil || strings.Contains(string(plain), `"extras"`) {
		t.Errorf("an item without extras must marshal without the key, got %s (err=%v)", plain, err)
	}

	show, err := s.Show(cfg, "KIRA-1", "")
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if !reflect.DeepEqual(show.Extras, want) {
		t.Errorf("show extras = %#v, want %#v", show.Extras, want)
	}
}

func TestListExtraIsEmptyParityWithLabels(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	s = declareQueryableExtras(t, s)
	explicit := mustCreate(t, s, cfg, "explicitly empty")
	mustCreate(t, s, cfg, "key absent")
	content := strings.Replace(mustReadItem(t, s, explicit.ID), "---\n", "---\nwatchers: []\n", 1)
	overwriteItem(t, s, explicit.ID, content)
	show, err := s.Show(cfg, "KIRA-1", "")
	if err != nil || !reflect.DeepEqual(show.Extras["watchers"], datamodel.ExtraJSON([]string{})) {
		t.Fatalf("fixture must carry an explicit empty list, got %#v (err=%v)", show.Extras, err)
	}

	for _, field := range []string{"watchers", "label"} {
		res, err := s.List(cfg, core.ListOpts{Query: field + " IS EMPTY"})
		if err != nil {
			t.Fatalf("%s IS EMPTY: %v", field, err)
		}
		requireIndexServed(t, res)
		if got := listNumbers(res); got != "KIRA-1,KIRA-2" {
			t.Errorf("%s IS EMPTY matched %q, want both items", field, got)
		}
		res, err = s.List(cfg, core.ListOpts{Query: field + " IS NOT EMPTY"})
		if err != nil {
			t.Fatalf("%s IS NOT EMPTY: %v", field, err)
		}
		if got := listNumbers(res); got != "" {
			t.Errorf("%s IS NOT EMPTY matched %q, want none", field, got)
		}
	}
}

func TestEditFromFileRejectsUndeclaredKey(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Clean item")
	before := mustReadItem(t, s, res.ID)
	src := writeTempItem(t, strings.Replace(before, "---\n", "---\nfoo: bar\n", 1))

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{FromFile: src})
	var e *errx.Error
	if !errors.As(err, &e) || e.Code != errx.ExitUser {
		t.Fatalf("want ExitUser refusal, got %v (%T)", err, err)
	}
	if !strings.Contains(e.Error(), `"foo"`) {
		t.Fatalf("refusal must name the offending key: %v", err)
	}
	if got := mustReadItem(t, s, res.ID); got != before {
		t.Fatalf("a refused edit must not mutate the file:\n%s", got)
	}
}

func TestEditFromFileAcceptsDeclaredExtra(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Declared extra from file")
	s = declareTicketExtras(t, s, futureFieldDecl)
	src := writeTempItem(t, strings.Replace(mustReadItem(t, s, res.ID), "---\n", "---\nfuture_field: x\n", 1))

	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{FromFile: src}); err != nil {
		t.Fatalf("a schema-declared extra must edit fine: %v", err)
	}
	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, "future_field: x\n") {
		t.Fatalf("edit dropped the declared extra:\n%s", got)
	}
}

func TestEditorModeRejectsUndeclaredKey(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Clean item")
	before := mustReadItem(t, s, res.ID)

	counter := filepath.Join(t.TempDir(), "counter")
	seen := filepath.Join(t.TempDir(), "seen")
	t.Setenv("KIRA_COUNTER", counter)
	t.Setenv("KIRA_SEEN", seen)
	writeEditorScript(t,
		"n=$(cat \"$KIRA_COUNTER\" 2>/dev/null || echo 0)\n"+
			"n=$((n+1)); echo \"$n\" > \"$KIRA_COUNTER\"\n"+
			"if [ \"$n\" -eq 1 ]; then\n"+
			"  head -n 1 \"$1\" > \"$1.tmp\"; printf 'foo: bar\\n' >> \"$1.tmp\"; tail -n +2 \"$1\" >> \"$1.tmp\"; mv \"$1.tmp\" \"$1\"\n"+
			"else\n"+
			"  cp \"$1\" \"$KIRA_SEEN\"\n"+
			"fi\n")

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{})
	var e *errx.Error
	if !errors.As(err, &e) || e.Code != errx.ExitUser {
		t.Fatalf("want ExitUser refusal, got %v (%T)", err, err)
	}
	banner, readErr := os.ReadFile(seen)
	if readErr != nil {
		t.Fatalf("editor never saw a retry banner: %v", readErr)
	}
	if !strings.Contains(string(banner), `"foo"`) {
		t.Fatalf("retry banner must name the offending key:\n%s", banner)
	}
	if got := mustReadItem(t, s, res.ID); got != before {
		t.Fatalf("a refused edit must not mutate the file:\n%s", got)
	}
}

const immutableExtraDecl = `{"name": "external_id", "type": "string", "immutable": true, "placement": "frontmatter"}`

func commitKira(t *testing.T, s *core.Store, subject string) {
	t.Helper()
	repo := gitx.Repo{Dir: s.Root()}
	if err := repo.Stage(storage.DirName); err != nil {
		t.Fatalf("staging %s: %v", storage.DirName, err)
	}
	if err := repo.CommitScoped([]string{storage.DirName}, subject); err != nil {
		t.Fatalf("committing %s: %v", storage.DirName, err)
	}
}

func headSHA(t *testing.T, s *core.Store) string {
	t.Helper()
	sha, err := gitx.Repo{Dir: s.Root()}.Output("rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return sha
}

func TestExtraEditsSurfaceInLogAndBlame(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	mustCreate(t, s, cfg, "History of an extra")
	s = declareTicketSchema(t, s, typedTicketSchema)

	for _, value := range []string{"LOW", "HIGH"} {
		if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "severity", Value: value}}}); err != nil {
			t.Fatalf("Edit severity=%s: %v", value, err)
		}
	}

	logRes, err := s.Log(cfg, "KIRA-1")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	transitions := map[string]bool{}
	for _, e := range logRes.Entries {
		if e.Kind == datamodel.LogKindEvent && e.Field == "severity" {
			transitions[e.Old+"->"+e.New] = true
		}
	}
	if !transitions["->LOW"] || !transitions["LOW->HIGH"] {
		t.Fatalf("log must carry both severity events, got %+v", logRes.Entries)
	}

	blame, err := s.Blame(cfg, "KIRA-1")
	if err != nil {
		t.Fatalf("Blame: %v", err)
	}
	var got *datamodel.BlameField
	for i := range blame.Fields {
		if blame.Fields[i].Field == "severity" {
			got = &blame.Fields[i]
		}
	}
	if got == nil {
		t.Fatalf("blame must attribute a declared extra, got %+v", blame.Fields)
	}
	if got.Value != "HIGH" || got.SourceKind != datamodel.BlameSourceCommit {
		t.Fatalf("blame severity = %+v, want HIGH sourced from its commit", *got)
	}
}

func TestAtCompilesQueriesAgainstTheSnapshotSchema(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	mustCreate(t, s, cfg, "tagged")
	s = declareQueryableExtras(t, s)
	commitKira(t, s, "declare the extras")
	setExtras(t, s, cfg, "KIRA-1")
	declared := headSHA(t, s)

	s = declareTicketExtras(t, s, futureFieldDecl)
	commitKira(t, s, "undeclare the extras")

	if _, err := s.List(cfg, core.ListOpts{Query: "team = infra"}); err == nil {
		t.Fatal("the fixture must undeclare team in the working tree, else --at proves nothing")
	}

	res, err := s.List(cfg, core.ListOpts{Query: "team = infra", At: declared})
	if err != nil {
		t.Fatalf("--at must compile the query against the schema of that tree: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-1" {
		t.Fatalf("--at matched %q, want KIRA-1", got)
	}
}

func TestImmutableExtraSurvivesFullDocumentEdits(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Immutable extra")
	overwriteItem(t, s, res.ID, strings.Replace(mustReadItem(t, s, res.ID), "---\n", "---\nexternal_id: abc\n", 1))
	s = declareTicketExtras(t, s, immutableExtraDecl)

	for name, edit := range map[string]func(string) string{
		"rewritten": func(c string) string { return strings.Replace(c, "external_id: abc", "external_id: zzz", 1) },
		"deleted":   func(c string) string { return strings.Replace(c, "external_id: abc\n", "", 1) },
	} {
		src := writeTempItem(t, edit(strings.Replace(mustReadItem(t, s, res.ID), "title:", "title: "+name+" #", 1)))
		if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{FromFile: src}); err != nil {
			t.Fatalf("%s: Edit --from-file: %v", name, err)
		}
		got := mustReadItem(t, s, res.ID)
		if !strings.Contains(got, "external_id: abc\n") {
			t.Fatalf("%s: --from-file changed an immutable extra:\n%s", name, got)
		}
		if !strings.Contains(got, name) {
			t.Fatalf("%s: the rest of the edit did not land:\n%s", name, got)
		}
	}
}

func TestImmutableExtraSurvivesEditorEdits(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Immutable extra in the editor")
	overwriteItem(t, s, res.ID, strings.Replace(mustReadItem(t, s, res.ID), "---\n", "---\nexternal_id: abc\n", 1))
	s = declareTicketExtras(t, s, immutableExtraDecl)
	writeEditorScript(t, "sed -e 's/external_id: abc/external_id: zzz/' -e 's/^title: \"/title: \"edited /' \"$1\" > \"$1.tmp\" && mv \"$1.tmp\" \"$1\"\n")

	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{}); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	got := mustReadItem(t, s, res.ID)
	if !strings.Contains(got, "external_id: abc\n") {
		t.Fatalf("the editor changed an immutable extra:\n%s", got)
	}
	if !strings.Contains(got, `title: "edited `) {
		t.Fatalf("the rest of the edit did not land:\n%s", got)
	}
}

func TestFieldHintSuggestsDeclaredExtras(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	mustCreate(t, s, cfg, "Typo'd extra")
	s = declareQueryableExtras(t, s)

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "tem", Value: "infra"}}})
	var e *errx.Error
	if !errors.As(err, &e) {
		t.Fatalf("want *errx.Error, got %v (%T)", err, err)
	}
	if !strings.Contains(e.Hint, "did you mean `--field team=...`") {
		t.Fatalf("hint must suggest the declared extra, got %q", e.Hint)
	}
}
