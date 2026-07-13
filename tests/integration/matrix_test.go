package integration

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/merge"
)

const (
	tsBase  = "2026-01-01T00:00:00Z"
	tsEarly = "2026-03-01T00:00:00Z"
	tsLate  = "2026-03-02T00:00:00Z"

	ulidX    = "01J8X8Q7RZTN5Y3VXW2A9K4E70"
	ulidY    = "01J8X8Q7RZTN5Y3VXW2A9K4E71"
	ulidOurs = "01J8X8Q7RZTN5Y3VXW2A9K4E7A"
	ulidThem = "01J8X8Q7RZTN5Y3VXW2A9K4E7B"
)

func matrixItem(ulid, number, title string) *datamodel.Item {
	return &datamodel.Item{
		ID:        ulid,
		Number:    number,
		Aliases:   []string{},
		Type:      datamodel.TypeTicket,
		Title:     title,
		State:     "TODO",
		Labels:    []string{},
		BlockedBy: []string{},
		Created:   tsBase,
		Updated:   tsBase,
		Body:      "## Description\n\nbody\n",
	}
}

func strPtr(s string) *string { return &s }

func ticketRel(ulid string) string { return ".kira/tickets/" + ulid + ".md" }

type world struct {
	t    *testing.T
	root string
	repo gitx.Repo
}

func (w *world) abs(ulid string) string {
	return filepath.Join(w.root, filepath.FromSlash(ticketRel(ulid)))
}

func (w *world) write(it *datamodel.Item) {
	w.t.Helper()
	abs := w.abs(it.ID)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(codec.Serialize(it)), 0o644); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) commit(msg string) {
	w.t.Helper()
	if _, err := w.repo.Output("add", "-A"); err != nil {
		w.t.Fatalf("git add: %v", err)
	}
	if _, err := w.repo.Output("commit", "-m", msg); err != nil {
		w.t.Fatalf("git commit: %v", err)
	}
}

func (w *world) item(ulid string) *datamodel.Item {
	w.t.Helper()
	data, err := os.ReadFile(w.abs(ulid))
	if err != nil {
		w.t.Fatalf("read %s: %v", ulid, err)
	}
	it, err := codec.Parse(string(data))
	if err != nil {
		w.t.Fatalf("parse %s: %v", ulid, err)
	}
	return it
}

type recoverable struct {
	ulid  string
	value string
}

type scenario struct {
	name      string
	conflicts bool
	loser     recoverable
	seed      func(w *world)
	ours      func(w *world)
	theirs    func(w *world)
	assert    func(t *testing.T, w *world)
}

