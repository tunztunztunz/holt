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

// WorktreeList
func WorktreeList(repoRoot string) ([]Worktree, error) {
	out, err := run("", "-C", repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var (
		list []Worktree
		cur  Worktree
	)

	for _, line := range strings.Split(out, "\n") {
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
