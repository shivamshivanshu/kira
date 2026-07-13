package index_test

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type repoFixture struct {
	root  string
	store *storage.FS
	repo  gitx.Repo
}

func newRepo(t *testing.T) repoFixture {
	t.Helper()
	root := t.TempDir()
	run(t, root, "git", "init")
	run(t, root, "git", "config", "user.email", "test@example.com")
	run(t, root, "git", "config", "user.name", "tester")
	if err := os.MkdirAll(filepath.Join(root, ".kira", "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repoFixture{root: root, store: storage.New(root), repo: gitx.Repo{Dir: root}}
}

func (f repoFixture) writeTicket(t *testing.T, ulid, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.root, ".kira", "tickets", ulid+".md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f repoFixture) removeTicket(t *testing.T, ulid string) {
	t.Helper()
	if err := os.Remove(filepath.Join(f.root, ".kira", "tickets", ulid+".md")); err != nil {
		t.Fatal(err)
	}
}

func (f repoFixture) commit(t *testing.T, msg string) {
	t.Helper()
	run(t, f.root, "git", "add", "-A")
	run(t, f.root, "git", "commit", "-q", "-m", msg)
}

func (f repoFixture) commitTrailer(t *testing.T, subject, trailer string) {
	t.Helper()
	run(t, f.root, "git", "add", "-A")
	run(t, f.root, "git", "commit", "-q", "--allow-empty", "-m", subject, "-m", trailer)
}

func trailerOpts() index.Options {
	return index.Options{ProjectKey: "KIRA", TrailerKey: "Kira-Ticket", CloseTrailer: "Kira-Closes"}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v: %s", name, args, err, out)
	}
}

func ticket(ulid, number, title string) string {
	body := "---\n" +
		"id: " + ulid + "\n" +
		"number: " + number + "\n" +
		"aliases: []\n" +
		"type: ticket\n" +
		"title: " + title + "\n" +
		"state: TODO\n" +
		"labels: []\n" +
		"epic: null\n" +
		"blocked_by: []\n" +
		"created: 2026-07-10T09:14:00+05:30\n" +
		"updated: 2026-07-10T09:14:00+05:30\n" +
		"---\n\n## Description\n\nbody text for " + number + "\n"
	return body
}

func ulids(items []*datamodel.Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}

func TestLoadReconstructsLosslessly(t *testing.T) {
	f := newRepo(t)
	f.writeTicket(t, "01J8X8Q7RZTN5Y3VXW2A9K4E7F", ticket("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-2", "second"))
	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first"))
	f.commit(t, "init")

	items, _, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	disk, _, err := f.store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(items) != len(disk) {
		t.Fatalf("index has %d items, disk has %d", len(items), len(disk))
	}
	byID := map[string]*datamodel.Item{}
	for _, it := range items {
		byID[it.ID] = it
	}
	for _, want := range disk {
		got := byID[want.ID]
		if got == nil {
			t.Fatalf("index missing %s", want.ID)
		}
		want.Body = ""
		got.Body = ""
		if !reflect.DeepEqual(normalize(got), normalize(want)) {
			t.Errorf("item %s mismatch:\n index: %+v\n disk:  %+v", want.ID, got, want)
		}
	}
}

func normalize(it *datamodel.Item) datamodel.Item {
	c := *it
	if len(c.Aliases) == 0 {
		c.Aliases = nil
	}
	if len(c.Labels) == 0 {
		c.Labels = nil
	}
	if len(c.BlockedBy) == 0 {
		c.BlockedBy = nil
	}
	if len(c.Links) == 0 {
		c.Links = nil
	}
	return c
}

func TestOrderPreservedForLabelsAndLinks(t *testing.T) {
	f := newRepo(t)
	content := "---\n" +
		"id: 01J8X8Q7RZTN5Y3VXW2A9K4E7F\n" +
		"number: KIRA-1\n" +
		"aliases: []\n" +
		"type: ticket\n" +
		"title: ordered\n" +
		"state: TODO\n" +
		"labels:\n  - zebra\n  - apple\n  - mango\n" +
		"epic: null\n" +
		"blocked_by: []\n" +
		"created: 2026-07-10T09:14:00+05:30\n" +
		"updated: 2026-07-10T09:14:00+05:30\n" +
		"---\n\n## Description\n"
	f.writeTicket(t, "01J8X8Q7RZTN5Y3VXW2A9K4E7F", content)
	f.commit(t, "init")

	items, _, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	want := []string{"zebra", "apple", "mango"}
	if !reflect.DeepEqual(items[0].Labels, want) {
		t.Errorf("labels reordered: got %v want %v (file order must survive)", items[0].Labels, want)
	}
}

func TestStalenessFreshThenIncrementalThenRewrite(t *testing.T) {
	f := newRepo(t)
	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first"))
	f.commit(t, "one")

	idx := open(t, f)
	if res, _ := ensure(t, idx, f); res.Items != 1 {
		t.Fatalf("initial build items=%d", res.Items)
	}
	if res, _ := ensure(t, idx, f); res.Action != "fresh" {
		t.Fatalf("second EnsureFresh action=%q want fresh", res.Action)
	}

	f.writeTicket(t, "01J8X8Q7RZTN5Y3VXW2A9K4E7F", ticket("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-2", "second"))
	f.commit(t, "two")
	res, _ := ensure(t, idx, f)
	if res.Action != "incremental" || res.Items != 2 {
		t.Fatalf("after commit: action=%q items=%d want incremental/2", res.Action, res.Items)
	}

	run(t, f.root, "git", "commit", "-q", "--amend", "-m", "two-amended")
	res, _ = ensure(t, idx, f)
	if res.Action != "full" || res.Items != 2 {
		t.Fatalf("after amend (rewrite): action=%q items=%d want full/2", res.Action, res.Items)
	}
}

func TestIncrementalHandlesDeletionAndDirtyRevert(t *testing.T) {
	f := newRepo(t)
	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "keep"))
	f.writeTicket(t, "01J8X8Q7RZTN5Y3VXW2A9K4E7F", ticket("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-2", "drop"))
	f.commit(t, "two")

	idx := open(t, f)
	ensure(t, idx, f)

	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "keep-edited"))
	res, items := ensure(t, idx, f)
	if res.Action != "incremental" {
		t.Fatalf("dirty edit action=%q want incremental", res.Action)
	}
	if titleOf(items, "01J8X7B1Q2W3E4R5T6Y7U8I9O0") != "keep-edited" {
		t.Fatalf("dirty edit not reflected")
	}

	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "keep"))
	_, items = ensure(t, idx, f)
	if titleOf(items, "01J8X7B1Q2W3E4R5T6Y7U8I9O0") != "keep" {
		t.Fatalf("dirty revert not reflected: index still stale")
	}

	f.removeTicket(t, "01J8X8Q7RZTN5Y3VXW2A9K4E7F")
	f.commit(t, "drop KIRA-2")
	_, items = ensure(t, idx, f)
	if len(items) != 1 {
		t.Fatalf("deletion not reflected: %v", ulids(items))
	}
}

