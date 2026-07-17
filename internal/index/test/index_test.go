package index_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

type repoFixture struct {
	root  string
	store *storage.FS
	repo  gitx.Repo
}

func newRepo(t *testing.T) repoFixture {
	t.Helper()
	root := testutil.InitGitRepo(t)
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
	return index.Options{
		ProjectKey:       "KIRA",
		TrailerKey:       "Kira-Ticket",
		CloseTrailer:     "Kira-Closes",
		LinkMarkers:      datamodel.LinkMarkers,
		ReferenceMarkers: datamodel.ReferenceMarkers,
	}
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
	t.Parallel()
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
	c.Activity = ""
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.writeTicket(t, b, ticket(b, "KIRA-2", "second"))
	f.commit(t, "seed tickets")
	f.commitTrailer(t, "implement feature", "Kira-Ticket: KIRA-1")
	f.commitTrailer(t, "mention KIRA-2 in the body", "unrelated: line")

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, trailerOpts(), false); err != nil {
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
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	f.commitTrailer(t, "commit one", "Kira-Ticket: KIRA-1")

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, trailerOpts(), false); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if links, _ := idx.CommitLinks(a); len(links) != 1 {
		t.Fatalf("after first scan want 1 link, got %d", len(links))
	}

	f.commitTrailer(t, "commit two", "Kira-Ticket: KIRA-1")
	if _, err := index.Refresh(f.store, f.repo, trailerOpts(), false); err != nil {
		t.Fatalf("EnsureFresh incremental: %v", err)
	}
	if links, _ := idx.CommitLinks(a); len(links) != 2 {
		t.Fatalf("after incremental want 2 links, got %d", len(links))
	}

	run(t, f.root, "git", "commit", "-q", "--amend", "-m", "commit two amended", "-m", "Kira-Ticket: KIRA-1")
	if _, err := index.Refresh(f.store, f.repo, trailerOpts(), false); err != nil {
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

func ticketWithAlias(ulid, number, alias, title string) string {
	return "---\n" +
		"id: " + ulid + "\n" +
		"number: " + number + "\n" +
		"aliases:\n  - " + alias + "\n" +
		"type: ticket\n" +
		"title: " + title + "\n" +
		"state: TODO\n" +
		"labels: []\n" +
		"epic: null\n" +
		"blocked_by: []\n" +
		"created: 2026-07-10T09:14:00+05:30\n" +
		"updated: 2026-07-10T09:14:00+05:30\n" +
		"---\n\n## Description\n\nbody\n"
}

func kindsOf(links []index.CommitLink) map[string]index.LinkKind {
	out := map[string]index.LinkKind{}
	for _, l := range links {
		out[l.Subject] = l.Kind
	}
	return out
}

func TestCommitLinkKinds(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticketWithAlias(a, "KIRA-1", "KIRA-9", "first"))
	f.writeTicket(t, b, ticket(b, "KIRA-2", "second"))
	f.commit(t, "seed tickets")
	f.commitTrailer(t, "[[KIRA-1]] subject marker", "unrelated: x")
	f.commitTrailer(t, "plain subject", "refs KIRA-2 in the body")
	f.commitTrailer(t, "[[KIRA-2]] conflict", "Kira-Ticket: KIRA-1")
	f.commitTrailer(t, "[[KIRA-9]] via alias", "unrelated: y")

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, trailerOpts(), false); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}

	linksA, err := idx.CommitLinks(a)
	if err != nil {
		t.Fatalf("CommitLinks(a): %v", err)
	}
	kindsA := kindsOf(linksA)
	for _, subj := range []string{"[[KIRA-1]] subject marker", "[[KIRA-2]] conflict", "[[KIRA-9]] via alias"} {
		if kindsA[subj] != index.LinkLinked {
			t.Fatalf("KIRA-1 commit %q kind=%q want linked; links=%+v", subj, kindsA[subj], linksA)
		}
	}
	if len(linksA) != 3 {
		t.Fatalf("KIRA-1 want 3 linked commits, got %d: %+v", len(linksA), linksA)
	}

	linksB, err := idx.CommitLinks(b)
	if err != nil {
		t.Fatalf("CommitLinks(b): %v", err)
	}
	kindsB := kindsOf(linksB)
	if kindsB["plain subject"] != index.LinkReferenced {
		t.Fatalf("KIRA-2 bare body ref should be referenced; got %q", kindsB["plain subject"])
	}
	if kindsB["[[KIRA-2]] conflict"] != index.LinkReferenced {
		t.Fatalf("KIRA-2 subject marker losing to the trailer should demote to referenced; got %q", kindsB["[[KIRA-2]] conflict"])
	}
	if len(linksB) != 2 {
		t.Fatalf("KIRA-2 want 2 referenced commits, got %d: %+v", len(linksB), linksB)
	}
}

