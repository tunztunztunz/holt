package gitx

import (
	"bytes"
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

// MergeTree dry-runs merging theirs into ours, entirely in memory. It returns
// the OID of the merged tree (feed it to CommitTree to chain cumulative probes)
// and the conflicted paths; empty conflicts means the merge is clean.
func MergeTree(repoRoot, ours, theirs string) (tree string, conflicts []string, err error) {
	out, code, err := runCode("", "-C", repoRoot, "merge-tree", "--write-tree", "--name-only", ours, theirs)
	if err != nil {
		return "", nil, err
	}
	switch code {
	case 0:
		return strings.TrimSpace(out), nil, nil // clean; whole output is the tree OID
	case 1:
		// stdout is: <tree-oid>\n<conflicted path>\n...\n[blank]\n<informational msgs>
		// Line 0 is the merged tree OID; then the conflicted file names, one per
		// line, terminated by a blank line. Everything after the blank is human
		// chatter (Auto-merging…, CONFLICT (add/add):…) — stop there.
		lines := strings.Split(strings.TrimSpace(out), "\n")
		tree = strings.TrimSpace(lines[0])
		for _, ln := range lines[1:] {
			if strings.TrimSpace(ln) == "" {
				break // end of conflicted-file list; messages follow
			}
			conflicts = append(conflicts, ln)
		}
		if len(conflicts) == 0 {
			conflicts = []string{"(conflict)"}
		}
		return tree, conflicts, nil
	default:
		return "", nil, fmt.Errorf("merge-tree failed (code %d)", code)
	}
}

// CommitTree writes a commit object with the given tree and parent and returns
// its OID. The commits are dangling and get garbage-collected.
func CommitTree(repoRoot, tree, parent string) (string, error) {
	return run("", "-C", repoRoot, "commit-tree", tree, "-p", parent, "-m", "holt harvest probe")
}

// WorktreeAddFrom creates a worktree at path on a NEW branch based at start.
// git -C repo worktree add -b <branch> <path> <start>
func WorktreeAddFrom(repoRoot, path, branch, start string) error {
	_, err := run("", "-C", repoRoot, "worktree", "add", "-b", branch, path, start)
	return err
}

// Merge merges branch into the checkout at worktree (a real merge commit, no fast-forward
// so the harvest history is legible). Returns an error on conflict; the caller
// inspects the working tree and stops for hand-resolution.
func Merge(worktree, branch string) error {
	_, err := run(worktree, "merge", "--no-ff", branch)
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

func gitExec(dir string, args ...string) (stdout, stderr string, code int, err error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		return string(out), errBuf.String(), ee.ExitCode(), nil
	}
	if err != nil {
		return "", errBuf.String(), -1, err // couldn't start git
	}
	return string(out), errBuf.String(), 0, nil
}

func run(dir string, args ...string) (string, error) {
	out, stderr, code, err := gitExec(dir, args...)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("%s", strings.TrimSpace(stderr))
	}
	return strings.TrimSpace(out), nil
}

func runCode(dir string, args ...string) (string, int, error) {
	out, _, code, err := gitExec(dir, args...)
	return out, code, err
}