func TestSuccessiveDirtyEditsReindex(t *testing.T) {
	f := newRepo(t)
	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "v1"))
	f.commit(t, "one")

	idx := open(t, f)
	ensure(t, idx, f)

	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "v2"))
	if _, items := ensure(t, idx, f); titleOf(items, "01J8X7B1Q2W3E4R5T6Y7U8I9O0") != "v2" {
		t.Fatalf("first dirty edit not reflected")
	}
	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "v3"))
	if _, items := ensure(t, idx, f); titleOf(items, "01J8X7B1Q2W3E4R5T6Y7U8I9O0") != "v3" {
		t.Fatalf("second dirty edit to the same path missed: staleness must key on content hash, not the dirty path set")
	}
}

func TestCommitLinksTrailerAndLenient(t *testing.T) {
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.writeTicket(t, b, ticket(b, "KIRA-2", "second"))
	f.commit(t, "seed tickets")
	f.commitTrailer(t, "implement feature", "Kira-Ticket: KIRA-1")
	f.commitTrailer(t, "mention KIRA-2 in the body", "unrelated: line")

	idx := open(t, f)
	if _, err := idx.EnsureFresh(f.store, f.repo, trailerOpts()); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	links, err := idx.CommitLinks(a)
	if err != nil {
		t.Fatalf("CommitLinks: %v", err)
	}
	if len(links) != 1 || links[0].Subject != "implement feature" {
		t.Fatalf("KIRA-1 trailer link wrong: %+v", links)
	}
	linksB, err := idx.CommitLinks(b)
	if err != nil {
		t.Fatalf("CommitLinks: %v", err)
	}
	if len(linksB) != 1 || linksB[0].Subject != "mention KIRA-2 in the body" {
		t.Fatalf("KIRA-2 lenient (bare token) link wrong: %+v", linksB)
	}
}