func matrixScenarios() []scenario {
	return []scenario{
		{
			name:      "different_tickets",
			conflicts: false,
			seed: func(w *world) {
				w.write(matrixItem(ulidX, "KIRA-1", "X"))
				w.write(matrixItem(ulidY, "KIRA-2", "Y"))
			},
			ours: func(w *world) {
				it := matrixItem(ulidX, "KIRA-1", "X")
				it.Owner, it.Updated = strPtr("alice"), tsLate
				w.write(it)
			},
			theirs: func(w *world) {
				it := matrixItem(ulidY, "KIRA-2", "Y")
				it.Owner, it.Updated = strPtr("bob"), tsLate
				w.write(it)
			},
			assert: func(t *testing.T, w *world) {
				if got := w.item(ulidX).Owner; got == nil || *got != "alice" {
					t.Fatalf("X owner = %v, want alice", got)
				}
				if got := w.item(ulidY).Owner; got == nil || *got != "bob" {
					t.Fatalf("Y owner = %v, want bob", got)
				}
			},
		},
		{
			name:      "different_fields_same_ticket",
			conflicts: true,
			seed:      func(w *world) { w.write(matrixItem(ulidX, "KIRA-1", "X")) },
			ours: func(w *world) {
				it := matrixItem(ulidX, "KIRA-1", "X")
				it.Owner, it.Updated = strPtr("alice"), tsLate
				w.write(it)
			},
			theirs: func(w *world) {
				it := matrixItem(ulidX, "KIRA-1", "X")
				it.Priority, it.Updated = strPtr("P1"), tsEarly
				w.write(it)
			},
			assert: func(t *testing.T, w *world) {
				it := w.item(ulidX)
				if it.Owner == nil || *it.Owner != "alice" {
					t.Fatalf("owner = %v, want alice", it.Owner)
				}
				if it.Priority == nil || *it.Priority != "P1" {
					t.Fatalf("priority = %v, want P1", it.Priority)
				}
				if it.Updated != tsLate {
					t.Fatalf("updated = %s, want %s (max of both sides)", it.Updated, tsLate)
				}
			},
		},
		{
			name:      "same_field_same_ticket_lww",
			conflicts: true,
			loser:     recoverable{ulid: ulidX, value: "DONE"},
			seed:      func(w *world) { w.write(matrixItem(ulidX, "KIRA-1", "X")) },
			ours: func(w *world) {
				it := matrixItem(ulidX, "KIRA-1", "X")
				it.State, it.Updated = "REVIEW", tsLate
				w.write(it)
			},
			theirs: func(w *world) {
				it := matrixItem(ulidX, "KIRA-1", "X")
				it.State, it.Updated = "DONE", tsEarly
				w.write(it)
			},
			assert: func(t *testing.T, w *world) {
				if s := w.item(ulidX).State; s != "REVIEW" {
					t.Fatalf("state = %s, want REVIEW (side with later updated wins)", s)
				}
			},
		},
		{
			name:      "same_field_exact_tie_remote_wins",
			conflicts: true,
			seed:      func(w *world) { w.write(matrixItem(ulidX, "KIRA-1", "X")) },
			ours: func(w *world) {
				it := matrixItem(ulidX, "KIRA-1", "X")
				it.State, it.Updated = "REVIEW", tsLate
				w.write(it)
			},
			theirs: func(w *world) {
				it := matrixItem(ulidX, "KIRA-1", "X")
				it.State, it.Updated = "DONE", tsLate
				w.write(it)
			},
			assert: func(t *testing.T, w *world) {
				if s := w.item(ulidX).State; s != "DONE" {
					t.Fatalf("tie state = %s, want DONE (remote/incoming side wins on both paths)", s)
				}
			},
		},
		{
			name:      "concurrent_comments",
			conflicts: true,
			seed:      func(w *world) { w.write(matrixItem(ulidX, "KIRA-1", "X")) },
			ours: func(w *world) {
				it := matrixItem(ulidX, "KIRA-1", "X")
				it.Body = codec.AppendComment(it.Body, datamodel.Comment{ID: "01BBB", Author: "ours", Ts: tsLate, Body: "from ours"})
				w.write(it)
			},
			theirs: func(w *world) {
				it := matrixItem(ulidX, "KIRA-1", "X")
				it.Body = codec.AppendComment(it.Body, datamodel.Comment{ID: "01AAA", Author: "them", Ts: tsEarly, Body: "from theirs"})
				w.write(it)
			},
			assert: func(t *testing.T, w *world) {
				cs := codec.ParseComments(w.item(ulidX).Body)
				if len(cs) != 2 || cs[0].ID != "01AAA" || cs[1].ID != "01BBB" {
					t.Fatalf("comments = %+v, want both present, ts-sorted [01AAA, 01BBB]", cs)
				}
			},
		},
		{
			name:      "concurrent_creates_number_collision",
			conflicts: false,
			seed:      func(w *world) { w.write(matrixItem(ulidX, "KIRA-1", "X")) },
			ours:      func(w *world) { w.write(matrixItem(ulidOurs, "KIRA-2", "Ours new")) },
			theirs:    func(w *world) { w.write(matrixItem(ulidThem, "KIRA-2", "Theirs new")) },
			assert: func(t *testing.T, w *world) {
				if n := w.item(ulidOurs).Number; n != "KIRA-2" {
					t.Fatalf("earlier ULID number = %s, want KIRA-2 (kept)", n)
				}
				later := w.item(ulidThem)
				if later.Number != "KIRA-3" {
					t.Fatalf("later ULID number = %s, want KIRA-3 (renumbered)", later.Number)
				}
				if !slices.Contains(later.Aliases, "KIRA-2") {
					t.Fatalf("aliases = %v, want retired KIRA-2 retained as alias", later.Aliases)
				}
			},
		},
	}
}

func isolateGit(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("EDITOR", "true")
}

func configUser(t *testing.T, repo gitx.Repo) {
	t.Helper()
	for _, kv := range [][2]string{{"user.email", "test@example.com"}, {"user.name", "tester"}} {
		if _, err := repo.Output("config", kv[0], kv[1]); err != nil {
			t.Fatalf("git config %s: %v", kv[0], err)
		}
	}
}

func bareRemote(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := (gitx.Repo{Dir: dir}).Output("init", "--bare", "-b", "main"); err != nil {
		t.Fatalf("init bare: %v", err)
	}
	return dir
}