func TestCommitLinkLeadingNumberUnresolvedTrailerBlocks(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	f.commitTrailer(t, "KIRA-1 fix flaky path", "Kira-Ticket: KIRA-15")

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, trailerOpts(), false); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	links := mustCommitLinks(t, idx, a)
	if len(links) != 1 || links[0].Kind != index.LinkReferenced {
		t.Fatalf("an unresolved trailer must still outrank the leading number: %+v", links)
	}
}

func TestCommitLinkLeadingNumberYieldsToMarker(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.writeTicket(t, b, ticket(b, "KIRA-2", "second"))
	f.commit(t, "seed")
	f.commitTrailer(t, "KIRA-1 fix [[KIRA-2]] widget", "unrelated: x")

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, trailerOpts(), false); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	linksB := mustCommitLinks(t, idx, b)
	if len(linksB) != 1 || linksB[0].Kind != index.LinkLinked {
		t.Fatalf("the subject marker outranks the leading number: %+v", linksB)
	}
	linksA := mustCommitLinks(t, idx, a)
	if len(linksA) != 1 || linksA[0].Kind != index.LinkReferenced {
		t.Fatalf("the leading number losing to the marker should demote to referenced: %+v", linksA)
	}
}

func TestCommitLinkLeadingNumber(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticketWithAlias(a, "KIRA-1", "KIRA-9", "first"))
	f.writeTicket(t, b, ticket(b, "KIRA-2", "second"))
	f.commit(t, "seed tickets")
	f.commitTrailer(t, "KIRA-1 implement directly", "unrelated: x")
	f.commitTrailer(t, "fix KIRA-1 mid subject", "unrelated: y")
	f.commitTrailer(t, "plain subject", "KIRA-1 in the body only")
	f.commitTrailer(t, "KIRA-2 claimed but trailer wins", "Kira-Ticket: KIRA-1")
	f.commitTrailer(t, "KIRA-9 via alias", "unrelated: z")

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, trailerOpts(), false); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}

	linksA, err := idx.CommitLinks(a)
	if err != nil {
		t.Fatalf("CommitLinks(a): %v", err)
	}
	kindsA := kindsOf(linksA)
	want := map[string]index.LinkKind{
		"KIRA-1 implement directly":       index.LinkLinked,
		"fix KIRA-1 mid subject":          index.LinkReferenced,
		"plain subject":                   index.LinkReferenced,
		"KIRA-2 claimed but trailer wins": index.LinkLinked,
		"KIRA-9 via alias":                index.LinkLinked,
	}
	for subj, kind := range want {
		if kindsA[subj] != kind {
			t.Fatalf("KIRA-1 commit %q kind=%q want %q; links=%+v", subj, kindsA[subj], kind, linksA)
		}
	}
	if len(linksA) != len(want) {
		t.Fatalf("KIRA-1 want %d commits, got %d: %+v", len(want), len(linksA), linksA)
	}

	linksB, err := idx.CommitLinks(b)
	if err != nil {
		t.Fatalf("CommitLinks(b): %v", err)
	}
	if len(linksB) != 1 || linksB[0].Kind != index.LinkReferenced {
		t.Fatalf("KIRA-2 leading number losing to the trailer should demote to referenced: %+v", linksB)
	}
}