func TestCommitLinksIncrementalAndRewrite(t *testing.T) {
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	f.commitTrailer(t, "commit one", "Kira-Ticket: KIRA-1")

	idx := open(t, f)
	if _, err := idx.EnsureFresh(f.store, f.repo, trailerOpts()); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if links, _ := idx.CommitLinks(a); len(links) != 1 {
		t.Fatalf("after first scan want 1 link, got %d", len(links))
	}

	f.commitTrailer(t, "commit two", "Kira-Ticket: KIRA-1")
	if _, err := idx.EnsureFresh(f.store, f.repo, trailerOpts()); err != nil {
		t.Fatalf("EnsureFresh incremental: %v", err)
	}
	if links, _ := idx.CommitLinks(a); len(links) != 2 {
		t.Fatalf("after incremental want 2 links, got %d", len(links))
	}

	run(t, f.root, "git", "commit", "-q", "--amend", "-m", "commit two amended", "-m", "Kira-Ticket: KIRA-1")
	if _, err := idx.EnsureFresh(f.store, f.repo, trailerOpts()); err != nil {
		t.Fatalf("EnsureFresh after rewrite: %v", err)
	}
	links, _ := idx.CommitLinks(a)
	if len(links) != 2 {
		t.Fatalf("after amend rewrite want 2 links (rebuilt), got %d: %+v", len(links), links)
	}
	for _, l := range links {
		if l.Subject == "commit two" {
			t.Fatalf("rewrite left a stale commit_link for the pre-amend sha")
		}
	}
}

func TestEventCacheRefreshesOnHeadChange(t *testing.T) {
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "x"))
	f.commit(t, "seed")
	if _, _, err := index.Load(f.store, f.repo, trailerOpts()); err != nil {
		t.Fatalf("prime: %v", err)
	}

	calls := 0
	derive := func() ([]datamodel.Event, error) {
		calls++
		return []datamodel.Event{{Ts: "2026-01-01T00:00:00Z", Field: "state", Old: "TODO", New: "DONE", CommitSHA: "abc"}}, nil
	}
	events, _, err := index.LogEntries(f.store, a, "head1", derive)
	if err != nil || calls != 1 || len(events) != 1 {
		t.Fatalf("first LogEntries: calls=%d events=%d err=%v", calls, len(events), err)
	}
	if _, _, err := index.LogEntries(f.store, a, "head1", derive); err != nil || calls != 1 {
		t.Fatalf("same head should hit cache: calls=%d err=%v", calls, err)
	}
	if _, _, err := index.LogEntries(f.store, a, "head2", derive); err != nil || calls != 2 {
		t.Fatalf("changed head should re-derive: calls=%d err=%v", calls, err)
	}
}

func TestCorruptedDBRecovers(t *testing.T) {
	f := newRepo(t)
	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first"))
	f.commit(t, "one")
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
		t.Fatalf("initial Load: %v", err)
	}
	if err := os.WriteFile(filepath.Join(f.store.CacheDir(), "index.db"), []byte("not a database"), 0o644); err != nil {
		t.Fatal(err)
	}
	items, _, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load after corruption: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("recovery produced %d items, want 1", len(items))
	}
}

func TestTransientGitFailureKeepsCache(t *testing.T) {
	f := newRepo(t)
	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first"))
	f.commit(t, "one")
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
		t.Fatalf("initial Load: %v", err)
	}
	dbFile := filepath.Join(f.store.CacheDir(), "index.db")
	if _, err := os.Stat(dbFile); err != nil {
		t.Fatalf("index.db missing after build: %v", err)
	}

	hidden := filepath.Join(f.root, ".git-hidden")
	if err := os.Rename(filepath.Join(f.root, ".git"), hidden); err != nil {
		t.Fatal(err)
	}
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err == nil {
		t.Fatal("Load must surface the git failure, not swallow it")
	}
	if _, err := os.Stat(dbFile); err != nil {
		t.Fatalf("transient git failure discarded the cache: %v", err)
	}

	if err := os.Rename(hidden, filepath.Join(f.root, ".git")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
		t.Fatalf("Load once git is reachable again: %v", err)
	}
}

