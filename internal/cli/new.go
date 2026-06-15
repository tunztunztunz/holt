package cli

import (
	"github.com/tunztunztunz/holt/internal/config"
	"github.com/tunztunztunz/holt/internal/gitx"
	"github.com/tunztunztunz/holt/internal/state"
)

// newCmd creates a worktree for a new branch: it resolves per-tree vars,
// allocates a free port, provisions the tree (copy/link/env files + setup
// commands), records it in state, then prints the new path so the holt() shell
// wrapper cds into it.
type newCmd struct {
	Branch string `arg:"" help:"Branch to create and check out in the new worktree."`
}

func (c *newCmd) Run(root Root, profile *config.Profile, store *state.Store) error {
	repo := string(root)

	if err := profile.Validate(); err != nil {
		return Exitf(ExitUsage, "%v", err)
	}

	v, err := prepareWorktree(repo, c.Branch, profile, store)
	if err != nil {
		return err
	}

	if v.Port, err = allocatePort(profile, v.SiteName, store); err != nil {
		return err
	}

	if err = gitx.WorktreeAdd(repo, v.Worktree, c.Branch, true); err != nil {
		return Exitf(ExitConflict, "%v", err)
	}

	rec := provisioningRecord(v, c.Branch, gitx.CurrentBranch(repo))
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
