package guard

import "github.com/tunztunztunz/acre/internal/gitx"

type Guard interface {
	Name() string
	Check(worktree, branch, defaultBranch string) (string, error)
}

// Registry maps profile guard names to implementations.
var Registry = map[string]Guard{
	"uncommitted": uncommitted{},
	"unmerged":    unmerged{},
	"unpushed":    unpushed{},
	"stashes":     stashes{},
}

type uncommitted struct{}

func (uncommitted) Name() string { return "uncommitted" }
func (uncommitted) Check(worktree, branch, defaultBranch string) (string, error) {
	st, err := gitx.WorktreeStatus(worktree)
	if err != nil {
		return "", err
	}
	if st.Dirty {
		return "uncommitted changes", nil
	}
	return "", nil
}

type unmerged struct{}

func (unmerged) Name() string { return "unmerged" }
func (unmerged) Check(worktree, branch, defaultBranch string) (string, error) {
	out, err := gitx.LogRange(worktree, defaultBranch, branch)
	if err != nil {
		return "", err
	}
	if out != "" {
		return "has commits not in " + defaultBranch, nil
	}
	return "", nil
}

type unpushed struct{}

func (unpushed) Name() string { return "unpushed" }
func (unpushed) Check(worktree, _, _ string) (string, error) {
	out, err := gitx.LogRange(worktree, "@{upstream}", "HEAD")
	if err != nil {
		return "", err
	}
	if out != "" {
		return "has unpushed commits", nil
	}
	return "", nil
}

type stashes struct{}

func (stashes) Name() string { return "stashes" }
func (stashes) Check(worktree, _, _ string) (string, error) {
	out, err := gitx.StashList(worktree)
	if err != nil {
		return "", err
	}
	if out != "" {
		return "has changes in git stash", nil
	}
	return "", nil
}