func TestForeignKeyCascadeOnDelete(t *testing.T) {
	const ulid = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	content := "---\nid: " + ulid + "\nnumber: KIRA-1\naliases: []\ntype: ticket\ntitle: t\n" +
		"state: TODO\nlabels:\n  - a\n  - b\nepic: null\nblocked_by: []\n" +
		"created: 2026-07-10T09:14:00+05:30\nupdated: 2026-07-10T09:14:00+05:30\n---\n\n## Description\n"
	f.writeTicket(t, ulid, content)
	f.commit(t, "one")

	idx := open(t, f)
	ensure(t, idx, f)

	f.removeTicket(t, ulid)
	f.commit(t, "drop")
	ensure(t, idx, f)

	db, err := sql.Open("sqlite", "file:"+filepath.Join(f.store.CacheDir(), "index.db")+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var labels int
	if err := db.QueryRow("SELECT count(*) FROM labels WHERE item_id = ?", ulid).Scan(&labels); err != nil {
		t.Fatal(err)
	}
	if labels != 0 {
		t.Fatalf("deleting the item left %d orphaned label rows; ON DELETE CASCADE was not enforced on the write connection", labels)
	}
}

func TestSchemaVersionMismatchRebuilds(t *testing.T) {
	f := newRepo(t)
	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first"))
	f.commit(t, "one")
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
		t.Fatalf("initial Load: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(f.store.CacheDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version=3"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if err := os.Remove(filepath.Join(f.store.CacheDir(), "meta.json")); err != nil {
		t.Fatal(err)
	}

	idx := open(t, f)
	res, err := idx.EnsureFresh(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("stale schema must rebuild transparently, not error: %v", err)
	}
	items, err := idx.Items()
	if err != nil {
		t.Fatalf("Items after rebuild: %v", err)
	}
	if res.Action != "full" || len(items) != 1 {
		t.Fatalf("stale schema rebuild: action=%q items=%d, want full/1", res.Action, len(items))
	}
}

func TestProbeReportsFreshnessReadOnly(t *testing.T) {
	f := newRepo(t)
	f.writeTicket(t, "01J8X7B1Q2W3E4R5T6Y7U8I9O0", ticket("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first"))
	f.commit(t, "one")

	rep, err := index.Probe(f.store, f.repo)
	if err != nil {
		t.Fatalf("Probe before build: %v", err)
	}
	if rep.Built || rep.Fresh {
		t.Fatalf("probe before build = %+v, want absent (not built)", rep)
	}

	idx := open(t, f)
	ensure(t, idx, f)

	rep, err = index.Probe(f.store, f.repo)
	if err != nil {
		t.Fatalf("Probe after build: %v", err)
	}
	if !rep.Built || !rep.Fresh {
		t.Fatalf("probe after build = %+v, want built+fresh", rep)
	}

	f.writeTicket(t, "01J8X8Q7RZTN5Y3VXW2A9K4E7F", ticket("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-2", "second"))
	rep, err = index.Probe(f.store, f.repo)
	if err != nil {
		t.Fatalf("Probe after edit: %v", err)
	}
	if rep.Fresh {
		t.Fatalf("probe after uncommitted edit = %+v, want stale", rep)
	}
}

func open(t *testing.T, f repoFixture) *index.Index {
	t.Helper()
	idx, err := index.Open(f.store.CacheDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx
}

func ensure(t *testing.T, idx *index.Index, f repoFixture) (index.Result, []*datamodel.Item) {
	t.Helper()
	res, err := idx.EnsureFresh(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	items, err := idx.Items()
	if err != nil {
		t.Fatalf("Items: %v", err)
	}
	return res, items
}

func titleOf(items []*datamodel.Item, ulid string) string {
	for _, it := range items {
		if it.ID == ulid {
			return it.Title
		}
	}
	return ""
}
