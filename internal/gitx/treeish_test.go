package gitx

import "testing"

func TestNumstatNoIndexSurfacesRealErrors(t *testing.T) {
	repo := Repo{Dir: "/nonexistent-kira-numstat-dir"}
	_, _, err := repo.NumstatNoIndex("a", "b")
	if err == nil {
		t.Fatalf("want error when git cannot run, got nil")
	}
}

func TestFlagShapedRefsAreRejected(t *testing.T) {
	dir := t.TempDir()
	gitRun(t, dir, "init", "-b", "main")
	gitRun(t, dir, "config", "user.name", "t")
	gitRun(t, dir, "config", "user.email", "t@e.c")
	gitRun(t, dir, "config", "commit.gpgsign", "false")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "base")
	repo := Repo{Dir: dir}

	if sha, err := repo.ResolveTreeish("main"); err != nil || sha == "" {
		t.Fatalf("ResolveTreeish(main) = %q, %v", sha, err)
	}
	if commits, err := repo.RevListSince("main", "2000-01-01"); err != nil || len(commits) != 1 {
		t.Fatalf("RevListSince(main) = %v, %v", commits, err)
	}
	if sha, err := repo.ResolveAtDate("2030-01-01", "main"); err != nil || sha == "" {
		t.Fatalf("ResolveAtDate(main) = %q, %v", sha, err)
	}

	if _, err := repo.ResolveTreeish("--all"); err == nil {
		t.Error("ResolveTreeish treated --all as a flag")
	}
	if commits, err := repo.RevListSince("--all", "2000-01-01"); err == nil {
		t.Errorf("RevListSince treated --all as a flag: %d commits", len(commits))
	}
	if _, err := repo.ResolveAtDate("2030-01-01", "--all"); err == nil {
		t.Error("ResolveAtDate treated --all as a flag")
	}
	if err := repo.Checkout("-f"); err == nil {
		t.Error("Checkout treated -f as a flag")
	}
	if err := repo.CheckoutNew("--track"); err == nil {
		t.Error("CheckoutNew accepted a flag-shaped branch")
	}
	if err := repo.Pull("--dry-run"); err == nil {
		t.Error("Pull accepted a flag-shaped remote")
	}
	if err := repo.Push("--dry-run"); err == nil {
		t.Error("Push accepted a flag-shaped remote")
	}
}