func TestCommitLinkLeadingNumberSubjectPrefix(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	f.commitTrailer(t, "kira: KIRA-1 state TODO -> DOING", "unrelated: x")
	f.commitTrailer(t, "notes on kira: KIRA-1 later", "unrelated: y")

	opts := trailerOpts()
	opts.SubjectPrefix = "kira: "

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, opts, false); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	kinds := kindsOf(mustCommitLinks(t, idx, a))
	if kinds["kira: KIRA-1 state TODO -> DOING"] != index.LinkLinked {
		t.Fatalf("prefixed leading number should link: %+v", kinds)
	}
	if kinds["notes on kira: KIRA-1 later"] != index.LinkReferenced {
		t.Fatalf("mid-subject prefix mention should stay referenced: %+v", kinds)
	}
}

func TestCommitLinkLeadingNumberDisabled(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	f.commitTrailer(t, "KIRA-1 old behavior", "unrelated: x")

	opts := trailerOpts()
	opts.LinkMarkers = []datamodel.LinkMarker{datamodel.LinkMarkerTrailer, datamodel.LinkMarkerSubject}

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, opts, false); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	links := mustCommitLinks(t, idx, a)
	if len(links) != 1 || links[0].Kind != index.LinkReferenced {
		t.Fatalf("with leading_number off a bare leading token must stay referenced: %+v", links)
	}
}

func TestCommitLinkLeadingNumberRetroactiveReclassify(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	f.commitTrailer(t, "KIRA-1 claim by convention", "unrelated: x")

	off := trailerOpts()
	off.LinkMarkers = []datamodel.LinkMarker{datamodel.LinkMarkerTrailer, datamodel.LinkMarkerSubject}

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, off, false); err != nil {
		t.Fatalf("EnsureFresh knob-off: %v", err)
	}
	links := mustCommitLinks(t, idx, a)
	if len(links) != 1 || links[0].Kind != index.LinkReferenced {
		t.Fatalf("knob-off scan should classify as referenced: %+v", links)
	}

	if _, err := index.Refresh(f.store, f.repo, trailerOpts(), false); err != nil {
		t.Fatalf("EnsureFresh knob-on: %v", err)
	}
	links = mustCommitLinks(t, idx, a)
	if len(links) != 1 || links[0].Kind != index.LinkLinked {
		t.Fatalf("flipping the knob must reclassify without a history change: %+v", links)
	}
}

func mustCommitLinks(t *testing.T, idx *index.Index, ulid string) []index.CommitLink {
	t.Helper()
	links, err := idx.CommitLinks(ulid)
	if err != nil {
		t.Fatalf("CommitLinks: %v", err)
	}
	return links
}

func TestCommitLinkMarkersDisable(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	f.commitTrailer(t, "[[KIRA-1]] marker only", "unrelated: x")
	f.commitTrailer(t, "bare only", "mentions KIRA-1 here")

	opts := trailerOpts()
	opts.LinkMarkers = []datamodel.LinkMarker{datamodel.LinkMarkerTrailer}
	opts.ReferenceMarkers = nil

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, opts, false); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	links, err := idx.CommitLinks(a)
	if err != nil {
		t.Fatalf("CommitLinks: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("subject markers ignored and bare disabled should yield no links, got %+v", links)
	}
}

func TestCommitLinkNonPrimaryMarkerReferenced(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.writeTicket(t, b, ticket(b, "KIRA-2", "second"))
	f.commit(t, "seed")
	f.commitTrailer(t, "[[KIRA-1]] primary [[KIRA-2]] secondary", "unrelated: x")

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, trailerOpts(), false); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	linksA, _ := idx.CommitLinks(a)
	if len(linksA) != 1 || linksA[0].Kind != index.LinkLinked {
		t.Fatalf("first subject marker should promote to linked: %+v", linksA)
	}
	linksB, _ := idx.CommitLinks(b)
	if len(linksB) != 1 || linksB[0].Kind != index.LinkReferenced {
		t.Fatalf("second subject marker should demote to referenced via the bracket-stripped bare scan: %+v", linksB)
	}
}

