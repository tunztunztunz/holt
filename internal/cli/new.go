package cli

import (
	"path/filepath"
	"time"

	"github.com/tunztunztunz/acre/internal/config"
	"github.com/tunztunztunz/acre/internal/gitx"
	"github.com/tunztunztunz/acre/internal/provision"
	"github.com/tunztunztunz/acre/internal/state"
	"github.com/tunztunztunz/acre/internal/vars"
)

type newCmd struct {
	Branch string `arg:"" help:"Branch to create and check out in the new worktree."`
}

func (c *newCmd) Run(root Root, profile *config.Profile, store *state.Store) error {
	repo := string(root)

	if err := profile.Validate(); err != nil {
		return Exitf(ExitUsage, "%v", err)
	}

	v, err := vars.Resolve(repo, c.Branch, profile)
	if err != nil {
		return Exitf(ExitUsage, "%v", err)
	}

	if _, ok := store.Worktrees[v.SiteName]; ok {
		return Exitf(ExitConflict, "worktree %s already exists", v.SiteName)
	}

	taken := make(map[int]bool)
	for _, t := range store.Worktrees {
		taken[t.Port] = true
	}

	port, err := provision.AllocatePort(profile.Port, v.SiteName, taken)
	if err != nil {
		return Exitf(ExitUsage, "%v", err)
	}
	v.Port = port

	if err := gitx.WorktreeAdd(repo, v.Worktree, c.Branch, true); err != nil {
		return Exitf(ExitConflict, "%v", err)
	}

	rec := &state.Record{
		SiteName:     v.SiteName,
		Branch:       c.Branch,
		BaseBranch:   gitx.CurrentBranch(repo),
		Path:         v.Worktree,
		Port:         port,
		Status:       "provisioning",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	store.Worktrees[v.SiteName] = rec

	if err = state.Save(repo, store); err != nil {
		return Exitf(ExitRuntime, "%v", err)
	}

	if err := provisionWorktree(repo, v, profile); err != nil {
		rec.Status = "broken"
		_ = state.Save(repo, store)
		return Exitf(ExitRuntime, "%v", err)
	}

	rec.Status = "ready"
	rec.LastActivity = time.Now()
	if err := state.Save(repo, store); err != nil {
		return Exitf(ExitRuntime, "%v", err)
	}

	resultf("%s\n", v.Worktree)
	return nil
}

// provisionWorktree runs the copy, env, and setup steps against an already
// created worktree. It returns the first error encountered, leaving the
// caller to decide how to record the failure.
func provisionWorktree(root string, v *vars.Vars, p *config.Profile) error {
	for _, entry := range p.Copy {
		e, err := v.Expand(entry)
		if err != nil {
			return err
		}

		skip, err := provision.Copy(filepath.Join(root, e), filepath.Join(v.Worktree, e))
		if err != nil {
			return err
		}
		if skip != "" {
			warnf("copy %s: %s", e, skip)
		}
	}

	for _, block := range p.Env {
		if err := provision.RenderEnv(v.Worktree, block, v.Expand); err != nil {
			return err
		}
	}

	env := v.Environ()
	for _, line := range p.Setup {
		l, err := v.Expand(line)
		if err != nil {
			return err
		}
		if err := provision.RunCommand(v.Worktree, l, env); err != nil {
			return err
		}
	}

	return nil
}
