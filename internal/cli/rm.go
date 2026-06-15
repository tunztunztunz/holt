package cli

import (
	"github.com/tunztunztunz/holt/internal/config"
	"github.com/tunztunztunz/holt/internal/gitx"
	"github.com/tunztunztunz/holt/internal/guard"
	"github.com/tunztunztunz/holt/internal/provision"
	"github.com/tunztunztunz/holt/internal/state"
	"github.com/tunztunztunz/holt/internal/vars"
)

// RmCmd removes a worktree: it runs the configured guards (which warn + confirm,
// or abort unless --force), runs teardown commands, removes the worktree and its
// branch, and drops it from state. If the user is standing in the tree being
// removed, it prints the repo root so the holt() shell wrapper cds them home.
type RmCmd struct {
	Name string `arg:"" optional:"" help:"Worktree to remove. Omit to pick from all; 'here' = the current tree."`
}

func (c *RmCmd) Run(root Root, profile *config.Profile, store *state.Store, g *Globals) error {
	repo := string(root)

	rec, err := pickTarget(c.Name, store, "Select a worktree to remove")
	if err != nil {
		return err
	}

	inside := isInside(rec.Path)

	// Guards compare against the tree's own base: its recorded fork point, else
	// the configured base, else the repo default. Using the repo default outright
	// would report "not in main" for a tree forked off another branch.
	base := rec.BaseBranch
	if base == "" {
		base = profile.Base
	}
	if base == "" {
		base = gitx.DefaultBranch()
	}

	// Guards
	for _, name := range profile.Guards {
		gd, ok := guard.Registry[name]
		if !ok {
			continue
		}
		reason, err := gd.Check(rec.Path, rec.Branch, base)
		if err != nil {
			return Exitf(ExitRuntime, "%v", err)
		}
		if reason != "" && !g.Force {
			warnf("%s in %s", reason, rec.SiteName)
			if !confirm("remove anyway?") {
				return Exitf(ExitGuard, "aborted by guard: %s", name)
			}
		}
	}

	if g.DryRun {
		infof("would remove %s (branch %s, port %d)", rec.SiteName, rec.Branch, rec.Port)
		return nil
	}

	if v, err := vars.Resolve(repo, rec.Branch, profile); err == nil {
		v.Port = rec.Port
		env := v.Environ()
		for _, line := range profile.Teardown {
			l, _ := v.Expand(line)
			if err = provision.RunCommand(rec.Path, l, env); err != nil {
				warnf("teardown %q: %v", l, err)
			}
		}
	}

	if err := gitx.WorktreeRemove(repo, rec.Path, true); err != nil {
		return Exitf(ExitRuntime, "%v", err)
	}
	_ = gitx.BranchDelete(repo, rec.Branch)
	_ = gitx.WorktreePrune(repo)

	delete(store.Worktrees, rec.SiteName)
	if err := state.Save(repo, store); err != nil {
		return Exitf(ExitRuntime, "%v", err)
	}

	// If we removed the tree the user was standing in, print the repo root so the
	// holt() function cds them home; otherwise stdout stays empty.
	if inside {
		resultf("%s\n", repo)
	} else {
		infof("removed %s", rec.SiteName)
	}
	return nil
}
