package gitx

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Worktree struct {
	Path   string
	Branch string
}

// WorktreeList returns the repo's worktrees as reported by git.
func WorktreeList(repoRoot string) ([]Worktree, error) {
	out, err := run("", "-C", repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var (
		list []Worktree
		cur  Worktree
	)

	for line := range strings.SplitSeq(out, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			cur = Worktree{Path: strings.TrimPrefix(line, "worktree ")}

		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")

		case line == "":
			if cur.Path != "" {
				list = append(list, cur)
			}
			cur = Worktree{}
		}
	}
	if cur.Path != "" {
		list = append(list, cur)
	}

	return list, nil
}

// WorktreeAdd creates a worktree; newBranch=false attaches an existing branch.
func WorktreeAdd(repoRoot, path, branch string, newBranch bool) error {
	args := []string{"-C", repoRoot, "worktree", "add"}
	if newBranch {
		args = append(args, "-b", branch, path)
	} else {
		args = append(args, path, branch)
	}
	_, err := run("", args...)
	return err
}

// RepoRoot returns the main worktree path, from anywhere inside the repo.
func RepoRoot() (string, error) {
	common, err := run("", "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}

	return filepath.Dir(common), nil
}

// DefaultBranch returns the repo's default branch (origin/HEAD), or "main".
func DefaultBranch() string {
	ref, err := run("", "symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "main"
	}
	return strings.TrimPrefix(ref, "refs/remotes/origin/")
}

// CurrentBranch returns the branch checked out in dir, or "" if HEAD is detached.
func CurrentBranch(dir string) string {
	ref, err := run("", "-C", dir, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return ""
	}
	return ref
}

// LogRange LogRange returns commits in `to` not in `from` (`from..to`); empty == none.
// Runs inside the worktree, like WorktreeStatus.
func LogRange(worktree, from, to string) (string, error) {
	return run(worktree, "log", "--oneline", from+".."+to)
}

// WorktreeRemove runs `git worktree remove [--force] <path>` from the repo.
func WorktreeRemove(repoRoot, path string, force bool) error {
	args := []string{"-C", repoRoot, "worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}
	_, err := run("", args...)
	return err
}

// WorktreePrune drops administrative files for worktrees removed from disk.
func WorktreePrune(repoRoot string) error {
	_, err := run("", "-C", repoRoot, "worktree", "prune")
	return err
}

// BranchDelete force-deletes a branch (`branch -D`).
func BranchDelete(repoRoot, branch string) error {
	_, err := run("", "-C", repoRoot, "branch", "-D", branch)
	return err
}

// StashList returns the repo's stash entries ("" == none). Stashes are
// repo-global (refs/stash), so this is the same list from any worktree.
// It warns whenever the repo has any stash, not just this branch's.
func StashList(worktree string) (string, error) {
	return run(worktree, "stash", "list")
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := errors.AsType[*exec.ExitError](err); ok {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