func TestEventCacheRefreshesOnHeadChange(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestForcedRefreshRebuildsAfterCorruption(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.writeTicket(t, b, ticket(b, "KIRA-2", "second"))
	f.commit(t, "one")
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
		t.Fatalf("initial Load: %v", err)
	}
	if err := os.WriteFile(filepath.Join(f.store.CacheDir(), "index.db"), []byte("not a database"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := index.Refresh(f.store, f.repo, index.Options{}, true)
	if err != nil {
		t.Fatalf("forced Refresh over corrupt cache: %v", err)
	}
	if res.Items != 2 {
		t.Fatalf("forced Refresh reported %d items, want 2", res.Items)
	}

	items, _, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load after forced rebuild: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("rebuilt index has %d items, want 2", len(items))
	}
}

func TestTransientGitFailureKeepsCache(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	defer func() { _ = db.Close() }()
	var labels int
	if err := db.QueryRow("SELECT count(*) FROM labels WHERE item_id = ?", ulid).Scan(&labels); err != nil {
		t.Fatal(err)
	}
	if labels != 0 {
		t.Fatalf("deleting the item left %d orphaned label rows; ON DELETE CASCADE was not enforced on the write connection", labels)
	}
}

func TestSchemaVersionMismatchRebuilds(t *testing.T) {
	t.Parallel()
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
	_ = db.Close()
	if err := os.Remove(filepath.Join(f.store.CacheDir(), "meta.json")); err != nil {
		t.Fatal(err)
	}

	idx := open(t, f)
	res, err := index.Refresh(f.store, f.repo, index.Options{}, false)
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
	t.Parallel()
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
	t.Cleanup(func() { _ = idx.Close() })
	return idx
}

func ensure(t *testing.T, idx *index.Index, f repoFixture) (index.Result, []*datamodel.Item) {
	t.Helper()
	res, err := index.Refresh(f.store, f.repo, index.Options{}, false)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
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

func (f repoFixture) commitTrailerAt(t *testing.T, date, subject, trailer string) {
	t.Helper()
	run(t, f.root, "git", "add", "-A")
	cmd := exec.Command("git", "commit", "-q", "--allow-empty", "-m", subject, "-m", trailer)
	cmd.Dir = f.root
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
}

func (f repoFixture) commitTs(t *testing.T, rev string) string {
	t.Helper()
	cmd := exec.Command("git", "show", "-s", "--format=%cI", rev)
	cmd.Dir = f.root
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git show: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestActivityIsMaxOfUpdatedAndCommitTs(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.writeTicket(t, b, ticket(b, "KIRA-2", "second"))
	f.commit(t, "seed tickets")
	f.commitTrailerAt(t, "2030-01-01T00:00:00Z", "land it", "Kira-Ticket: KIRA-1")

	items, _, err := index.Load(f.store, f.repo, trailerOpts())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	byID := map[string]*datamodel.Item{}
	for _, it := range items {
		byID[it.ID] = it
	}

	linked := byID[a]
	if want := f.commitTs(t, "HEAD"); linked.Activity != want {
		t.Fatalf("KIRA-1 activity = %q, want the 2030 commit ts %q", linked.Activity, want)
	}
	if !after(t, linked.Activity, linked.Updated) {
		t.Fatalf("KIRA-1 activity %q should exceed updated %q", linked.Activity, linked.Updated)
	}

	unlinked := byID[b]
	if unlinked.Activity != unlinked.Updated {
		t.Fatalf("KIRA-2 has no commit link: activity %q should equal updated %q", unlinked.Activity, unlinked.Updated)
	}
}

func after(t *testing.T, a, b string) bool {
	t.Helper()
	ta, err := time.Parse(time.RFC3339, a)
	if err != nil {
		t.Fatalf("parsing %q: %v", a, err)
	}
	tb, err := time.Parse(time.RFC3339, b)
	if err != nil {
		t.Fatalf("parsing %q: %v", b, err)
	}
	return ta.After(tb)
}

func TestUntrackedTicketsIndexed(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	if err := os.WriteFile(filepath.Join(f.root, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.commit(t, "init")
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
		t.Fatalf("initial Load: %v", err)
	}

	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	items, res, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load with untracked ticket: %v", err)
	}
	if res.Action == "fresh" || len(items) != 1 {
		t.Fatalf("untracked ticket invisible: action=%q items=%d want reindex/1", res.Action, len(items))
	}

	f.writeTicket(t, b, ticket(b, "KIRA-2", "second"))
	items, res, err = index.Load(f.store, f.repo, index.Options{})
	if err != nil || res.Action != "incremental" || len(items) != 2 {
		t.Fatalf("second untracked ticket missed: action=%q items=%d err=%v want incremental/2", res.Action, len(items), err)
	}

	f.writeTicket(t, a, ticket(a, "KIRA-1", "edited"))
	if items, _, err = index.Load(f.store, f.repo, index.Options{}); err != nil || titleOf(items, a) != "edited" {
		t.Fatalf("edit to untracked ticket missed: title=%q err=%v", titleOf(items, a), err)
	}
}

const deadSHA = "0123456789abcdef0123456789abcdef01234567"

func poisonMetaSHAs(t *testing.T, f repoFixture, sha string) {
	t.Helper()
	path := filepath.Join(f.store.CacheDir(), "meta.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	m["last_indexed_head_sha"] = sha
	if wm, ok := m["trailer_watermarks"].(map[string]any); ok {
		for k := range wm {
			wm[k] = sha
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDeadSHAInMetaTriggersRebuild(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	f.commitTrailer(t, "commit one", "Kira-Ticket: KIRA-1")
	if _, _, err := index.Load(f.store, f.repo, trailerOpts()); err != nil {
		t.Fatalf("initial Load: %v", err)
	}

	poisonMetaSHAs(t, f, deadSHA)
	items, res, err := index.Load(f.store, f.repo, trailerOpts())
	if err != nil {
		t.Fatalf("dead SHA must trigger a rebuild, not a permanent fallback: %v", err)
	}
	if res.Action != "full" || res.Reason != "history-rewritten" || len(items) != 1 {
		t.Fatalf("dead SHA recovery: action=%q reason=%q items=%d want full/history-rewritten/1", res.Action, res.Reason, len(items))
	}

	idx := open(t, f)
	links := mustCommitLinks(t, idx, a)
	if len(links) != 1 {
		t.Fatalf("commit links after dead-watermark rebuild: %+v", links)
	}
}

func TestDeletedDBRebuilds(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
		t.Fatalf("initial Load: %v", err)
	}

	if err := os.Remove(filepath.Join(f.store.CacheDir(), "index.db")); err != nil {
		t.Fatal(err)
	}
	items, res, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load after rm index.db: %v", err)
	}
	if res.Action != "full" || len(items) != 1 {
		t.Fatalf("deleted DB served stale meta: action=%q items=%d want full/1", res.Action, len(items))
	}
}

func TestDuplicateIDSkippedWithWarning(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.writeTicket(t, b, ticket(a, "KIRA-2", "dup"))
	f.commit(t, "seed")

	items, res, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load over duplicate ids must not fail: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("duplicate id: items=%d want 1", len(items))
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], b+".md") ||
		!strings.Contains(res.Warnings[0], a+".md") || !strings.Contains(res.Warnings[0], "duplicate id") {
		t.Fatalf("duplicate warning must name both files: %v", res.Warnings)
	}

	_, res, err = index.Load(f.store, f.repo, index.Options{})
	if err != nil || res.Action != "fresh" {
		t.Fatalf("fresh run: action=%q err=%v", res.Action, err)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "duplicate id") {
		t.Fatalf("duplicate warning must replay on fresh runs: %v", res.Warnings)
	}
}

func TestDuplicateIDSkippedOnRefresh(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
		t.Fatalf("initial Load: %v", err)
	}

	f.writeTicket(t, b, ticket(a, "KIRA-2", "dup"))
	items, res, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(items) != 1 || titleOf(items, a) != "first" {
		t.Fatalf("refresh over duplicate id: items=%d title=%q", len(items), titleOf(items, a))
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "duplicate id") {
		t.Fatalf("refresh duplicate warning missing: %v", res.Warnings)
	}
}

func TestSkipWarningReplaysUntilFixed(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.writeTicket(t, b, "not a ticket\n")
	f.commit(t, "seed")

	items, res, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(items) != 1 || len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], b+".md") {
		t.Fatalf("malformed ticket: items=%d warnings=%v", len(items), res.Warnings)
	}

	_, res, err = index.Load(f.store, f.repo, index.Options{})
	if err != nil || len(res.Warnings) != 1 {
		t.Fatalf("warning must replay while the file stays broken: warnings=%v err=%v", res.Warnings, err)
	}

	f.writeTicket(t, b, ticket(b, "KIRA-2", "fixed"))
	items, res, err = index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load after fix: %v", err)
	}
	if len(items) != 2 || len(res.Warnings) != 0 {
		t.Fatalf("fixed file must clear the warning: items=%d warnings=%v", len(items), res.Warnings)
	}
}