func cloneWorld(t *testing.T, bare string) *world {
	t.Helper()
	dir := t.TempDir()
	if _, err := (gitx.Repo{Dir: bare}).Output("clone", bare, dir); err != nil {
		t.Fatalf("git clone: %v", err)
	}
	repo := gitx.Repo{Dir: dir}
	configUser(t, repo)
	return &world{t: t, root: dir, repo: repo}
}

func setManual(t *testing.T, w *world) {
	t.Helper()
	setMergePolicyManual(t, w.root)
	w.commit("kira: merge.policy manual")
}

func runSyncPath(t *testing.T, sc scenario, policy datamodel.MergePolicy) (*world, *core.Store, *datamodel.Config, error) {
	isolateGit(t)
	bare := bareRemote(t)

	seed := cloneWorld(t, bare)
	initStore(t, seed.root)
	if policy == datamodel.MergeManual {
		setManual(t, seed)
	}
	sc.seed(seed)
	seed.commit("base")
	if _, err := seed.repo.Output("push", "-u", "origin", "main"); err != nil {
		t.Fatalf("push seed: %v", err)
	}

	ours := cloneWorld(t, bare)
	them := cloneWorld(t, bare)

	sc.theirs(them)
	them.commit("theirs")
	if _, err := them.repo.Output("push", "origin", "main"); err != nil {
		t.Fatalf("push theirs: %v", err)
	}

	sc.ours(ours)
	ours.commit("ours")

	s, cfg := discoverStore(t, ours.root)
	_, syncErr := s.Sync(cfg, core.SyncOpts{}, nil)
	return ours, s, cfg, syncErr
}

func runDriverPath(t *testing.T, sc scenario, policy datamodel.MergePolicy) (*world, *core.Store, *datamodel.Config, error) {
	root := initGitRepo(t)
	initStore(t, root)
	w := &world{t: t, root: root, repo: gitx.Repo{Dir: root}}
	if policy == datamodel.MergeManual {
		setManual(t, w)
	}
	s, cfg := discoverStore(t, root)

	main, err := w.repo.Output("branch", "--show-current")
	if err != nil {
		t.Fatalf("branch --show-current: %v", err)
	}

	sc.seed(w)
	w.commit("base")
	if _, err := w.repo.Output("checkout", "-b", "other"); err != nil {
		t.Fatalf("checkout -b other: %v", err)
	}
	sc.theirs(w)
	w.commit("theirs")
	if _, err := w.repo.Output("checkout", main); err != nil {
		t.Fatalf("checkout %s: %v", main, err)
	}
	sc.ours(w)
	w.commit("ours")

	registerDriver(t, root)
	if _, err := w.repo.Output("merge", "other"); err != nil {
		return w, s, cfg, err
	}
	if _, err := s.Reconcile(cfg); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	return w, s, cfg, nil
}

func discoverStore(t *testing.T, root string) (*core.Store, *datamodel.Config) {
	t.Helper()
	s, err := coreDiscover(root)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	return s, cfg
}

type mergePath struct {
	name       string
	run        func(*testing.T, scenario, datamodel.MergePolicy) (*world, *core.Store, *datamodel.Config, error)
	onSurfaced func(t *testing.T, w *world, s *core.Store, cfg *datamodel.Config, err error) (proceed bool)
}

func TestMergeRegressionMatrix(t *testing.T) {
	paths := []mergePath{
		{"sync", runSyncPath, onSyncSurfaced},
		{"driver", runDriverPath, onDriverSurfaced},
	}
	policies := []datamodel.MergePolicy{datamodel.MergeAuto, datamodel.MergeManual}

	for _, sc := range matrixScenarios() {
		t.Run(sc.name, func(t *testing.T) {
			for _, pol := range policies {
				t.Run(string(pol), func(t *testing.T) {
					for _, p := range paths {
						t.Run(p.name, func(t *testing.T) {
							w, s, cfg, err := p.run(t, sc, pol)
							if pol == datamodel.MergeManual && sc.conflicts {
								if !p.onSurfaced(t, w, s, cfg, err) {
									return
								}
							} else if err != nil {
								t.Fatalf("unexpected error: %v", err)
							}
							assertNoConflictMarkers(t, w.root)
							assertNoUnmerged(t, w.repo)
							sc.assert(t, w)
							if sc.loser.value != "" {
								assertRecoverable(t, w.repo, ticketRel(sc.loser.ulid), sc.loser.value)
							}
							assertIdempotent(t, w, s, cfg)
						})
					}
				})
			}
		})
	}
}

