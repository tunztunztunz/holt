package gitx

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// mustGit runs a git command in repo and fails the test on error.
func mustGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	out, err := run(repo, args...)
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return out
}

func writeFile(t *testing.T, repo, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// conflictRepo builds a repo with a "base" branch plus two branches that change
// the same line of file.txt incompatibly (feat-a, feat-b) and one that touches a
// different file entirely (feat-c). Returns the repo path.
func conflictRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	mustGit(t, repo, "init")
	mustGit(t, repo, "config", "user.email", "t@example.com")
	mustGit(t, repo, "config", "user.name", "Test")
	mustGit(t, repo, "config", "commit.gpgsign", "false")

	writeFile(t, repo, "file.txt", "1\n2\n")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-m", "base")
	mustGit(t, repo, "branch", "-M", "base")

	mustGit(t, repo, "checkout", "-b", "feat-a")
	writeFile(t, repo, "file.txt", "A\n2\n")
	mustGit(t, repo, "commit", "-am", "a")

	mustGit(t, repo, "checkout", "base")
	mustGit(t, repo, "checkout", "-b", "feat-b")
	writeFile(t, repo, "file.txt", "B\n2\n")
	mustGit(t, repo, "commit", "-am", "b")

	mustGit(t, repo, "checkout", "base")
	mustGit(t, repo, "checkout", "-b", "feat-c")
	writeFile(t, repo, "other.txt", "x\n")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-m", "c")

	return repo
}

func TestMergeTree(t *testing.T) {
	repo := conflictRepo(t)

	t.Run("clean merge returns a tree and no conflicts", func(t *testing.T) {
		tree, conflicts, err := MergeTree(repo, "base", "feat-c")
		if err != nil {
			t.Fatalf("MergeTree: %v", err)
		}
		if len(conflicts) != 0 {
			t.Errorf("conflicts = %v, want none", conflicts)
		}
		if tree == "" {
			t.Error("want a merged tree OID, got empty")
		}
	})

	t.Run("conflicting merge names the conflicted file", func(t *testing.T) {
		tree, conflicts, err := MergeTree(repo, "feat-a", "feat-b")
		if err != nil {
			t.Fatalf("MergeTree: %v", err)
		}
		if tree == "" {
			t.Error("want a merged tree OID even on conflict, got empty")
		}
		if !slices.Contains(conflicts, "file.txt") {
			t.Errorf("conflicts = %v, want it to contain %q", conflicts, "file.txt")
		}
	})
}