func TestCommitLinkMarkerNonDefaultBoard(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.writeTicket(t, b, ticket(b, "CORE-1", "other board"))
	f.commit(t, "seed")
	f.commitTrailer(t, "[[CORE-1]] cross-board fix", "unrelated: x")

	opts := trailerOpts()
	opts.BoardKeys = []string{"KIRA", "CORE"}

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, opts, false); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	links := mustCommitLinks(t, idx, b)
	if len(links) != 1 || links[0].Kind != index.LinkLinked {
		t.Fatalf("explicit [[CORE-1]] marker on a non-default board must link: %+v", links)
	}
}

func TestCommitLinkMarkerRetroactiveReclassify(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.writeTicket(t, b, ticket(b, "CORE-1", "other board"))
	f.commit(t, "seed")
	f.commitTrailer(t, "[[CORE-1]] explicit marker", "unrelated: x")

	off := trailerOpts()
	off.BoardKeys = []string{"KIRA", "CORE"}
	off.LinkMarkers = []datamodel.LinkMarker{datamodel.LinkMarkerTrailer}

	idx := open(t, f)
	if _, err := index.Refresh(f.store, f.repo, off, false); err != nil {
		t.Fatalf("EnsureFresh markers-off: %v", err)
	}
	if links := mustCommitLinks(t, idx, b); len(links) != 1 || links[0].Kind != index.LinkReferenced {
		t.Fatalf("subject markers off must demote to referenced: %+v", links)
	}

	on := trailerOpts()
	on.BoardKeys = []string{"KIRA", "CORE"}
	if _, err := index.Refresh(f.store, f.repo, on, false); err != nil {
		t.Fatalf("EnsureFresh markers-on: %v", err)
	}
	links := mustCommitLinks(t, idx, b)
	if len(links) != 1 || links[0].Kind != index.LinkLinked {
		t.Fatalf("config change must retroactively reclassify the marker: %+v", links)
	}
}