func onSyncSurfaced(t *testing.T, w *world, _ *core.Store, _ *datamodel.Config, err error) bool {
	t.Helper()
	if err == nil {
		t.Fatal("merge.policy manual must not silently auto-resolve on the sync path")
	}
	if w.repo.RebaseInProgress() {
		t.Fatal("manual-policy sync must not strand a rebase in progress")
	}
	assertNoConflictMarkers(t, w.root)
	return false
}

func onDriverSurfaced(t *testing.T, w *world, s *core.Store, cfg *datamodel.Config, err error) bool {
	t.Helper()
	if err == nil {
		t.Fatal("merge.policy manual must surface a raw conflict on the driver path")
	}
	assertUnmergedPresent(t, w.repo)
	assertMarkersPresent(t, w.root)
	resolveAndComplete(t, w, s, cfg)
	return true
}

func TestByteIdenticalSyncVsDriver(t *testing.T) {
	for _, sc := range matrixScenarios() {
		t.Run(sc.name, func(t *testing.T) {
			syncW, _, _, err := runSyncPath(t, sc, datamodel.MergeAuto)
			if err != nil {
				t.Fatalf("sync path: %v", err)
			}
			driverW, _, _, err := runDriverPath(t, sc, datamodel.MergeAuto)
			if err != nil {
				t.Fatalf("driver path: %v", err)
			}
			assertSameTickets(t, ticketBytes(t, syncW.root), ticketBytes(t, driverW.root))
			assertDriverMatchesEngine(t, driverW)
		})
	}
}

func showBlob(repo gitx.Repo, ref, rel string) (string, bool) {
	out, err := repo.OutputRaw("show", ref+":"+rel)
	if err != nil {
		return "", false
	}
	return out, true
}

func assertDriverMatchesEngine(t *testing.T, w *world) {
	t.Helper()
	mergeCommit, err := w.repo.Output("rev-list", "--merges", "-1", "HEAD")
	if err != nil || mergeCommit == "" {
		t.Fatalf("locate merge commit: %v (out=%q)", err, mergeCommit)
	}
	base, err := w.repo.Output("merge-base", mergeCommit+"^1", mergeCommit+"^2")
	if err != nil {
		t.Fatalf("merge-base: %v", err)
	}
	listed, err := w.repo.Output("ls-tree", "-r", "--name-only", mergeCommit, "--", ".kira/tickets")
	if err != nil {
		t.Fatalf("ls-tree: %v", err)
	}
	for _, rel := range strings.Split(strings.TrimSpace(listed), "\n") {
		if rel == "" {
			continue
		}
		baseBlob, okB := showBlob(w.repo, base, rel)
		oursBlob, okO := showBlob(w.repo, mergeCommit+"^1", rel)
		theirsBlob, okT := showBlob(w.repo, mergeCommit+"^2", rel)
		mergedBlob, okM := showBlob(w.repo, mergeCommit, rel)
		if !okB || !okO || !okT || !okM {
			continue
		}
		bi, _ := codec.Parse(baseBlob)
		oi, _ := codec.Parse(oursBlob)
		ti, _ := codec.Parse(theirsBlob)
		want := codec.Serialize(merge.Merge(bi, oi, ti, merge.Theirs, gitMerger).Item)
		if mergedBlob != want {
			t.Fatalf("%s: driver merge output is not byte-identical to direct merge.Merge:\n--- driver ---\n%s\n--- engine ---\n%s", rel, mergedBlob, want)
		}
	}
}

