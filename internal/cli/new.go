package cli

import (
	"github.com/tunztunztunz/holt/internal/config"
	"github.com/tunztunztunz/holt/internal/gitx"
	"github.com/tunztunztunz/holt/internal/state"
)

// newCmd creates a worktree for a branch: it attaches an existing local branch
// or creates a new one off HEAD, allocates a free port, provisions the tree, and
// prints the new path so the holt() shell wrapper cds into it. With no branch
// arg it shows a picker of local branches not already in a worktree.
type newCmd struct {
	Branch string `arg:"" optional:"" help:"Branch to attach (if it exists) or create. Omit to pick."`
}

func (c *newCmd) Run(root Root, profile *config.Profile, store *state.Store) error {
	repo := string(root)

	if err := profile.Validate(); err != nil {
		return Exitf(ExitUsage, "%v", err)
	}

	branch, err := pickBranch(c.Branch, repo)
	if err != nil {
		return err
	}
	newBranch := !gitx.BranchExists(repo, branch)

	v, err := prepareWorktree(repo, branch, profile, store)
	if err != nil {
		return err
	}
	if v.Port, err = allocatePort(profile, v.SiteName, store); err != nil {
		return err
	}

	// Attach the existing branch, or create it off the current HEAD.
	if err = gitx.WorktreeAdd(repo, v.Worktree, branch, newBranch); err != nil {
		return Exitf(ExitConflict, "%v", err)
	}

	rec := provisioningRecord(v, branch, baseForNew(repo, newBranch, profile))
	store.Worktrees[v.SiteName] = rec
	if err = state.Save(repo, store); err != nil {
		return Exitf(ExitRuntime, "%v", err)
	}

	if err = finalizeWorktree(repo, v, profile, rec, store, true); err != nil {
		return err
	}

	resultf("%s\n", v.Worktree)
	return nil
}

// baseForNew records where the worktree forks from: a freshly created branch
// forks off the current HEAD; an attached existing branch has no obvious fork
// point, so fall back to the shared base resolution.
func baseForNew(repo string, newBranch bool, profile *config.Profile) string {
	if newBranch {
		return gitx.CurrentBranch(repo)
	}
	return baseFor("", profile)
}