func TestCrashedRebuildLeavesEmptyDBRecovers(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "first"))
	f.commit(t, "seed")
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
		t.Fatalf("initial Load: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(f.store.CacheDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DELETE FROM items"); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	items, res, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load over schema-only empty db: %v", err)
	}
	if res.Action != "full" || len(items) != 1 {
		t.Fatalf("empty db with fresh meta must full-rebuild: action=%q items=%d want full/1", res.Action, len(items))
	}
}

func TestMismatchedIDNotShadowedByOwnRow(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, b, ticket(a, "KIRA-1", "v1"))
	f.commit(t, "seed")
	items, _, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil || len(items) != 1 || titleOf(items, a) != "v1" {
		t.Fatalf("initial Load: items=%d title=%q err=%v", len(items), titleOf(items, a), err)
	}

	f.writeTicket(t, b, ticket(a, "KIRA-1", "v2"))
	items, res, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load after edit: %v", err)
	}
	if titleOf(items, a) != "v2" || len(res.Warnings) != 0 {
		t.Fatalf("file shadowed by its own previous row: title=%q warnings=%v", titleOf(items, a), res.Warnings)
	}
}

func TestRenameWithEditRefreshes(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "before"))
	f.commit(t, "seed")
	if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
		t.Fatalf("initial Load: %v", err)
	}

	run(t, f.root, "git", "mv", ".kira/tickets/"+a+".md", ".kira/tickets/"+b+".md")
	f.writeTicket(t, b, ticket(a, "KIRA-1", "after"))
	f.commit(t, "rename and edit")

	items, res, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load after rename+edit: %v", err)
	}
	if len(items) != 1 || titleOf(items, a) != "after" || len(res.Warnings) != 0 {
		t.Fatalf("rename+edit served stale row: items=%d title=%q warnings=%v", len(items), titleOf(items, a), res.Warnings)
	}
}

