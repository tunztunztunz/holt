package cli

import (
	"github.com/tunztunztunz/acre/internal/config"
	"github.com/tunztunztunz/acre/internal/gitx"
	"github.com/tunztunztunz/acre/internal/guard"
	"github.com/tunztunztunz/acre/internal/provision"
	"github.com/tunztunztunz/acre/internal/state"
	"github.com/tunztunztunz/acre/internal/vars"
)

type RmCmd struct {
	Name string `arg:"" optional:"" help:"Worktree to remove. Omit to pick from all; 'here' = the current tree."`
}

func (c *RmCmd) Run(root Root, profile *config.Profile, store *state.Store, g *Globals) error {
	repo := string(root)

	rec, err := pickTarget(c.Name, store, "Select a worktree to remove")
	if err != nil {
		return err
	}

	// If we're standing inside the target we can't delete it out from under the
	// shell. Instead, we can print the repo root at the end and let the acre() shell
	// cd us home (same trick as cd).
	inside := isInside(rec.Path)

	def := gitx.DefaultBranch()

	// Guards
	for _, name := range profile.Guards {
		gd, ok := guard.Registry[name]
		if !ok {
			continue
		}
		reason, err := gd.Check(rec.Path, rec.Branch, def)
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
	// acre() function cds them home; otherwise stdout stays empty.
	if inside {
		resultf("%s\n", repo)
	} else {
		infof("removed %s", rec.SiteName)
	}
	return nil
}