func TestTemplateFileStillConflictsRaw(t *testing.T) {
	for _, pol := range []datamodel.MergePolicy{datamodel.MergeAuto, datamodel.MergeManual} {
		t.Run(string(pol), func(t *testing.T) {
			root := initGitRepo(t)
			initStore(t, root)
			w := &world{t: t, root: root, repo: gitx.Repo{Dir: root}}
			if pol == datamodel.MergeManual {
				setManual(t, w)
			}
			registerDriver(t, root)

			rel := filepath.FromSlash(".kira/templates/ticket.md")
			abs := filepath.Join(root, rel)
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				t.Fatal(err)
			}
			writeTemplate := func(mid, msg string) {
				if err := os.WriteFile(abs, []byte("line1\n"+mid+"\nline3\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				w.commit(msg)
			}
			writeTemplate("shared", "template base")
			main, _ := w.repo.Output("branch", "--show-current")
			if _, err := w.repo.Output("checkout", "-b", "other"); err != nil {
				t.Fatalf("checkout -b other: %v", err)
			}
			writeTemplate("THEIRS", "theirs template")
			if _, err := w.repo.Output("checkout", main); err != nil {
				t.Fatalf("checkout %s: %v", main, err)
			}
			writeTemplate("OURS", "ours template")

			if _, err := w.repo.Output("merge", "other"); err == nil {
				t.Fatal("template conflict must not auto-merge: the merge=kira glob would be over-matching")
			}
			data, _ := os.ReadFile(abs)
			if !strings.Contains(string(data), "<<<<<<<") {
				t.Fatalf("template merge must leave raw git conflict markers, got:\n%s", data)
			}
		})
	}
}

func assertMarkersPresent(t *testing.T, root string) {
	t.Helper()
	for _, body := range ticketBytes(t, root) {
		if strings.Contains(body, "<<<<<<<") {
			return
		}
	}
	t.Fatal("manual-policy driver merge must leave raw conflict markers, found none")
}

func resolveAndComplete(t *testing.T, w *world, s *core.Store, cfg *datamodel.Config) {
	t.Helper()
	if _, err := s.Resolve(nil, false); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if _, err := w.repo.Output("commit", "--no-edit"); err != nil {
		t.Fatalf("commit resolved merge: %v", err)
	}
	if _, err := s.Reconcile(cfg); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
}

func assertNoConflictMarkers(t *testing.T, root string) {
	t.Helper()
	for name, body := range ticketBytes(t, root) {
		if strings.Contains(body, "<<<<<<<") || strings.Contains(body, ">>>>>>>") {
			t.Fatalf("%s contains conflict markers:\n%s", name, body)
		}
	}
}

func assertUnmergedPresent(t *testing.T, repo gitx.Repo) {
	t.Helper()
	unmerged, err := repo.UnmergedPaths()
	if err != nil {
		t.Fatalf("unmerged paths: %v", err)
	}
	if len(unmerged) == 0 {
		t.Fatal("manual-policy driver merge must leave the ticket path unmerged in the index")
	}
}

func assertNoUnmerged(t *testing.T, repo gitx.Repo) {
	t.Helper()
	unmerged, err := repo.UnmergedPaths()
	if err != nil {
		t.Fatalf("unmerged paths: %v", err)
	}
	if len(unmerged) > 0 {
		t.Fatalf("unmerged paths remain: %v", unmerged)
	}
}

func assertRecoverable(t *testing.T, repo gitx.Repo, ticket, value string) {
	t.Helper()
	out, err := repo.Output("log", "--all", "-p", "--", ticket)
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if !strings.Contains(out, value) {
		t.Fatalf("losing value %q not recoverable from any commit reachable for %s", value, ticket)
	}
}

func assertIdempotent(t *testing.T, w *world, s *core.Store, cfg *datamodel.Config) {
	t.Helper()
	before := ticketBytes(t, w.root)
	if res, err := s.Resolve(nil, false); err != nil {
		t.Fatalf("re-resolve: %v", err)
	} else if len(res.Resolved) != 0 {
		t.Fatalf("re-resolve mutated %d already-merged items", len(res.Resolved))
	}
	if res, err := s.Reconcile(cfg); err != nil {
		t.Fatalf("re-reconcile: %v", err)
	} else if len(res.Renumbered) != 0 {
		t.Fatalf("re-reconcile renumbered %d already-stable items", len(res.Renumbered))
	}
	if !maps.Equal(before, ticketBytes(t, w.root)) {
		t.Fatal("re-running resolve/reconcile changed ticket files: merge output is not a fixpoint")
	}
}

func ticketBytes(t *testing.T, root string) map[string]string {
	t.Helper()
	dir := filepath.Join(root, ".kira", "tickets")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read tickets dir: %v", err)
	}
	out := map[string]string{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = string(data)
	}
	return out
}

func assertSameTickets(t *testing.T, sync, driver map[string]string) {
	t.Helper()
	if maps.Equal(sync, driver) {
		return
	}
	all := map[string]bool{}
	for k := range sync {
		all[k] = true
	}
	for k := range driver {
		all[k] = true
	}
	for k := range all {
		if sync[k] != driver[k] {
			t.Fatalf("ticket %s differs between paths:\n--- sync ---\n%s\n--- driver ---\n%s", k, sync[k], driver[k])
		}
	}
}