func TestRefreshOrderIndependence(t *testing.T) {
	t.Parallel()
	const lo = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const hi = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	for _, tc := range []struct {
		name, canonical, claimer string
	}{
		{"deleted-sorts-first", lo, hi},
		{"claimer-sorts-first", hi, lo},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newRepo(t)
			f.writeTicket(t, tc.canonical, ticket(tc.canonical, "KIRA-1", "before"))
			f.commit(t, "seed")
			if _, _, err := index.Load(f.store, f.repo, index.Options{}); err != nil {
				t.Fatalf("initial Load: %v", err)
			}

			f.removeTicket(t, tc.canonical)
			f.writeTicket(t, tc.claimer, ticket(tc.canonical, "KIRA-1", "after"))
			f.commit(t, "swap holder")

			items, res, err := index.Load(f.store, f.repo, index.Options{})
			if err != nil {
				t.Fatalf("Load after swap: %v", err)
			}
			if len(items) != 1 || items[0].ID != tc.canonical || items[0].Title != "after" || len(res.Warnings) != 0 {
				t.Fatalf("delete+claim not order independent: items=%d id=%q title=%q warnings=%v",
					len(items), items[0].ID, items[0].Title, res.Warnings)
			}
		})
	}
}

func TestFullPrefersCanonicalFile(t *testing.T) {
	t.Parallel()
	const impostor = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const canonical = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, impostor, ticket(canonical, "KIRA-9", "impostor"))
	f.writeTicket(t, canonical, ticket(canonical, "KIRA-1", "canonical"))
	f.commit(t, "seed")

	items, res, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(items) != 1 || titleOf(items, canonical) != "canonical" {
		t.Fatalf("full() must index the canonical file even when the impostor sorts first: items=%d title=%q",
			len(items), titleOf(items, canonical))
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], impostor+".md") ||
		!strings.Contains(res.Warnings[0], canonical+".md") {
		t.Fatalf("skip note must land on the impostor and name both files: %v", res.Warnings)
	}
}

func TestSkippedDuplicateReindexedWhenWinnerDeleted(t *testing.T) {
	t.Parallel()
	const a = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	const b = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"
	f := newRepo(t)
	f.writeTicket(t, a, ticket(a, "KIRA-1", "winner"))
	f.writeTicket(t, b, ticket(a, "KIRA-2", "loser"))
	f.commit(t, "seed")
	items, res, err := index.Load(f.store, f.repo, index.Options{})
	if err != nil || len(items) != 1 || len(res.Warnings) != 1 {
		t.Fatalf("seed Load: items=%d warnings=%v err=%v", len(items), res.Warnings, err)
	}

	f.removeTicket(t, a)
	f.commit(t, "drop winner")
	items, res, err = index.Load(f.store, f.repo, index.Options{})
	if err != nil {
		t.Fatalf("Load after winner deleted: %v", err)
	}
	if len(items) != 1 || items[0].ID != a || items[0].Title != "loser" {
		t.Fatalf("loser must be indexed in the same run: items=%+v", items)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("stale duplicate warning must clear: %v", res.Warnings)
	}
}

func TestDirtyPathsSurviveToplevelSymlinkResolution(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	actual := filepath.Join(base, "real")
	if err := os.MkdirAll(actual, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := testutil.GitInit(actual); err != nil {
		t.Fatalf("git init: %v", err)
	}
	linked := filepath.Join(base, "linked")
	if err := os.Symlink(actual, linked); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(linked, ".kira", "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := repoFixture{root: linked, store: storage.New(linked), repo: gitx.Repo{Dir: linked}}

	const id = "01J8X7B1Q2W3E4R5T6Y7U8I9O0"
	f.writeTicket(t, id, ticket(id, "KIRA-1", "v1"))
	f.commit(t, "one")

	open(t, f)
	if _, err := index.Refresh(f.store, f.repo, index.Options{}, false); err != nil {
		t.Fatalf("initial Refresh: %v", err)
	}

	f.writeTicket(t, id, ticket(id, "KIRA-1", "v2"))
	res, err := index.Refresh(f.store, f.repo, index.Options{}, false)
	if err != nil {
		t.Fatalf("Refresh with a dirty edit through a symlinked toplevel: %v", err)
	}
	if res.Action != "incremental" {
		t.Fatalf("action=%q want incremental", res.Action)
	}
}
