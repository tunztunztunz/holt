package gitx

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultBranch returns the repo's default branch (origin/HEAD), or "main".
func DefaultBranch() string {
	ref, err := run("", "symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "main"
	}
	return strings.TrimPrefix(ref, "refs/remotes/origin/")
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

// RepoRoot returns the main worktree path, from anywhere inside the repo.
func RepoRoot() (string, error) {
	common, err := run("", "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}

	return filepath.Dir(common), nil
}
